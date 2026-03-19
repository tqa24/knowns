package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
)

// EstimateTokens returns a rough token count (~4 chars per token for English).
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// markdownHeading represents a heading extracted from markdown content.
type markdownHeading struct {
	Level      int
	Title      string
	Content    string
	StartIndex int
	EndIndex   int
}

var headingRE = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// extractHeadings splits markdown into heading sections.
func extractHeadings(markdown string) []markdownHeading {
	lines := strings.Split(markdown, "\n")
	var headings []markdownHeading
	var current *markdownHeading

	for i, line := range lines {
		m := headingRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		// Close previous heading.
		if current != nil {
			endIdx := 0
			for j := 0; j < i; j++ {
				endIdx += len(lines[j]) + 1
			}
			current.Content = strings.TrimSpace(markdown[current.StartIndex:endIdx])
			current.EndIndex = endIdx
			headings = append(headings, *current)
		}

		level := len(m[1])
		title := strings.TrimSpace(m[2])

		// Compute content start (after this heading line).
		startIdx := 0
		for j := 0; j <= i; j++ {
			startIdx += len(lines[j]) + 1
		}
		if startIdx > len(markdown) {
			startIdx = len(markdown)
		}

		current = &markdownHeading{
			Level:      level,
			Title:      title,
			StartIndex: startIdx,
			EndIndex:   len(markdown),
		}
	}

	// Close last heading.
	if current != nil {
		current.Content = strings.TrimSpace(markdown[current.StartIndex:])
		current.EndIndex = len(markdown)
		headings = append(headings, *current)
	}

	return headings
}

// ChunkDocument splits a document into chunks by headings (port of TS chunkDocument).
func ChunkDocument(content string, path, title, description string, maxTokens int) ChunkResult {
	if maxTokens <= 0 {
		maxTokens = 512
	}

	var chunks []Chunk
	totalTokens := 0

	// Chunk 0: metadata (title + description).
	metaContent := title
	if description != "" {
		metaContent = title + "\n\n" + description
	}
	metaTokens := EstimateTokens(metaContent)
	chunks = append(chunks, Chunk{
		ID:           fmt.Sprintf("doc:%s:chunk:0", path),
		Type:         ChunkTypeDoc,
		DocPath:      path,
		Section:      "# Metadata",
		Content:      metaContent,
		TokenCount:   metaTokens,
		HeadingLevel: 1,
		Position:     0,
	})
	totalTokens += metaTokens

	// Extract headings and chunk by section.
	headings := extractHeadings(content)
	position := 1
	parentSection := ""

	for _, h := range headings {
		if h.Level == 1 {
			continue // skip h1 (usually doc title, already in metadata)
		}

		sectionTitle := fmt.Sprintf("%s %s", strings.Repeat("#", h.Level), h.Title)
		sectionContent := sectionTitle + "\n\n" + h.Content
		tokenCount := EstimateTokens(sectionContent)

		if tokenCount > maxTokens {
			// Split by paragraphs.
			paragraphs := splitParagraphs(h.Content)
			currentContent := sectionTitle
			currentTokens := EstimateTokens(sectionTitle)

			for _, para := range paragraphs {
				paraTokens := EstimateTokens(para)

				if currentTokens+paraTokens > maxTokens && currentContent != sectionTitle {
					// Save current chunk.
					ps := ""
					if h.Level > 2 {
						ps = parentSection
					}
					chunks = append(chunks, Chunk{
						ID:            fmt.Sprintf("doc:%s:chunk:%d", path, position),
						Type:          ChunkTypeDoc,
						DocPath:       path,
						Section:       sectionTitle,
						Content:       currentContent,
						TokenCount:    currentTokens,
						HeadingLevel:  h.Level,
						ParentSection: ps,
						Position:      position,
					})
					totalTokens += currentTokens
					position++

					// Start continuation chunk.
					currentContent = sectionTitle + " (continued)\n\n" + para
					currentTokens = EstimateTokens(currentContent)
				} else {
					currentContent += "\n\n" + para
					currentTokens += paraTokens
				}
			}

			// Remaining content.
			if currentContent != sectionTitle {
				ps := ""
				if h.Level > 2 {
					ps = parentSection
				}
				chunks = append(chunks, Chunk{
					ID:            fmt.Sprintf("doc:%s:chunk:%d", path, position),
					Type:          ChunkTypeDoc,
					DocPath:       path,
					Section:       sectionTitle,
					Content:       currentContent,
					TokenCount:    currentTokens,
					HeadingLevel:  h.Level,
					ParentSection: ps,
					Position:      position,
				})
				totalTokens += currentTokens
				position++
			}
		} else {
			// Fits in one chunk.
			ps := ""
			if h.Level > 2 {
				ps = parentSection
			}
			chunks = append(chunks, Chunk{
				ID:            fmt.Sprintf("doc:%s:chunk:%d", path, position),
				Type:          ChunkTypeDoc,
				DocPath:       path,
				Section:       sectionTitle,
				Content:       sectionContent,
				TokenCount:    tokenCount,
				HeadingLevel:  h.Level,
				ParentSection: ps,
				Position:      position,
			})
			totalTokens += tokenCount
			position++
		}

		if h.Level == 2 {
			parentSection = sectionTitle
		}
	}

	// If no headings, treat entire content as one chunk.
	if len(headings) == 0 && strings.TrimSpace(content) != "" {
		ct := EstimateTokens(content)
		chunks = append(chunks, Chunk{
			ID:           fmt.Sprintf("doc:%s:chunk:1", path),
			Type:         ChunkTypeDoc,
			DocPath:      path,
			Section:      "# Content",
			Content:      content,
			TokenCount:   ct,
			HeadingLevel: 1,
			Position:     1,
		})
		totalTokens += ct
	}

	return ChunkResult{Chunks: chunks, TotalTokens: totalTokens}
}

// ChunkTask splits a task into chunks by field (port of TS chunkTask).
func ChunkTask(task *models.Task) ChunkResult {
	var chunks []Chunk
	totalTokens := 0

	type fieldSpec struct {
		Name    string
		Content string
	}

	fields := []fieldSpec{
		{"description", taskFieldContent(task, "description")},
		{"ac", taskFieldContent(task, "ac")},
		{"plan", taskFieldContent(task, "plan")},
		{"notes", taskFieldContent(task, "notes")},
	}

	for _, f := range fields {
		if f.Content == "" {
			continue
		}
		tc := EstimateTokens(f.Content)
		chunks = append(chunks, Chunk{
			ID:         fmt.Sprintf("task:%s:chunk:%s", task.ID, f.Name),
			Type:       ChunkTypeTask,
			TaskID:     task.ID,
			Field:      f.Name,
			Content:    f.Content,
			TokenCount: tc,
			Status:     task.Status,
			Priority:   task.Priority,
			Labels:     task.Labels,
		})
		totalTokens += tc
	}

	return ChunkResult{Chunks: chunks, TotalTokens: totalTokens}
}

// taskFieldContent extracts content for a specific field of a task.
func taskFieldContent(task *models.Task, field string) string {
	switch field {
	case "description":
		if task.Description != "" {
			return task.Title + "\n\n" + task.Description
		}
		return task.Title
	case "ac":
		if len(task.AcceptanceCriteria) == 0 {
			return ""
		}
		var parts []string
		for _, ac := range task.AcceptanceCriteria {
			check := "[ ]"
			if ac.Completed {
				check = "[x]"
			}
			parts = append(parts, fmt.Sprintf("- %s %s", check, ac.Text))
		}
		return strings.Join(parts, "\n")
	case "plan":
		return task.ImplementationPlan
	case "notes":
		return task.ImplementationNotes
	}
	return ""
}

var paragraphSplitRE = regexp.MustCompile(`\n\n+`)

func splitParagraphs(text string) []string {
	parts := paragraphSplitRE.Split(text, -1)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
