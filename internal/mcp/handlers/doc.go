package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterDocTools registers all documentation-related MCP tools.
func RegisterDocTools(s *server.MCPServer, getStore func() *storage.Store) {
	// list_docs
	s.AddTool(
		mcp.NewTool("list_docs",
			mcp.WithDescription("List all documentation files with optional tag filter."),
			mcp.WithString("tag",
				mcp.Description("Filter by tag"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			docs, err := store.Docs.List()
			if err != nil {
				return errFailed("list docs", err)
			}

			args := req.GetArguments()
			tagFilter, _ := stringArg(args, "tag")

			var filtered []*models.Doc
			for _, d := range docs {
				if tagFilter != "" && !containsString(d.Tags, tagFilter) {
					continue
				}
				// Don't include content in the list view.
				d.Content = ""
				filtered = append(filtered, d)
			}
			if filtered == nil {
				filtered = []*models.Doc{}
			}

			out, _ := json.MarshalIndent(filtered, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_doc
	s.AddTool(
		mcp.NewTool("get_doc",
			mcp.WithDescription("Get a documentation file by path. Smart mode auto-returns full content if small (<=2000 tokens), else returns stats and TOC."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Document path (e.g., 'readme', 'guides/setup')"),
			),
			mcp.WithBoolean("smart",
				mcp.Description("Smart mode: auto-return full content if small, else stats+TOC"),
			),
			mcp.WithBoolean("info",
				mcp.Description("Return document stats (size, tokens, headings) without content"),
			),
			mcp.WithBoolean("toc",
				mcp.Description("Return table of contents only (list of headings)"),
			),
			mcp.WithString("section",
				mcp.Description("Return specific section by heading title or number (e.g., '2. Overview' or '2')"),
			),
			mcp.WithString("line",
				mcp.Description("Return specific lines. Single line (e.g., '42') or range (e.g., '10-20')"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			path, err := req.RequireString("path")
			if err != nil {
				return errResult(err.Error())
			}

			doc, err := store.Docs.Get(path)
			if err != nil {
				return errNotFound("Doc", err)
			}

			args := req.GetArguments()
			smart := boolArg(args, "smart")
			info := boolArg(args, "info")
			tocOnly := boolArg(args, "toc")
			section, hasSection := stringArg(args, "section")
			lineParam, hasLine := stringArg(args, "line")

			contentLen := utf8.RuneCountInString(doc.Content)
			// Approximate token count: ~4 chars per token.
			approxTokens := contentLen / 4

			if info {
				headings := extractHeadings(doc.Content)
				result := map[string]any{
					"path":        doc.Path,
					"title":       doc.Title,
					"description": doc.Description,
					"tags":        doc.Tags,
					"size":        contentLen,
					"tokens":      approxTokens,
					"headings":    headings,
					"createdAt":   doc.CreatedAt,
					"updatedAt":   doc.UpdatedAt,
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if tocOnly {
				headings := extractHeadings(doc.Content)
				result := map[string]any{
					"path":     doc.Path,
					"title":    doc.Title,
					"headings": headings,
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if hasSection && section != "" {
				sectionContent := extractSection(doc.Content, section)
				result := map[string]any{
					"path":    doc.Path,
					"title":   doc.Title,
					"section": section,
					"content": sectionContent,
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if hasLine && lineParam != "" {
				lineContent, lineLabel, err := extractLines(doc.Content, lineParam)
				if err != nil {
					return errResult(err.Error())
				}
				result := map[string]any{
					"path":  doc.Path,
					"title": doc.Title,
					"lines": lineLabel,
					"content": lineContent,
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if smart {
				const smartThreshold = 2000
				if approxTokens <= smartThreshold {
					// Small doc: return full content.
					out, _ := json.MarshalIndent(doc, "", "  ")
					return mcp.NewToolResultText(string(out)), nil
				}
				// Large doc: return stats and TOC.
				headings := extractHeadings(doc.Content)
				result := map[string]any{
					"path":        doc.Path,
					"title":       doc.Title,
					"description": doc.Description,
					"tags":        doc.Tags,
					"size":        contentLen,
					"tokens":      approxTokens,
					"headings":    headings,
					"note":        "Document is large. Use 'section' parameter to read a specific section, or 'toc: true' to see the table of contents.",
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			out, _ := json.MarshalIndent(doc, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// create_doc
	s.AddTool(
		mcp.NewTool("create_doc",
			mcp.WithDescription("Create a new documentation file."),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("Document title"),
			),
			mcp.WithString("description",
				mcp.Description("Document description"),
			),
			mcp.WithString("content",
				mcp.Description("Initial markdown content"),
			),
			mcp.WithArray("tags",
				mcp.Description("Document tags"),
				mcp.WithStringItems(),
			),
			mcp.WithString("folder",
				mcp.Description("Folder path (e.g., 'guides', 'patterns/auth')"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			title, err := req.RequireString("title")
			if err != nil {
				return errResult(err.Error())
			}

			args := req.GetArguments()

			// Build the doc path from title and optional folder.
			slug := slugify(title)
			folder, _ := stringArg(args, "folder")
			var docPath string
			if folder != "" {
				docPath = folder + "/" + slug
			} else {
				docPath = slug
			}

			doc := &models.Doc{
				Path:      docPath,
				Title:     title,
				Tags:      []string{},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			if v, ok := stringArg(args, "description"); ok {
				doc.Description = v
			}
			if v, ok := stringArg(args, "content"); ok {
				doc.Content = v
			}
			if v, ok := stringSliceArg(args, "tags"); ok {
				doc.Tags = v
			}

			if err := store.Docs.Create(doc); err != nil {
				return errFailed("create doc", err)
			}

			search.BestEffortIndexDoc(store, doc.Path)

			// Save initial version.
			_ = store.Versions.SaveDocVersion(doc.Path, models.DocVersion{
				Changes:  store.Versions.TrackDocChanges(nil, doc),
				Snapshot: storage.DocToSnapshot(doc),
			})

			// Notify server for real-time UI updates.
			go notifyDocUpdated(store, doc.Path)

			out, _ := json.MarshalIndent(doc, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// update_doc
	s.AddTool(
		mcp.NewTool("update_doc",
			mcp.WithDescription("Update an existing documentation file. Use 'section' with 'content' to replace only a specific section."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Document path (e.g., 'readme', 'guides/setup')"),
			),
			mcp.WithString("title",
				mcp.Description("New title"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("content",
				mcp.Description("Replace content (or section content if 'section' is specified)"),
			),
			mcp.WithArray("tags",
				mcp.Description("New tags"),
				mcp.WithStringItems(),
			),
			mcp.WithString("appendContent",
				mcp.Description("Append to existing content"),
			),
			mcp.WithString("section",
				mcp.Description("Target section to replace by heading title or number (use with 'content')"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			path, err := req.RequireString("path")
			if err != nil {
				return errResult(err.Error())
			}

			doc, err := store.Docs.Get(path)
			if err != nil {
				return errNotFound("Doc", err)
			}

			oldDoc := *doc // snapshot before changes

			args := req.GetArguments()

			if v, ok := stringArg(args, "title"); ok {
				doc.Title = v
			}
			if v, ok := stringArg(args, "description"); ok {
				doc.Description = v
			}
			if _, ok := args["tags"]; ok {
				if v, ok := stringSliceArg(args, "tags"); ok {
					doc.Tags = v
				} else {
					doc.Tags = []string{}
				}
			}

			sectionTarget, hasSection := stringArg(args, "section")
			newContent, hasContent := stringArg(args, "content")
			appendContent, hasAppend := stringArg(args, "appendContent")

			if hasSection && sectionTarget != "" && hasContent {
				// Replace a specific section.
				doc.Content = replaceSection(doc.Content, sectionTarget, newContent)
			} else if hasContent {
				doc.Content = newContent
			}

			if hasAppend && appendContent != "" {
				if doc.Content == "" {
					doc.Content = appendContent
				} else {
					if !strings.HasSuffix(doc.Content, "\n") {
						doc.Content += "\n"
					}
					doc.Content += appendContent
				}
			}

			doc.UpdatedAt = time.Now().UTC()

			if err := store.Docs.Update(doc); err != nil {
				return errFailed("update doc", err)
			}

			search.BestEffortIndexDoc(store, doc.Path)

			// Save version if something changed.
			changes := store.Versions.TrackDocChanges(&oldDoc, doc)
			if len(changes) > 0 {
				_ = store.Versions.SaveDocVersion(doc.Path, models.DocVersion{
					Changes:  changes,
					Snapshot: storage.DocToSnapshot(doc),
				})
			}

			// Notify server for real-time UI updates.
			go notifyDocUpdated(store, doc.Path)

			out, _ := json.MarshalIndent(doc, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_doc_history
	s.AddTool(
		mcp.NewTool("get_doc_history",
			mcp.WithDescription("Get the version history of a document, showing all changes over time."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Document path (e.g., 'readme', 'guides/setup')"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			path, err := req.RequireString("path")
			if err != nil {
				return errResult(err.Error())
			}

			history, err := store.Versions.GetDocHistory(path)
			if err != nil {
				return errFailed("get doc history", err)
			}

			out, _ := json.MarshalIndent(history, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// delete_doc
	s.AddTool(
		mcp.NewTool("delete_doc",
			mcp.WithDescription("Delete a documentation file permanently. Runs in dry-run mode by default (preview only). Set dryRun: false to actually delete."),
			mcp.WithString("path",
				mcp.Description("Document path (e.g., 'readme', 'guides/setup')"),
				mcp.Required(),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without deleting (default: true for safety)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			args := req.GetArguments()
			path, ok := stringArg(args, "path")
			if !ok || path == "" {
				return errResult(ErrPathReq)
			}

			// Default to dry-run for safety.
			dryRun := true
			if v, exists := args["dryRun"]; exists {
				if b, ok := v.(bool); ok {
					dryRun = b
				}
			}

			doc, err := store.Docs.Get(path)
			if err != nil {
				return errNotFound("Doc", err)
			}

			if dryRun {
				out, _ := json.MarshalIndent(map[string]any{
					"dryRun":  true,
					"message": fmt.Sprintf(MsgWouldDeleteDoc, doc.Path, doc.Title),
					"doc":     map[string]string{"path": doc.Path, "title": doc.Title, "description": doc.Description},
				}, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if err := store.Docs.Delete(path); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to delete doc: %s", err.Error())), nil
			}

			search.BestEffortRemoveDoc(store, path)

			// Notify server for real-time UI updates.
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(map[string]any{
				"deleted": true,
				"message": fmt.Sprintf(MsgDeletedDoc, doc.Path, doc.Title),
			}, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// boolArg safely extracts a bool from args; returns false if not present.
func boolArg(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// extractHeadings returns a list of headings found in the markdown content.
func extractHeadings(content string) []string {
	var headings []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "#") {
			headings = append(headings, strings.TrimSpace(line))
		}
	}
	return headings
}

// extractLines returns specific lines from content.
// lineParam can be "42" (single line) or "10-20" (range).
// Returns the extracted content, a human-readable label, and any error.
func extractLines(content, lineParam string) (string, string, error) {
	allLines := strings.Split(content, "\n")
	total := len(allLines)

	// Try range first: "10-20"
	if parts := strings.SplitN(lineParam, "-", 2); len(parts) == 2 {
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil && start >= 1 && end >= start {
			if start > total {
				return "", "", fmt.Errorf("line %d exceeds document length (%d lines)", start, total)
			}
			if end > total {
				end = total
			}
			extracted := allLines[start-1 : end]
			label := fmt.Sprintf("%d-%d", start, end)
			return strings.Join(extracted, "\n"), label, nil
		}
	}

	// Single line: "42"
	line, err := strconv.Atoi(lineParam)
	if err != nil {
		return "", "", fmt.Errorf("invalid line parameter: %q (use '42' or '10-20')", lineParam)
	}
	if line < 1 || line > total {
		return "", "", fmt.Errorf("line %d out of range (document has %d lines)", line, total)
	}
	return allLines[line-1], fmt.Sprintf("%d", line), nil
}

// extractSection finds the content of a specific heading section.
// The section parameter can be a heading title (with or without # prefix) or a number like "2".
func extractSection(content, section string) string {
	lines := strings.Split(content, "\n")

	// Determine if section is a number.
	sectionNum := 0
	if n, err := fmt.Sscanf(section, "%d", &sectionNum); n == 1 && err == nil {
		// Find the nth heading.
		headingCount := 0
		startLine := -1
		headingLevel := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "#") {
				headingCount++
				if headingCount == sectionNum {
					startLine = i
					// Count # chars to determine level.
					for _, c := range line {
						if c == '#' {
							headingLevel++
						} else {
							break
						}
					}
					break
				}
			}
		}
		if startLine == -1 {
			return ""
		}
		return extractSectionFromLine(lines, startLine, headingLevel)
	}

	// Search by title text.
	searchTitle := strings.TrimLeft(section, "# ")
	for i, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		lineTitle := strings.TrimLeft(line, "# ")
		if strings.EqualFold(strings.TrimSpace(lineTitle), strings.TrimSpace(searchTitle)) {
			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			return extractSectionFromLine(lines, i, level)
		}
	}
	return ""
}

// extractSectionFromLine extracts content from startLine until the next heading of equal or higher level.
func extractSectionFromLine(lines []string, startLine, level int) string {
	var result []string
	result = append(result, lines[startLine])
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#") {
			lineLevel := 0
			for _, c := range line {
				if c == '#' {
					lineLevel++
				} else {
					break
				}
			}
			if lineLevel <= level {
				break
			}
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// replaceSection replaces a section in the content with new content.
func replaceSection(content, sectionTarget, newContent string) string {
	lines := strings.Split(content, "\n")

	// Find the section.
	sectionNum := 0
	startLine := -1
	headingLevel := 0

	if n, err := fmt.Sscanf(sectionTarget, "%d", &sectionNum); n == 1 && err == nil {
		headingCount := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "#") {
				headingCount++
				if headingCount == sectionNum {
					startLine = i
					for _, c := range line {
						if c == '#' {
							headingLevel++
						} else {
							break
						}
					}
					break
				}
			}
		}
	} else {
		searchTitle := strings.TrimLeft(sectionTarget, "# ")
		for i, line := range lines {
			if !strings.HasPrefix(line, "#") {
				continue
			}
			lineTitle := strings.TrimLeft(line, "# ")
			if strings.EqualFold(strings.TrimSpace(lineTitle), strings.TrimSpace(searchTitle)) {
				startLine = i
				for _, c := range line {
					if c == '#' {
						headingLevel++
					} else {
						break
					}
				}
				break
			}
		}
	}

	if startLine == -1 {
		return content
	}

	// Find end of section.
	endLine := len(lines)
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#") {
			lineLevel := 0
			for _, c := range line {
				if c == '#' {
					lineLevel++
				} else {
					break
				}
			}
			if lineLevel <= headingLevel {
				endLine = i
				break
			}
		}
	}

	var result []string
	result = append(result, lines[:startLine]...)
	result = append(result, newContent)
	result = append(result, lines[endLine:]...)
	return strings.Join(result, "\n")
}

// slugify converts a title to a URL/path-safe slug.
func slugify(title string) string {
	title = strings.ToLower(title)
	var b strings.Builder
	prevHyphen := false
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if r == '-' || r == ' ' || r == '_' {
			if !prevHyphen {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
