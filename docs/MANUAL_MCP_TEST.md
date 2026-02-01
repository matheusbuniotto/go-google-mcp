# Manual MCP test

Run the MCP server via stdio and call tools one by one to verify behavior.

## Prerequisites

1. **Auth** (required for Google tools):
   ```bash
   go-google-mcp auth login --secrets path/to/client_secrets.json
   ```
   If you skip this, the server exits with "Auth error" and the test will hang on Initialize.

2. **Build the server** (optional but faster startup):
   ```bash
   go build -o go-google-mcp ./cmd/go-google-mcp
   ```

## Run the test

From the **repo root**:

```bash
go run ./cmd/mcp-test
```

The test will:

1. **Initialize** – handshake with the server  
2. **ListTools** – list all available tools  
3. **ping** – call `ping` with message `"hello from manual test"`  
4. **tasks_list_tasklists** – list task lists (max 5)  
5. **drive_search** – search Drive (empty query, limit 2)  
6. **drive_get_recent_activity** – recent activity (24h, limit 3)  
7. **drive_find_files** – fullText search for "readme" (limit 2)  

Server stderr (auth errors, logs) is forwarded to your terminal. If any tool fails (e.g. auth or API error), the test continues and prints the error.

## Expected output (success)

- Initialize: server name and version  
- ListTools: list of tool names  
- ping: `Pong: hello from manual test`  
- tasks_list_tasklists: list of task lists or "No task lists found."  
- drive_search: list of files or "No files found."  
- drive_get_recent_activity: activity lines or "No recent activity found."  
- drive_find_files: list of files or "No files found."  

## Troubleshooting

- **Initialize hangs** → Run `go-google-mcp auth login` and ensure token exists under `~/.go-google-mcp/`.  
- **"Auth error" on stderr** → Same as above.  
- **Tool returns error** → Check scope (e.g. Tasks, Drive) was granted during login; re-run login to add new scopes if needed.  
