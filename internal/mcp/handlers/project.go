package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterProjectTools registers project management MCP tools.
func RegisterProjectTools(
	s *server.MCPServer,
	getStore func() *storage.Store,
	setStore func(*storage.Store, string),
	getRoot func() string,
) {
	// detect_projects
	s.AddTool(
		mcp.NewTool("detect_projects",
			mcp.WithDescription("Scan common directories for Knowns projects (.knowns/ folders). Returns a list of detected project root paths."),
			mcp.WithArray("additionalPaths",
				mcp.Description("Additional directory paths to scan for projects"),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()

			// Default directories to scan.
			home, _ := os.UserHomeDir()
			dirs := []string{
				filepath.Join(home, "Desktop"),
				filepath.Join(home, "Documents"),
				filepath.Join(home, "Workspaces"),
				filepath.Join(home, "workspace"),
				filepath.Join(home, "projects"),
				filepath.Join(home, "Projects"),
				filepath.Join(home, "dev"),
				filepath.Join(home, "Dev"),
				"/tmp",
			}

			// Append any additional paths.
			if extra, ok := args["additionalPaths"]; ok {
				switch v := extra.(type) {
				case []any:
					for _, item := range v {
						if s, ok := item.(string); ok {
							dirs = append(dirs, s)
						}
					}
				case []string:
					dirs = append(dirs, v...)
				}
			}

			type projectInfo struct {
				ProjectRoot string `json:"projectRoot"`
				Name        string `json:"name,omitempty"`
			}

			seen := make(map[string]bool)
			var projects []projectInfo

			for _, dir := range dirs {
				if dir == "" {
					continue
				}
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					candidate := filepath.Join(dir, e.Name())
					knownsDir := filepath.Join(candidate, ".knowns")
					configFile := filepath.Join(knownsDir, "config.json")
					if _, err := os.Stat(configFile); err != nil {
						continue
					}
					if seen[candidate] {
						continue
					}
					seen[candidate] = true

					info := projectInfo{ProjectRoot: candidate}
					// Try to read project name.
					store := storage.NewStore(knownsDir)
					if proj, err := store.Config.Load(); err == nil {
						info.Name = proj.Name
					}
					projects = append(projects, info)
				}
			}

			out, _ := json.MarshalIndent(projects, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// set_project
	s.AddTool(
		mcp.NewTool("set_project",
			mcp.WithDescription("Set the active Knowns project. Must be called before any task, doc, or time operations."),
			mcp.WithString("projectRoot",
				mcp.Required(),
				mcp.Description("Absolute path to the project root directory (must contain a .knowns/ folder)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectRoot, err := req.RequireString("projectRoot")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			knownsDir := filepath.Join(projectRoot, ".knowns")
			if _, err := os.Stat(knownsDir); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("No .knowns/ directory found at %s", projectRoot)), nil
			}

			store := storage.NewStore(knownsDir)

			// Validate by loading config.
			proj, err := store.Config.Load()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to load project config: %s", err.Error())), nil
			}

			setStore(store, projectRoot)

			result := map[string]any{
				"success":     true,
				"projectRoot": projectRoot,
				"projectName": proj.Name,
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_current_project
	s.AddTool(
		mcp.NewTool("get_current_project",
			mcp.WithDescription("Get the currently active project path and verify it is valid."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			root := getRoot()
			store := getStore()

			if store == nil || root == "" {
				result := map[string]any{
					"projectRoot": nil,
					"valid":       false,
					"message":     "No project set. Call set_project first.",
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			result := map[string]any{
				"projectRoot": root,
				"valid":       true,
			}

			if proj, err := store.Config.Load(); err == nil {
				result["projectName"] = proj.Name
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
