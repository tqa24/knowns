package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"gopkg.in/yaml.v3"
)

// MemoryStore reads and writes memory files from .knowns/memory/ (project),
// .knowns/.working-memory/ (working), and ~/.knowns/memory/ (global).
type MemoryStore struct {
	root       string // .knowns/ directory (project)
	globalRoot string // ~/.knowns/ directory
}

func (ms *MemoryStore) projectDir() string { return filepath.Join(ms.root, "memory") }
func (ms *MemoryStore) workingDir() string { return filepath.Join(ms.root, ".working-memory") }
func (ms *MemoryStore) globalDir() string  { return filepath.Join(ms.globalRoot, "memory") }

// dirForLayer returns the directory for a given memory layer.
func (ms *MemoryStore) dirForLayer(layer string) (string, error) {
	switch layer {
	case models.MemoryLayerProject:
		return ms.projectDir(), nil
	case models.MemoryLayerWorking:
		return ms.workingDir(), nil
	case models.MemoryLayerGlobal:
		return ms.globalDir(), nil
	default:
		return "", fmt.Errorf("invalid memory layer: %q", layer)
	}
}

// memoryFrontmatter mirrors the YAML frontmatter in every memory file.
type memoryFrontmatter struct {
	ID        string            `yaml:"id"`
	Title     string            `yaml:"title"`
	Layer     string            `yaml:"layer"`
	Category  string            `yaml:"category,omitempty"`
	Tags      []string          `yaml:"tags,omitempty"`
	Metadata  map[string]string `yaml:"metadata,omitempty"`
	CreatedAt string            `yaml:"createdAt"`
	UpdatedAt string            `yaml:"updatedAt"`
}

// List returns memory entries, optionally filtered by layer.
// If layer is empty, returns entries from all layers.
func (ms *MemoryStore) List(layer string) ([]*models.MemoryEntry, error) {
	var entries []*models.MemoryEntry

	layers := []string{models.MemoryLayerWorking, models.MemoryLayerProject, models.MemoryLayerGlobal}
	if layer != "" {
		if !models.ValidMemoryLayer(layer) {
			return nil, fmt.Errorf("invalid memory layer: %q", layer)
		}
		layers = []string{layer}
	}

	for _, l := range layers {
		dir, _ := ms.dirForLayer(l)
		found, err := ms.listDir(dir, l)
		if err != nil {
			continue // non-fatal
		}
		entries = append(entries, found...)
	}

	return entries, nil
}

// listDir reads all memory files from a directory.
func (ms *MemoryStore) listDir(dir, layer string) ([]*models.MemoryEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listDir %s: %w", dir, err)
	}

	var entries []*models.MemoryEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if !strings.HasPrefix(e.Name(), "memory-") {
			continue
		}
		absPath := filepath.Join(dir, e.Name())
		entry, err := ms.parseFile(absPath, layer)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Get retrieves a memory entry by ID. Searches project, working, then global.
func (ms *MemoryStore) Get(id string) (*models.MemoryEntry, error) {
	filename := models.MemoryFileName(id)

	// Search order: project → working → global
	dirs := []struct {
		dir   string
		layer string
	}{
		{ms.projectDir(), models.MemoryLayerProject},
		{ms.workingDir(), models.MemoryLayerWorking},
		{ms.globalDir(), models.MemoryLayerGlobal},
	}

	for _, d := range dirs {
		absPath := filepath.Join(d.dir, filename)
		if _, err := os.Stat(absPath); err == nil {
			return ms.parseFile(absPath, d.layer)
		}
	}

	return nil, fmt.Errorf("memory %q not found", id)
}

// Create writes a new memory entry to the appropriate layer directory.
func (ms *MemoryStore) Create(entry *models.MemoryEntry) error {
	if entry.Layer == "" {
		entry.Layer = models.MemoryLayerProject
	}
	if !models.ValidMemoryLayer(entry.Layer) {
		return fmt.Errorf("invalid memory layer: %q", entry.Layer)
	}
	if entry.ID == "" {
		entry.ID = models.NewTaskID()
	}

	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = now
	}

	dir, err := ms.dirForLayer(entry.Layer)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create memory dir: %w", err)
	}

	absPath := filepath.Join(dir, models.MemoryFileName(entry.ID))
	return atomicWrite(absPath, []byte(renderMemory(entry)))
}

// Update overwrites an existing memory entry.
func (ms *MemoryStore) Update(entry *models.MemoryEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("memory ID is required")
	}

	// Find the existing file to determine its current location.
	existing, err := ms.Get(entry.ID)
	if err != nil {
		return err
	}

	entry.UpdatedAt = time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = existing.CreatedAt
	}

	dir, err := ms.dirForLayer(existing.Layer)
	if err != nil {
		return err
	}

	absPath := filepath.Join(dir, models.MemoryFileName(entry.ID))
	return atomicWrite(absPath, []byte(renderMemory(entry)))
}

// Delete removes a memory entry by ID.
func (ms *MemoryStore) Delete(id string) error {
	filename := models.MemoryFileName(id)

	dirs := []string{ms.projectDir(), ms.workingDir(), ms.globalDir()}
	for _, dir := range dirs {
		absPath := filepath.Join(dir, filename)
		if _, err := os.Stat(absPath); err == nil {
			return os.Remove(absPath)
		}
	}

	return fmt.Errorf("memory %q not found", id)
}

// Promote moves a memory entry up one layer (working→project→global).
func (ms *MemoryStore) Promote(id string) (*models.MemoryEntry, error) {
	entry, err := ms.Get(id)
	if err != nil {
		return nil, err
	}

	newLayer, ok := models.PromoteLayer(entry.Layer)
	if !ok {
		return nil, fmt.Errorf("cannot promote: already at top layer (%s)", entry.Layer)
	}

	return ms.moveLayer(entry, newLayer)
}

// Demote moves a memory entry down one layer (global→project→working).
func (ms *MemoryStore) Demote(id string) (*models.MemoryEntry, error) {
	entry, err := ms.Get(id)
	if err != nil {
		return nil, err
	}

	newLayer, ok := models.DemoteLayer(entry.Layer)
	if !ok {
		return nil, fmt.Errorf("cannot demote: already at bottom layer (%s)", entry.Layer)
	}

	return ms.moveLayer(entry, newLayer)
}

// moveLayer moves a memory entry from its current layer to a new layer.
func (ms *MemoryStore) moveLayer(entry *models.MemoryEntry, newLayer string) (*models.MemoryEntry, error) {
	oldDir, err := ms.dirForLayer(entry.Layer)
	if err != nil {
		return nil, err
	}
	newDir, err := ms.dirForLayer(newLayer)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(newDir, 0755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}

	filename := models.MemoryFileName(entry.ID)
	oldPath := filepath.Join(oldDir, filename)
	newPath := filepath.Join(newDir, filename)

	// Update layer and timestamp.
	entry.Layer = newLayer
	entry.UpdatedAt = time.Now().UTC()

	// Write to new location first, then remove old.
	if err := atomicWrite(newPath, []byte(renderMemory(entry))); err != nil {
		return nil, err
	}
	_ = os.Remove(oldPath)

	return entry, nil
}

// Clean removes all files from .knowns/.working-memory/.
func (ms *MemoryStore) Clean() (int, error) {
	dir := ms.workingDir()
	dirEntries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("clean working memory: %w", err)
	}

	count := 0
	for _, e := range dirEntries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
			count++
		}
	}
	return count, nil
}

// CountByLayer returns the number of entries in a given layer.
func (ms *MemoryStore) CountByLayer(layer string) (int, error) {
	dir, err := ms.dirForLayer(layer)
	if err != nil {
		return 0, err
	}
	dirEntries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range dirEntries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "memory-") && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count, nil
}

// parseFile reads and parses a single memory markdown file.
func (ms *MemoryStore) parseFile(absPath, layer string) (*models.MemoryEntry, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("parseFile %s: %w", absPath, err)
	}
	return parseMemoryContent(string(data), layer)
}

// parseMemoryContent parses the content of a memory markdown file.
func parseMemoryContent(content, layer string) (*models.MemoryEntry, error) {
	yamlBlock, body := splitFrontmatter(content)

	entry := &models.MemoryEntry{
		Layer:   layer,
		Content: strings.TrimSpace(body),
	}

	if yamlBlock == "" {
		return entry, nil
	}

	var fm memoryFrontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("parse memory frontmatter: %w", err)
	}

	entry.ID = fm.ID
	entry.Title = fm.Title
	entry.Category = fm.Category
	entry.Tags = fm.Tags
	if entry.Tags == nil {
		entry.Tags = []string{}
	}
	entry.Metadata = fm.Metadata
	entry.CreatedAt, _ = parseISO(fm.CreatedAt)
	entry.UpdatedAt, _ = parseISO(fm.UpdatedAt)

	// Layer from frontmatter takes precedence if set.
	if fm.Layer != "" {
		entry.Layer = fm.Layer
	}

	return entry, nil
}

// renderMemory produces the canonical markdown content for a memory file.
func renderMemory(entry *models.MemoryEntry) string {
	var b strings.Builder

	now := time.Now().UTC()
	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := entry.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}

	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", entry.ID)
	fmt.Fprintf(&b, "title: %s\n", yamlScalar(entry.Title))
	fmt.Fprintf(&b, "layer: %s\n", entry.Layer)

	if entry.Category != "" {
		fmt.Fprintf(&b, "category: %s\n", yamlScalar(entry.Category))
	}

	if len(entry.Tags) == 0 {
		b.WriteString("tags: []\n")
	} else {
		b.WriteString("tags:\n")
		for _, t := range entry.Tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}

	if len(entry.Metadata) > 0 {
		b.WriteString("metadata:\n")
		for k, v := range entry.Metadata {
			fmt.Fprintf(&b, "  %s: %s\n", k, yamlScalar(v))
		}
	}

	fmt.Fprintf(&b, "createdAt: '%s'\n", formatISO(createdAt))
	fmt.Fprintf(&b, "updatedAt: '%s'\n", formatISO(updatedAt))
	b.WriteString("---\n")

	if entry.Content != "" {
		b.WriteString("\n")
		b.WriteString(entry.Content)
		if !strings.HasSuffix(entry.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}
