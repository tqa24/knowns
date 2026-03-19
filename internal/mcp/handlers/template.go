package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTemplateTools registers all template-related MCP tools.
func RegisterTemplateTools(s *server.MCPServer, getStore func() *storage.Store) {
	// list_templates
	s.AddTool(
		mcp.NewTool("list_templates",
			mcp.WithDescription("List all available code generation templates."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			templates, err := store.Templates.List()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list templates: %s", err.Error())), nil
			}

			if templates == nil {
				templates = []*models.Template{}
			}

			out, _ := json.MarshalIndent(templates, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_template
	s.AddTool(
		mcp.NewTool("get_template",
			mcp.WithDescription("Get template configuration including prompts, files, and linked documentation."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Template name. Supports import prefix (e.g., 'knowns/component' for imported template)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			name, err := req.RequireString("name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			tmpl, err := store.Templates.Get(name)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Template not found: %s", err.Error())), nil
			}

			out, _ := json.MarshalIndent(tmpl, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// run_template
	s.AddTool(
		mcp.NewTool("run_template",
			mcp.WithDescription("Run a code generation template. By default runs in dry-run mode (preview only). Set dryRun: false to actually write files."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Template name to run"),
			),
			mcp.WithObject("variables",
				mcp.Description("Variables for the template (e.g., { name: 'MyComponent' })"),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without writing files (default: true for safety)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			name, err := req.RequireString("name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			args := req.GetArguments()

			// Default dryRun to true for safety.
			dryRun := true
			if v, ok := args["dryRun"]; ok {
				if b, ok := v.(bool); ok {
					dryRun = b
				}
			}

			// Extract variables.
			variables := make(map[string]string)
			if v, ok := args["variables"]; ok {
				switch m := v.(type) {
				case map[string]any:
					for k, val := range m {
						if s, ok := val.(string); ok {
							variables[k] = s
						} else {
							variables[k] = fmt.Sprintf("%v", val)
						}
					}
				}
			}

			tmpl, err := store.Templates.Get(name)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Template not found: %s", err.Error())), nil
			}

			if dryRun {
				// Return a preview of what would be generated.
				var files []string
				for _, action := range tmpl.Actions {
					if action.Path != "" {
						files = append(files, action.Path)
					}
				}

				result := map[string]any{
					"dryRun":    true,
					"template":  tmpl.Name,
					"variables": variables,
					"actions":   tmpl.Actions,
					"files":     files,
					"message":   "Dry run: no files were written. Set dryRun: false to generate files.",
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			// TODO: Actual template execution (Handlebars rendering) requires a
			// full template engine implementation. For now, return a stub.
			result := map[string]any{
				"success":   false,
				"template":  tmpl.Name,
				"variables": variables,
				"message":   "Template execution (non-dry-run) is not yet implemented in the Go MCP server. Use the TypeScript CLI for template generation.",
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// create_template
	s.AddTool(
		mcp.NewTool("create_template",
			mcp.WithDescription("Create a new code generation template scaffold."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Template name (will be folder name in .knowns/templates/)"),
			),
			mcp.WithString("description",
				mcp.Description("Template description"),
			),
			mcp.WithString("doc",
				mcp.Description("Link to documentation (e.g., 'patterns/my-pattern')"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			name, err := req.RequireString("name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			args := req.GetArguments()
			description, _ := stringArg(args, "description")

			if err := store.Templates.Create(name, description); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create template: %s", err.Error())), nil
			}

			// Load the created template to return its details.
			tmpl, err := store.Templates.Get(name)
			if err != nil {
				result := map[string]any{
					"success": true,
					"name":    name,
					"message": fmt.Sprintf("Template '%s' created successfully", name),
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			out, _ := json.MarshalIndent(tmpl, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
