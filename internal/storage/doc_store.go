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

// DocStore reads and writes doc files from .knowns/docs/ (and .knowns/imports/).
type DocStore struct {
	root string
}

func (ds *DocStore) docsDir() string    { return filepath.Join(ds.root, "docs") }
func (ds *DocStore) importsDir() string { return filepath.Join(ds.root, "imports") }

// docFrontmatter mirrors the YAML frontmatter in every doc file.
type docFrontmatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	CreatedAt   string   `yaml:"createdAt"`
	UpdatedAt   string   `yaml:"updatedAt"`
	Tags        []string `yaml:"tags"`
	Order       *int     `yaml:"order,omitempty"`
}

// List returns all docs from .knowns/docs/ and .knowns/imports/*/docs/.
func (ds *DocStore) List() ([]*models.Doc, error) {
	var docs []*models.Doc

	local, err := ds.walkDocs(ds.docsDir(), "", false, "")
	if err != nil {
		return nil, err
	}
	docs = append(docs, local...)

	imported, err := ds.listImported()
	if err != nil {
		// Non-fatal: return what we have so far.
		return docs, nil
	}
	docs = append(docs, imported...)

	return docs, nil
}

// listImported scans .knowns/imports/*/docs/ for additional docs.
func (ds *DocStore) listImported() ([]*models.Doc, error) {
	entries, err := os.ReadDir(ds.importsDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var docs []*models.Doc
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		importSource := e.Name()
		importDocsDir := filepath.Join(ds.importsDir(), importSource, "docs")
		imported, err := ds.walkDocs(importDocsDir, "", true, importSource)
		if err != nil {
			continue
		}
		docs = append(docs, imported...)
	}
	return docs, nil
}

// walkDocs recursively collects docs from a directory.
func (ds *DocStore) walkDocs(dir, relBase string, imported bool, importSource string) ([]*models.Doc, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("walkDocs %s: %w", dir, err)
	}
	var docs []*models.Doc
	for _, e := range entries {
		fullPath := filepath.Join(dir, e.Name())
		if e.IsDir() {
			subBase := e.Name()
			if relBase != "" {
				subBase = relBase + "/" + e.Name()
			}
			sub, err := ds.walkDocs(fullPath, subBase, imported, importSource)
			if err != nil {
				continue
			}
			docs = append(docs, sub...)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		baseName := strings.TrimSuffix(e.Name(), ".md")
		var relPath string
		if relBase == "" {
			relPath = baseName
		} else {
			relPath = relBase + "/" + baseName
		}
		folder := relBase
		doc, err := ds.parseFile(fullPath, relPath, folder, imported, importSource)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// Get retrieves a doc by its relative path (without .md extension).
// Examples: "readme", "patterns/module", "specs/user-auth"
func (ds *DocStore) Get(path string) (*models.Doc, error) {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".md")

	// Try local docs first.
	absPath := filepath.Join(ds.docsDir(), filepath.FromSlash(path)+".md")
	if _, err := os.Stat(absPath); err == nil {
		folder := filepath.ToSlash(filepath.Dir(filepath.FromSlash(path)))
		if folder == "." {
			folder = ""
		}
		return ds.parseFile(absPath, path, folder, false, "")
	}

	// Try imported docs: files are nested as .knowns/imports/{name}/docs/{name}/...
	entries, _ := os.ReadDir(ds.importsDir())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		importSource := e.Name()
		// Direct lookup: path already includes the source prefix.
		candidate := filepath.Join(ds.importsDir(), importSource, "docs", filepath.FromSlash(path)+".md")
		if _, err := os.Stat(candidate); err == nil {
			folder := filepath.ToSlash(filepath.Dir(filepath.FromSlash(path)))
			if folder == "." {
				folder = ""
			}
			return ds.parseFile(candidate, path, folder, true, importSource)
		}
		// Fallback: mentions inside imported docs use short paths without the
		// source prefix (e.g. @doc/patterns/foo instead of @doc/source/patterns/foo).
		// Try prepending the import source name.
		prefixed := importSource + "/" + path
		candidate = filepath.Join(ds.importsDir(), importSource, "docs", filepath.FromSlash(prefixed)+".md")
		if _, err := os.Stat(candidate); err == nil {
			folder := filepath.ToSlash(filepath.Dir(filepath.FromSlash(prefixed)))
			if folder == "." {
				folder = ""
			}
			return ds.parseFile(candidate, prefixed, folder, true, importSource)
		}
	}

	return nil, fmt.Errorf("doc %q not found", path)
}

// Create writes a new doc to .knowns/docs/{path}.md.
// doc.Path must be set (relative, without .md).
func (ds *DocStore) Create(doc *models.Doc) error {
	if doc.Path == "" {
		return fmt.Errorf("doc path is required")
	}
	absPath := filepath.Join(ds.docsDir(), filepath.FromSlash(doc.Path)+".md")
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("create doc dir: %w", err)
	}
	return ds.writeFile(absPath, doc)
}

// Update writes updated doc content.
func (ds *DocStore) Update(doc *models.Doc) error {
	if doc.Path == "" {
		return fmt.Errorf("doc path is required")
	}
	absPath := filepath.Join(ds.docsDir(), filepath.FromSlash(doc.Path)+".md")
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}
	return ds.writeFile(absPath, doc)
}

// Delete removes a doc file.
func (ds *DocStore) Delete(path string) error {
	path = strings.TrimSuffix(path, ".md")
	absPath := filepath.Join(ds.docsDir(), filepath.FromSlash(path)+".md")
	return os.Remove(absPath)
}

// parseFile reads and parses a single doc markdown file.
func (ds *DocStore) parseFile(absPath, relPath, folder string, imported bool, importSource string) (*models.Doc, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("parseFile %s: %w", absPath, err)
	}
	return parseDocContent(string(data), relPath, folder, imported, importSource)
}

// parseDocContent parses the content of a doc markdown file.
func parseDocContent(content, relPath, folder string, imported bool, importSource string) (*models.Doc, error) {
	yamlBlock, body := splitFrontmatter(content)

	doc := &models.Doc{
		Path:         relPath,
		Folder:       folder,
		IsImported:   imported,
		ImportSource: importSource,
		Content:      strings.TrimSpace(body),
	}

	if yamlBlock == "" {
		// No frontmatter: derive title from the path basename.
		doc.Title = filepath.Base(relPath)
		doc.Tags = []string{}
		return doc, nil
	}

	var fm docFrontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("parse doc frontmatter: %w", err)
	}

	doc.Title = fm.Title
	doc.Description = fm.Description
	doc.Tags = fm.Tags
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	doc.Order = fm.Order
	doc.CreatedAt, _ = parseISO(fm.CreatedAt)
	doc.UpdatedAt, _ = parseISO(fm.UpdatedAt)

	return doc, nil
}

// writeFile serialises a doc to the canonical markdown format.
func (ds *DocStore) writeFile(path string, doc *models.Doc) error {
	return atomicWrite(path, []byte(renderDoc(doc)))
}

// renderDoc produces the canonical markdown content for a doc file.
func renderDoc(doc *models.Doc) string {
	var b strings.Builder

	now := time.Now().UTC()
	createdAt := doc.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := doc.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}

	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", yamlScalar(doc.Title))
	fmt.Fprintf(&b, "description: %s\n", yamlScalar(doc.Description))
	fmt.Fprintf(&b, "createdAt: '%s'\n", formatISO(createdAt))
	fmt.Fprintf(&b, "updatedAt: '%s'\n", formatISO(updatedAt))

	if len(doc.Tags) == 0 {
		b.WriteString("tags: []\n")
	} else {
		b.WriteString("tags:\n")
		for _, t := range doc.Tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}

	if doc.Order != nil {
		fmt.Fprintf(&b, "order: %d\n", *doc.Order)
	}

	b.WriteString("---\n")

	if doc.Content != "" {
		b.WriteString("\n")
		b.WriteString(doc.Content)
		if !strings.HasSuffix(doc.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}
