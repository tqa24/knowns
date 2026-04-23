package models

import "time"

// Memory layer constants.
const (
	MemoryLayerProject = "project"
	MemoryLayerGlobal  = "global"
)

// MemoryEntry represents a single memory entry stored as a markdown file
// with YAML frontmatter. Content is free-form markdown.
type MemoryEntry struct {
	ID       string `json:"id"                    yaml:"id"`
	Title    string `json:"title"                 yaml:"title"`
	Layer    string `json:"layer"                 yaml:"layer"`                           // "project", "global"
	Category string `json:"category,omitempty"    yaml:"category,omitempty"`              // "pattern", "decision", "convention", "preference", etc.

	// Content holds the markdown body. Not persisted in frontmatter.
	Content string `json:"content,omitempty" yaml:"-"`

	Tags      []string          `json:"tags,omitempty"     yaml:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt"          yaml:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"          yaml:"updatedAt"`
}

// MemoryFileName returns the canonical file name for a memory entry.
//
// Format: "memory-{id}.md"
func MemoryFileName(id string) string {
	return "memory-" + id + ".md"
}

// ValidMemoryLayer reports whether layer is a recognised memory layer.
func ValidMemoryLayer(layer string) bool {
	return layer == MemoryLayerProject || layer == MemoryLayerGlobal
}

// ValidPersistentMemoryLayer reports whether layer is a persistent memory layer.
func ValidPersistentMemoryLayer(layer string) bool {
	return layer == MemoryLayerProject || layer == MemoryLayerGlobal
}

// PromoteLayer returns the next layer up, or an error string if already at top.
func PromoteLayer(layer string) (string, bool) {
	switch layer {
	case MemoryLayerProject:
		return MemoryLayerGlobal, true
	default:
		return "", false
	}
}

// DemoteLayer returns the next layer down, or an error string if already at bottom.
func DemoteLayer(layer string) (string, bool) {
	switch layer {
	case MemoryLayerGlobal:
		return MemoryLayerProject, true
	default:
		return "", false
	}
}

// PromotePersistentMemoryLayer returns the next persistent layer up.
func PromotePersistentMemoryLayer(layer string) (string, bool) {
	switch layer {
	case MemoryLayerProject:
		return MemoryLayerGlobal, true
	default:
		return "", false
	}
}

// DemotePersistentMemoryLayer returns the next persistent layer down.
func DemotePersistentMemoryLayer(layer string) (string, bool) {
	switch layer {
	case MemoryLayerGlobal:
		return MemoryLayerProject, true
	default:
		return "", false
	}
}
