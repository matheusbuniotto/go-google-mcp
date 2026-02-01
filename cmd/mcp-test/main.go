// Manual MCP test: runs the go-google-mcp server via stdio and calls tools one by one.
//
// Run from repo root:
//
//   go build -o go-google-mcp ./cmd/go-google-mcp   # build server (faster startup)
//   go run ./cmd/mcp-test
//
// Prerequisites:
//   - Run `go-google-mcp auth login --secrets path/to/client_secrets.json` first.
//     Otherwise the server exits with "Auth error" and the test will hang on Initialize.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// serverCommand returns (command, args) to run the MCP server. Prefers built binary in repo root.
func serverCommand() (string, []string) {
	if wd, err := os.Getwd(); err == nil {
		bin := filepath.Join(wd, "go-google-mcp")
		if info, err := os.Stat(bin); err == nil && !info.IsDir() {
			return bin, nil
		}
	}
	return "go", []string{"run", "./cmd/go-google-mcp"}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Prefer built binary for fast startup (run: go build -o go-google-mcp ./cmd/go-google-mcp)
	serverCmd, serverArgs := serverCommand()
	log.Printf("Starting MCP server: %s %s", serverCmd, strings.Join(serverArgs, " "))
	log.Println("(If Initialize hangs, run: go-google-mcp auth login --secrets path/to/client_secrets.json)")

	cli, err := client.NewStdioMCPClient(serverCmd, nil, serverArgs...)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	defer cli.Close()

	// Forward server stderr so we see auth errors and logs
	if stderr, ok := client.GetStderr(cli); ok {
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					os.Stderr.Write(buf[:n])
				}
				if err != nil {
					return
				}
			}
		}()
	}

	// 1. Initialize
	log.Println("--- Initialize ---")
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "mcp-test", Version: "0.1.0"}
	result, err := cli.Initialize(ctx, initReq)
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	log.Printf("Server: %s %s\n", result.ServerInfo.Name, result.ServerInfo.Version)

	// 2. List tools
	log.Println("--- ListTools ---")
	toolsRes, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("ListTools failed: %v", err)
	}
	log.Printf("Tools (%d): ", len(toolsRes.Tools))
	for i, t := range toolsRes.Tools {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(t.Name)
	}
	fmt.Println()

	// 3. Ping
	log.Println("--- CallTool: ping ---")
	pingRes, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "ping",
			Arguments: map[string]any{"message": "hello from manual test"},
		},
	})
	if err != nil {
		log.Printf("ping failed: %v", err)
	} else {
		log.Printf("ping result: %s", toolResultText(pingRes))
	}

	// 4. tasks_list_tasklists
	log.Println("--- CallTool: tasks_list_tasklists ---")
	taskListsRes, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "tasks_list_tasklists",
			Arguments: map[string]any{"max_results": 5},
		},
	})
	if err != nil {
		log.Printf("tasks_list_tasklists failed: %v", err)
	} else {
		log.Printf("tasks_list_tasklists result: %s", toolResultText(taskListsRes))
	}

	// 5. drive_search (empty query, small limit)
	log.Println("--- CallTool: drive_search ---")
	driveSearchRes, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "drive_search",
			Arguments: map[string]any{"limit": 2},
		},
	})
	if err != nil {
		log.Printf("drive_search failed: %v", err)
	} else {
		log.Printf("drive_search result: %s", toolResultText(driveSearchRes))
	}

	// 6. drive_get_recent_activity
	log.Println("--- CallTool: drive_get_recent_activity ---")
	activityRes, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "drive_get_recent_activity",
			Arguments: map[string]any{"hours": 24, "limit": 3},
		},
	})
	if err != nil {
		log.Printf("drive_get_recent_activity failed: %v", err)
	} else {
		log.Printf("drive_get_recent_activity result: %s", toolResultText(activityRes))
	}

	// 7. drive_find_files (needs search_term)
	log.Println("--- CallTool: drive_find_files ---")
	findRes, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "drive_find_files",
			Arguments: map[string]any{"search_term": "readme", "limit": 2},
		},
	})
	if err != nil {
		log.Printf("drive_find_files failed: %v", err)
	} else {
		log.Printf("drive_find_files result: %s", toolResultText(findRes))
	}

	log.Println("--- Manual test done ---")
}

func toolResultText(r *mcp.CallToolResult) string {
	if r == nil {
		return "(nil result)"
	}
	var parts []string
	if r.IsError {
		parts = append(parts, "[ERROR]")
	}
	for _, c := range r.Content {
		parts = append(parts, mcp.GetTextFromContent(c))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
