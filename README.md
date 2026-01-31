# go-google-mcp

[![Go Report Card](https://goreportcard.com/badge/github.com/matheusbuniotto/go-google-mcp)](https://goreportcard.com/report/github.com/matheusbuniotto/go-google-mcp)
[![CI](https://github.com/matheusbuniotto/go-google-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/matheusbuniotto/go-google-mcp/actions/workflows/ci.yml)

**Unified Model Context Protocol (MCP) server for Google Workspace.**

`go-google-mcp` is a powerful, secure, and extensible Go-based tool that allows AI agents (like Claude Desktop, Cursor, or Gemini CLI) to interact directly with your Google services through a standardized interface.

## 🚀 Features

Interact with Google Workspace using natural language through these integrated services:

- **📂 Google Drive**: Powerful search, read text content, create files/folders, update content, move, share, and trash.
- **📧 Gmail**: Search/list threads, read full conversations, create drafts, move to trash, and send emails.
- **📅 Google Calendar**: List upcoming events, create new meetings (with attendees), and delete events.
- **📊 Google Sheets**: Create spreadsheets, read ranges, append rows, and update specific cells.
- **📄 Google Docs**: Create new documents and read full document text.
- **👥 Google People**: List contacts and create new connections.
- **✅ Google Tasks**: List task lists and tasks, create, update, and delete tasks (with optional status/due filtering).

## 🛠 Installation

Ensure you have [Go](https://go.dev/doc/install) installed (version 1.24 or later recommended).

```bash
go install github.com/matheusbuniotto/go-google-mcp/cmd/go-google-mcp@latest
```

## 🔐 Authentication

This tool supports both **User OAuth 2.0** (best for personal/CLI use) and **Service Accounts** (best for server/automated use).

### Option 1: User OAuth (Recommended)

1.  **Create Credentials**: Go to the [Google Cloud Console](https://console.cloud.google.com/), enable the necessary APIs (Drive, Gmail, etc.), and create a **Desktop App** OAuth client.
2.  **Download JSON**: Save the client secrets file as `client_secrets.json`.
3.  **One-time Login**:
    ```bash
    go-google-mcp auth login --secrets path/to/client_secrets.json
    ```
    *This securely saves your token to `~/.go-google-mcp/`.*

### Option 2: Service Account

1.  Download your Service Account JSON key.
2.  Run with the `-creds` flag:
    ```bash
    go-google-mcp -creds path/to/service-account.json
    ```

## 🤖 Usage with AI Agents

### Claude Desktop / Cursor

Add the following to your `claude_desktop_config.json` (or your IDE's MCP settings):

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "go-google-mcp",
      "args": []
    }
  }
}
```

### Gemini CLI

```bash
gemini mcp add google-workspace $(which go-google-mcp)
```

## 🛠 Development

```bash
git clone https://github.com/matheusbuniotto/go-google-mcp.git
cd go-google-mcp
go build ./cmd/go-google-mcp
```

## 📜 License

MIT License. See [LICENSE](LICENSE) for details.
