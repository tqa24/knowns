---
title: MCP Server Pattern
createdAt: '2025-12-29T07:02:33.684Z'
updatedAt: '2026-03-08T18:21:23.560Z'
description: Documentation for the Model Context Protocol (MCP) server pattern
tags:
  - architecture
  - patterns
  - mcp
  - ai
---
## Overview

MCP (Model Context Protocol) is a protocol that allows AI models to interact with tools via JSON-RPC. Knowns implements an MCP server using the `mcp-go` library (`github.com/mark3labs/mcp-go`) so Claude and other AI agents can access tasks and documentation directly.
## Location

```
internal/mcp/
├── server.go              # MCP server setup, tool registration
└── handlers/
    ├── board.go           # Board/kanban tools
    ├── doc.go             # Document tools
    ├── project.go         # Project detection/selection
    ├── search.go          # Search tools
    ├── task.go            # Task CRUD tools
    ├── template.go        # Template/codegen tools
    ├── time.go            # Time tracking tools
    └── validate.go        # Validation tools
```
## Architecture

```
+------------------------------------------------------+
|                   Claude Desktop                      |
|                                                       |
|  "Work on task 42"                                   |
|         |                                            |
|         v                                            |
|  +----------------+                                  |
|  | MCP Client     |                                  |
|  | (built-in)     |                                  |
|  +-------+--------+                                  |
+----------|-------------------------------------------+
           | JSON-RPC over stdio
           |
+----------v-------------------------------------------+
|              Knowns MCP Server (mcp-go)               |
|  +------------------------------------------------+  |
|  |             Tool Definitions                    |  |
|  |  - create_task    - list_tasks                 |  |
|  |  - get_task       - update_task                |  |
|  |  - start_time     - stop_time                  |  |
|  |  - list_docs      - get_doc                    |  |
|  +------------------------------------------------+  |
|                        |                              |
|  +---------------------v--------------------------+  |
|  |             Handler Functions                   |  |
|  |  handleCreateTask(args) -> (result, error)     |  |
|  |  handleGetTask(args) -> (result, error)        |  |
|  +------------------------------------------------+  |
|                        |                              |
|  +---------------------v--------------------------+  |
|  |             Store                               |  |
|  |  Read/Write .knowns/ files                     |  |
|  +------------------------------------------------+  |
+------------------------------------------------------+
```
## Key Components

### 1. Tool Definitions (mcp-go)

Tools are defined using `mcp.NewTool()` with JSON Schema for input validation:

```go
package mcp

import (
    "github.com/mark3labs/mcp-go/mcp"
)

var createTaskTool = mcp.NewTool("create_task",
    mcp.WithDescription("Create a new task"),
    mcp.WithString("title",
        mcp.Required(),
        mcp.Description("Task title"),
    ),
    mcp.WithString("description",
        mcp.Description("Task description"),
    ),
    mcp.WithString("status",
        mcp.Description("Task status"),
        mcp.Enum("todo", "in-progress", "in-review", "done"),
    ),
    mcp.WithString("priority",
        mcp.Description("Task priority"),
        mcp.Enum("low", "medium", "high"),
    ),
)

var getTaskTool = mcp.NewTool("get_task",
    mcp.WithDescription("Get task details by ID"),
    mcp.WithString("taskId",
        mcp.Required(),
        mcp.Description("Task ID to retrieve"),
    ),
)
```

### 2. Tool Registration

Tools and their handlers are registered on the MCP server:

```go
// internal/mcp/server.go
package mcp

import (
    "github.com/mark3labs/mcp-go/server"
)

func NewMCPServer(store *storage.Store) *server.MCPServer {
    s := server.NewMCPServer(
        "knowns-mcp",
        "1.0.0",
        server.WithToolCapabilities(true),
        server.WithResourceCapabilities(true, false),
    )

    // Register tools with handlers
    s.AddTool(createTaskTool, handlers.HandleCreateTask(store))
    s.AddTool(getTaskTool, handlers.HandleGetTask(store))
    s.AddTool(listTasksTool, handlers.HandleListTasks(store))
    s.AddTool(updateTaskTool, handlers.HandleUpdateTask(store))
    // ... more tools

    return s
}
```

### 3. Tool Handlers

Each handler is a function that receives parsed arguments and returns a result:

```go
// internal/mcp/handlers/task.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
)

// HandleCreateTask returns a handler for creating tasks.
func HandleCreateTask(store *storage.Store) server.ToolHandlerFunc {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        title := request.Params.Arguments["title"].(string)
        description, _ := request.Params.Arguments["description"].(string)
        priority, _ := request.Params.Arguments["priority"].(string)

        task, err := store.CreateTask(storage.CreateTaskInput{
            Title:       title,
            Description: description,
            Priority:    priority,
        })
        if err != nil {
            return nil, fmt.Errorf("create task: %w", err)
        }

        data, _ := json.MarshalIndent(task, "", "  ")
        return mcp.NewToolResultText(string(data)), nil
    }
}

// HandleGetTask returns a handler for retrieving tasks.
func HandleGetTask(store *storage.Store) server.ToolHandlerFunc {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        taskID := request.Params.Arguments["taskId"].(string)

        task, err := store.GetTask(taskID)
        if err != nil {
            return nil, fmt.Errorf("task %s not found: %w", taskID, err)
        }

        data, _ := json.MarshalIndent(task, "", "  ")
        return mcp.NewToolResultText(string(data)), nil
    }
}
```

### 4. Resource Providers

MCP also allows exposing resources (docs) for AI to read:

```go
// List available resources
s.AddResourceTemplate(
    mcp.NewResourceTemplate(
        "knowns://docs/{path}",
        "Project documentation",
    ),
    handleReadDoc(store),
)

func handleReadDoc(store *storage.Store) server.ResourceTemplateHandlerFunc {
    return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
        path := extractPathFromURI(request.Params.URI)
        content, err := store.GetDocContent(path)
        if err != nil {
            return nil, err
        }
        return []mcp.ResourceContents{
            mcp.TextResourceContents(request.Params.URI, content, "text/markdown"),
        }, nil
    }
}
```

### 5. Stdio Transport

MCP server runs as a subprocess, communicating via stdin/stdout:

```go
// cmd/knowns/main.go (or internal/cli/mcp.go)
import (
    "github.com/mark3labs/mcp-go/server"
)

func runMCPServer(store *storage.Store) error {
    mcpServer := mcp.NewMCPServer(store)
    return server.ServeStdio(mcpServer)
}
```
## Tools Exposed

| Tool | Description | Input |
|------|-------------|-------|
| `create_task` | Create new task | title, description?, status?, priority?, labels? |
| `get_task` | Get task by ID | taskId |
| `list_tasks` | List tasks with filters | status?, assignee?, labels? |
| `update_task` | Update task fields | taskId, fields to update |
| `search` | Unified search (tasks + docs) | query, type?, mode?, filters |
| `start_time` | Start timer | taskId |
| `stop_time` | Stop current timer | - |
| `get_time_entries` | Get time entries | taskId?, dateRange? |
| `list_docs` | List all docs | folder? |
| `get_doc` | Get doc content | path |
| `create_doc` | Create new doc | title, content, tags? |
| `update_doc` | Update doc | path, content |

## Auto-Fetch Linked Docs

When Claude calls `get_task`, the server automatically fetches docs linked in the description:

```go
// fetchLinkedDocs extracts and resolves doc references from task descriptions.
func fetchLinkedDocs(store *storage.Store, description string) []DocContent {
    refs := extractDocReferences(description)
    // @doc/architecture/patterns/command -> .knowns/docs/architecture/patterns/command.md

    var docs []DocContent
    for _, ref := range refs {
        content, err := store.GetDocContent(ref.Path)
        if err == nil {
            docs = append(docs, DocContent{Name: ref.Path, Content: content})
        }
    }
    return docs
}
```
## Configuration

In Claude Desktop config:

```json
{
  "mcpServers": {
    "knowns": {
      "command": "knowns",
      "args": ["mcp"],
      "cwd": "/path/to/project"
    }
  }
}
```

## Benefits

1. **AI-Native**: Claude directly interacts with project data
2. **Type-Safe**: mcp-go validates inputs via JSON Schema
3. **Auto-Context**: Automatically fetches linked docs
4. **Extensible**: Easy to add new tools with `AddTool()`
5. **Standard Protocol**: Compatible with any MCP client
6. **Single binary**: No Node.js runtime required -- ships as a Go binary
## Adding New Tools

1. Define the tool with its schema:

```go
// internal/mcp/server.go (or a dedicated tools file)
var myTool = mcp.NewTool("my_tool",
    mcp.WithDescription("What this tool does"),
    mcp.WithString("param1",
        mcp.Required(),
        mcp.Description("First parameter"),
    ),
    mcp.WithNumber("param2",
        mcp.Description("Optional number parameter"),
    ),
)
```

2. Create the handler:

```go
// internal/mcp/handlers/my_handler.go
func HandleMyTool(store *storage.Store) server.ToolHandlerFunc {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        param1 := request.Params.Arguments["param1"].(string)
        // Implementation
        return mcp.NewToolResultText(result), nil
    }
}
```

3. Register in server setup:

```go
// internal/mcp/server.go
s.AddTool(myTool, handlers.HandleMyTool(store))
```
## Related Docs

- @doc/architecture/patterns/command - CLI Command Pattern
- @doc/architecture/patterns/storage - File-Based Storage Pattern
