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

// countTokens uses the tokenizer if available, otherwise falls back to EstimateTokens.
func countTokens(text string, tok Tokenizer) int {
	if tok == nil {
		return EstimateTokens(text)
	}
	out := tok.Encode(text, 999999)
	return len(out.InputIDs)
}

// markdownHeading represents a heading extracted from markdown content.
type markdownHeading struct {
	Level      int
	Title      string
	HeaderPath string // full hierarchy path e.g. "API/Endpoints/GET /users"
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
	inCodeBlock := false

	// Header stack for tracking full hierarchy path.
	type headerEntry struct {
		level int
		title string
	}
	var headerStack []headerEntry

	buildHeaderPath := func(stack []headerEntry) string {
		if len(stack) == 0 {
			return ""
		}
		parts := make([]string, len(stack))
		for i, e := range stack {
			parts[i] = e.title
		}
		return strings.Join(parts, "/")
	}

	for i, line := range lines {
		// Track fenced code blocks (triple-backtick toggle).
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Skip heading detection inside code blocks.
		if inCodeBlock {
			continue
		}

		m := headingRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		level := len(m[1])
		title := strings.TrimSpace(m[2])

		// Update header stack: pop entries of equal or higher level.
		for len(headerStack) > 0 && headerStack[len(headerStack)-1].level >= level {
			headerStack = headerStack[:len(headerStack)-1]
		}
		headerStack = append(headerStack, headerEntry{level: level, title: title})
		headerPath := buildHeaderPath(headerStack)

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
			HeaderPath: headerPath,
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
// If tok is non-nil, it is used for accurate token counting; otherwise EstimateTokens is used.
func ChunkDocument(content string, path, title, description string, maxTokens int, tok Tokenizer) ChunkResult {
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
	metaTokens := countTokens(metaContent, tok)
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

	// Capture H1 body content (between H1 and first H2) into metadata chunk.
	for _, h := range headings {
		if h.Level == 1 && strings.TrimSpace(h.Content) != "" {
			metaContent += "\n\n" + strings.TrimSpace(h.Content)
			break // only first H1
		}
	}
	// Update metadata chunk with any H1 body content.
	metaTokens = countTokens(metaContent, tok)
	chunks[0].Content = metaContent
	chunks[0].TokenCount = metaTokens

	for _, h := range headings {
		if h.Level == 1 {
			continue // skip h1 (title + body already in metadata chunk)
		}

		sectionTitle := fmt.Sprintf("%s %s", strings.Repeat("#", h.Level), h.Title)
		sectionContent := sectionTitle + "\n\n" + h.Content
		tokenCount := countTokens(sectionContent, tok)

		if tokenCount > maxTokens {
			// Split by paragraphs.
			paragraphs := splitParagraphs(h.Content)
			currentContent := sectionTitle
			currentTokens := countTokens(sectionTitle, tok)

			for _, para := range paragraphs {
				paraTokens := countTokens(para, tok)

				if currentTokens+paraTokens > maxTokens && currentContent != sectionTitle {
					// Save current chunk.
					chunks = append(chunks, Chunk{
						ID:           fmt.Sprintf("doc:%s:chunk:%d", path, position),
						Type:         ChunkTypeDoc,
						DocPath:      path,
						Section:      sectionTitle,
						Content:      currentContent,
						TokenCount:   currentTokens,
						HeadingLevel: h.Level,
						HeaderPath:   h.HeaderPath,
						Position:     position,
					})
					totalTokens += currentTokens
					position++

					// Start continuation chunk.
					currentContent = sectionTitle + " (continued)\n\n" + para
					currentTokens = countTokens(currentContent, tok)
				} else {
					currentContent += "\n\n" + para
					currentTokens += paraTokens
				}
			}

			// Remaining content.
			if currentContent != sectionTitle {
				chunks = append(chunks, Chunk{
					ID:           fmt.Sprintf("doc:%s:chunk:%d", path, position),
					Type:         ChunkTypeDoc,
					DocPath:      path,
					Section:      sectionTitle,
					Content:      currentContent,
					TokenCount:   currentTokens,
					HeadingLevel: h.Level,
					HeaderPath:   h.HeaderPath,
					Position:     position,
				})
				totalTokens += currentTokens
				position++
			}
		} else {
			// Fits in one chunk.
			chunks = append(chunks, Chunk{
				ID:           fmt.Sprintf("doc:%s:chunk:%d", path, position),
				Type:         ChunkTypeDoc,
				DocPath:      path,
				Section:      sectionTitle,
				Content:      sectionContent,
				TokenCount:   tokenCount,
				HeadingLevel: h.Level,
				HeaderPath:   h.HeaderPath,
				Position:     position,
			})
			totalTokens += tokenCount
			position++
		}
	}

	// If no headings, treat entire content as one chunk.
	if len(headings) == 0 && strings.TrimSpace(content) != "" {
		ct := countTokens(content, tok)
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
// If tok is non-nil, it is used for accurate token counting; otherwise EstimateTokens is used.
// Fields exceeding maxTokens are split by paragraph.
func ChunkTask(task *models.Task, maxTokens int, tok Tokenizer) ChunkResult {
	if maxTokens <= 0 {
		maxTokens = 512
	}

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
		tc := countTokens(f.Content, tok)

		if tc > maxTokens {
			// Split by paragraphs, matching ChunkDocument behavior.
			paragraphs := splitParagraphs(f.Content)
			partIdx := 0
			currentContent := ""
			currentTokens := 0

			for _, para := range paragraphs {
				paraTokens := countTokens(para, tok)

				if currentTokens+paraTokens > maxTokens && currentContent != "" {
					// Save current chunk.
					suffix := ""
					if partIdx > 0 {
						suffix = fmt.Sprintf(":%d", partIdx)
					}
					chunks = append(chunks, Chunk{
						ID:         fmt.Sprintf("task:%s:chunk:%s%s", task.ID, f.Name, suffix),
						Type:       ChunkTypeTask,
						TaskID:     task.ID,
						Field:      f.Name,
						Content:    currentContent,
						TokenCount: currentTokens,
						Status:     task.Status,
						Priority:   task.Priority,
						Labels:     task.Labels,
					})
					totalTokens += currentTokens
					partIdx++

					currentContent = para
					currentTokens = paraTokens
				} else {
					if currentContent != "" {
						currentContent += "\n\n" + para
					} else {
						currentContent = para
					}
					currentTokens += paraTokens
				}
			}

			// Remaining content.
			if currentContent != "" {
				suffix := ""
				if partIdx > 0 {
					suffix = fmt.Sprintf(":%d", partIdx)
				}
				chunks = append(chunks, Chunk{
					ID:         fmt.Sprintf("task:%s:chunk:%s%s", task.ID, f.Name, suffix),
					Type:       ChunkTypeTask,
					TaskID:     task.ID,
					Field:      f.Name,
					Content:    currentContent,
					TokenCount: currentTokens,
					Status:     task.Status,
					Priority:   task.Priority,
					Labels:     task.Labels,
				})
				totalTokens += currentTokens
			}
		} else {
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
