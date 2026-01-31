package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matheusbuniotto/go-google-mcp/pkg/auth"
	drivesvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/drive"
	gmailsvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/gmail"
	calendarsvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/calendar"
	sheetssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/sheets"
	peoplesvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/people"
	docssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/docs"
	taskssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/tasks"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/people/v1"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/tasks/v1"
)

func main() {
	// Subcommand parsing
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		handleAuthCommand()
		return
	}

	// Normal server mode
	credentialsFile := flag.String("creds", "", "Path to Google Service Account JSON file (optional)")
	flag.Parse()

	if *credentialsFile != "" {
		fmt.Fprintf(os.Stderr, "Using credentials file: %s\n", *credentialsFile)
	}

	// Initialize Auth
	// Using Drive, Gmail, Calendar, Sheets, People, Docs scopes
	scopes := []string{
		drive.DriveScope,
		gmail.GmailReadonlyScope,
		gmail.GmailSendScope,
		gmail.GmailModifyScope,
		calendar.CalendarScope,
		sheets.SpreadsheetsScope,
		people.ContactsScope,
		docs.DocumentsScope,
		tasks.TasksScope,
	}
	opts, err := auth.GetClientOptions(context.Background(), *credentialsFile, scopes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Auth error: %v\n", err)
		os.Exit(1)
	}

	// Initialize Drive Service
	driveService, err := drivesvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Drive service: %v\n", err)
		os.Exit(1)
	}

	// Initialize Gmail Service
	gmailService, err := gmailsvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Gmail service: %v\n", err)
		os.Exit(1)
	}

	// Initialize Calendar Service
	calendarService, err := calendarsvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Calendar service: %v\n", err)
		os.Exit(1)
	}

	// Initialize Sheets Service
	sheetsService, err := sheetssvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Sheets service: %v\n", err)
		os.Exit(1)
	}

	// Initialize People Service
	peopleService, err := peoplesvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create People service: %v\n", err)
		os.Exit(1)
	}

	// Initialize Docs Service
	docsService, err := docssvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Docs service: %v\n", err)
		os.Exit(1)
	}

	// Initialize Tasks Service
	tasksService, err := taskssvc.New(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Tasks service: %v\n", err)
		os.Exit(1)
	}

	// Initialize MCP Server
	s := server.NewMCPServer(
		"go-google-mcp",
		"0.1.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Tool: Ping
	s.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Ping the server to check availability"),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo back")),
	), pingHandler)

	// Tool: Drive Search
	s.AddTool(mcp.NewTool("drive_search",
		mcp.WithDescription("Search for files in Google Drive. Use raw 'query' (Drive query syntax) OR helper args. Use content_contains for fullText search; set include_snippet to get a short preview without reading the whole file."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of files to return (default 10)")),
		mcp.WithString("query", mcp.Description("Raw Google Drive query string (e.g. \"name contains 'foo'\")")),
		mcp.WithString("name_contains", mcp.Description("Filter by name containing this string")),
		mcp.WithString("content_contains", mcp.Description("Filter by content containing this string (fullText)")),
		mcp.WithString("mime_type", mcp.Description("Filter by exact mimeType (e.g. 'application/vnd.google-apps.folder')")),
		mcp.WithString("include_snippet", mcp.Description("If 'true', include a short content snippet per file when using content_contains (default: false)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int64(request.GetInt("limit", 10))
		rawQuery := request.GetString("query", "")
		nameContains := request.GetString("name_contains", "")
		contentContains := request.GetString("content_contains", "")
		mimeType := request.GetString("mime_type", "")
		includeSnippet := request.GetString("include_snippet", "false") == "true"

		var queryParts []string
		if rawQuery != "" {
			queryParts = append(queryParts, fmt.Sprintf("(%s)", rawQuery))
		}
		if nameContains != "" {
			queryParts = append(queryParts, fmt.Sprintf("name contains '%s'", nameContains))
		}
		if contentContains != "" {
			queryParts = append(queryParts, fmt.Sprintf("fullText contains '%s'", contentContains))
		}
		if mimeType != "" {
			queryParts = append(queryParts, fmt.Sprintf("mimeType = '%s'", mimeType))
		}

		finalQuery := strings.Join(queryParts, " and ")

		if includeSnippet && finalQuery != "" {
			results, err := driveService.SearchFilesWithSnippets(finalQuery, limit, 300)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to search files: %v", err)), nil
			}
			var result string
			for _, r := range results {
				result += fmt.Sprintf("[%s] %s (%s)\n", r.File.Id, r.File.Name, r.File.MimeType)
				if r.Snippet != "" {
					snip := strings.TrimSpace(r.Snippet)
					if len(snip) > 280 {
						snip = snip[:280] + "..."
					}
					result += fmt.Sprintf("  snippet: %s\n", snip)
				}
			}
			if len(results) == 0 {
				result = "No files found."
			}
			return mcp.NewToolResultText(result), nil
		}

		files, err := driveService.SearchFiles(finalQuery, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search files: %v", err)), nil
		}
		var result string
		for _, f := range files {
			result += fmt.Sprintf("[%s] %s (%s)\n", f.Id, f.Name, f.MimeType)
		}
		if len(files) == 0 {
			result = "No files found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Drive Find Files (account-wide discovery)
	s.AddTool(mcp.NewTool("drive_find_files",
		mcp.WithDescription("Find files across Google Drive by content (fullText search). Optimized for account-wide discovery when you know a phrase or keyword. Use drive_read_file to read a file's full content."),
		mcp.WithString("search_term", mcp.Required(), mcp.Description("Phrase or keyword to search for in file content")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of files to return (default 20)")),
		mcp.WithString("include_snippet", mcp.Description("If 'true', include a short content snippet per file (default: false)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		searchTerm, err := request.RequireString("search_term")
		if err != nil {
			return mcp.NewToolResultError("search_term is required"), nil
		}
		limit := int64(request.GetInt("limit", 20))
		includeSnippet := request.GetString("include_snippet", "false") == "true"

		if includeSnippet {
			results, err := driveService.FindFilesWithSnippets(searchTerm, limit, 300)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to find files: %v", err)), nil
			}
			var result string
			for _, r := range results {
				result += fmt.Sprintf("[%s] %s (%s)\n", r.File.Id, r.File.Name, r.File.MimeType)
				if r.Snippet != "" {
					snip := strings.TrimSpace(r.Snippet)
					if len(snip) > 280 {
						snip = snip[:280] + "..."
					}
					result += fmt.Sprintf("  snippet: %s\n", snip)
				}
			}
			if len(results) == 0 {
				result = "No files found."
			}
			return mcp.NewToolResultText(result), nil
		}

		files, err := driveService.FindFiles(searchTerm, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to find files: %v", err)), nil
		}
		var result string
		for _, f := range files {
			result += fmt.Sprintf("[%s] %s (%s)\n", f.Id, f.Name, f.MimeType)
		}
		if len(files) == 0 {
			result = "No files found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Drive Read File
	s.AddTool(mcp.NewTool("drive_read_file",
		mcp.WithDescription("Read the text content of a file from Google Drive. CAUTION: Only use for text-based files."),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("ID of the file to read")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileID, err := request.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError("file_id is required"), nil
		}

		// Limit to 32KB by default to avoid blowing up context
		content, err := driveService.ReadFileContent(fileID, 32*1024)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
		}

		return mcp.NewToolResultText(content), nil
	})

	// Tool: Drive Create File
	s.AddTool(mcp.NewTool("drive_create_file",
		mcp.WithDescription("Create a new text file in Google Drive"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the file")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Text content of the file")),
		mcp.WithString("parent_id", mcp.Description("ID of the parent folder (optional)")),
		mcp.WithString("mime_type", mcp.Description("MimeType (optional, default: text/plain)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("name is required"), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}
		parentID := request.GetString("parent_id", "")
		mimeType := request.GetString("mime_type", "text/plain")

		file, err := driveService.CreateFile(name, parentID, content, mimeType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create file: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created file: %s (ID: %s)", file.Name, file.Id)), nil
	})

	// Tool: Drive Create Folder
	s.AddTool(mcp.NewTool("drive_create_folder",
		mcp.WithDescription("Create a new folder in Google Drive"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the folder")),
		mcp.WithString("parent_id", mcp.Description("ID of the parent folder (optional)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("name is required"), nil
		}
		parentID := request.GetString("parent_id", "")

		folder, err := driveService.CreateFolder(name, parentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create folder: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created folder: %s (ID: %s)", folder.Name, folder.Id)), nil
	})

	// Tool: Drive Update File
	s.AddTool(mcp.NewTool("drive_update_file",
		mcp.WithDescription("Update a file's metadata or content in Google Drive"),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("ID of the file to update")),
		mcp.WithString("name", mcp.Description("New name (optional)")),
		mcp.WithString("content", mcp.Description("New text content (optional)")),
		mcp.WithString("add_parent_id", mcp.Description("Add this parent folder ID (optional, effectively moving/aliasing)")),
		mcp.WithString("remove_parent_id", mcp.Description("Remove this parent folder ID (optional)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileID, err := request.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError("file_id is required"), nil
		}
		name := request.GetString("name", "")
		content := request.GetString("content", "")
		addParent := request.GetString("add_parent_id", "")
		removeParent := request.GetString("remove_parent_id", "")
		
		var contentPtr *string
		if content != "" {
			contentPtr = &content
		}

		file, err := driveService.UpdateFile(fileID, name, addParent, removeParent, contentPtr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update file: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Updated file: %s (ID: %s)", file.Name, file.Id)), nil
	})

	// Tool: Drive Trash File
	s.AddTool(mcp.NewTool("drive_trash_file",
		mcp.WithDescription("Move a file or folder to trash (recoverable)"),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("ID of the file/folder to trash")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileID, err := request.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError("file_id is required"), nil
		}

		if err := driveService.TrashFile(fileID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trash file: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Trashed file: %s", fileID)), nil
	})

	// Tool: Drive Share File
	s.AddTool(mcp.NewTool("drive_share_file",
		mcp.WithDescription("Share a file/folder with a user"),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("ID of the file")),
		mcp.WithString("email", mcp.Required(), mcp.Description("Email address to share with")),
		mcp.WithString("role", mcp.Description("Role: 'reader', 'commenter', 'writer' (default: reader)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileID, err := request.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError("file_id is required"), nil
		}
		email, err := request.RequireString("email")
		if err != nil {
			return mcp.NewToolResultError("email is required"), nil
		}
		role := request.GetString("role", "reader")

		if err := driveService.AddPermission(fileID, role, "user", email); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to share file: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Shared file %s with %s as %s", fileID, email, role)), nil
	})

	// Tool: Gmail List Threads
	s.AddTool(mcp.NewTool("gmail_list_threads",
		mcp.WithDescription("List/Search email threads in Gmail"),
		mcp.WithString("query", mcp.Description("Gmail search query (e.g. 'from:boss', 'is:unread')")),
		mcp.WithNumber("limit", mcp.Description("Max threads to return (default 10)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := request.GetString("query", "")
		limit := int64(request.GetInt("limit", 10))

		threads, err := gmailService.ListThreads(query, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list threads: %v", err)), nil
		}

		var result string
		for _, t := range threads {
			result += fmt.Sprintf("[Thread ID: %s] %s\n", t.Id, t.Snippet)
		}
		if len(threads) == 0 {
			result = "No threads found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Gmail Read Thread
	s.AddTool(mcp.NewTool("gmail_read_thread",
		mcp.WithDescription("Read a specific email thread"),
		mcp.WithString("thread_id", mcp.Required(), mcp.Description("ID of the thread to read")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		threadID, err := request.RequireString("thread_id")
		if err != nil {
			return mcp.NewToolResultError("thread_id is required"), nil
		}

		thread, err := gmailService.GetThread(threadID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get thread: %v", err)), nil
		}

		var result string
		result += fmt.Sprintf("Thread ID: %s\n", thread.Id)
		for _, msg := range thread.Messages {
			subject := gmailsvc.GetHeader(msg.Payload.Headers, "Subject")
			from := gmailsvc.GetHeader(msg.Payload.Headers, "From")
			date := gmailsvc.GetHeader(msg.Payload.Headers, "Date")
			body := gmailsvc.ExtractMessageBody(msg.Payload)
			
			// Truncate body if too long for safety
			if len(body) > 2000 {
				body = body[:2000] + "...(truncated)"
			}

			result += fmt.Sprintf("---\nMsg ID: %s\nFrom: %s\nDate: %s\nSubject: %s\n\n%s\n", msg.Id, from, date, subject, body)
		}

		return mcp.NewToolResultText(result), nil
	})

	// Tool: Gmail Send Email
	s.AddTool(mcp.NewTool("gmail_send_email",
		mcp.WithDescription("Send an email"),
		mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address")),
		mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Email body content")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		to, err := request.RequireString("to")
		if err != nil {
			return mcp.NewToolResultError("to is required"), nil
		}
		subject, err := request.RequireString("subject")
		if err != nil {
			return mcp.NewToolResultError("subject is required"), nil
		}
		body, err := request.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError("body is required"), nil
		}

		msg, err := gmailService.SendEmail(to, subject, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send email: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Email sent! ID: %s", msg.Id)), nil
	})

	// Tool: Gmail Create Draft
	s.AddTool(mcp.NewTool("gmail_create_draft",
		mcp.WithDescription("Create a draft email"),
		mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address")),
		mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Email body content")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		to, err := request.RequireString("to")
		if err != nil {
			return mcp.NewToolResultError("to is required"), nil
		}
		subject, err := request.RequireString("subject")
		if err != nil {
			return mcp.NewToolResultError("subject is required"), nil
		}
		body, err := request.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError("body is required"), nil
		}

		draft, err := gmailService.CreateDraft(to, subject, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create draft: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Draft created! ID: %s", draft.Id)), nil
	})

	// Tool: Gmail Trash Thread
	s.AddTool(mcp.NewTool("gmail_trash_thread",
		mcp.WithDescription("Move an email thread to trash"),
		mcp.WithString("thread_id", mcp.Required(), mcp.Description("ID of the thread to trash")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		threadID, err := request.RequireString("thread_id")
		if err != nil {
			return mcp.NewToolResultError("thread_id is required"), nil
		}

		if err := gmailService.TrashThread(threadID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trash thread: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Thread %s moved to trash.", threadID)), nil
	})

	// Tool: Gmail List Labels
	s.AddTool(mcp.NewTool("gmail_list_labels",
		mcp.WithDescription("List all Gmail labels"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		labels, err := gmailService.ListLabels()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list labels: %v", err)), nil
		}

		var result string
		for _, l := range labels {
			result += fmt.Sprintf("ID: %s | Name: %s | Type: %s\n", l.Id, l.Name, l.Type)
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Calendar List Events
	s.AddTool(mcp.NewTool("calendar_list_events",
		mcp.WithDescription("List upcoming events from Google Calendar"),
		mcp.WithString("calendar_id", mcp.Description("Calendar ID (default: 'primary')")),
		mcp.WithNumber("max_results", mcp.Description("Max events to return (default 10)")),
		mcp.WithString("time_min", mcp.Description("Start time (RFC3339). Default: now.")),
		mcp.WithString("time_max", mcp.Description("End time (RFC3339). Optional.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		calendarID := request.GetString("calendar_id", "primary")
		maxResults := int64(request.GetInt("max_results", 10))
		timeMin := request.GetString("time_min", "")
		timeMax := request.GetString("time_max", "")

		events, err := calendarService.ListEvents(calendarID, maxResults, timeMin, timeMax)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list events: %v", err)), nil
		}

		var result string
		for _, e := range events {
			start := e.Start.DateTime
			if start == "" {
				start = e.Start.Date // All-day event
			}
			result += fmt.Sprintf("[%s] %s (%s)\n", start, e.Summary, e.Id)
		}
		if len(events) == 0 {
			result = "No upcoming events found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Calendar Create Event
	s.AddTool(mcp.NewTool("calendar_create_event",
		mcp.WithDescription("Create a new event in Google Calendar"),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Event title")),
		mcp.WithString("start_time", mcp.Required(), mcp.Description("Start time (RFC3339, e.g. '2025-01-31T10:00:00Z')")),
		mcp.WithString("end_time", mcp.Required(), mcp.Description("End time (RFC3339)")),
		mcp.WithString("description", mcp.Description("Event description")),
		mcp.WithString("attendees", mcp.Description("Comma-separated list of attendee emails")),
		mcp.WithString("calendar_id", mcp.Description("Calendar ID (default: 'primary')")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError("summary is required"), nil
		}
		startTime, err := request.RequireString("start_time")
		if err != nil {
			return mcp.NewToolResultError("start_time is required"), nil
		}
		endTime, err := request.RequireString("end_time")
		if err != nil {
			return mcp.NewToolResultError("end_time is required"), nil
		}
		description := request.GetString("description", "")
		attendeesStr := request.GetString("attendees", "")
		calendarID := request.GetString("calendar_id", "primary")

		var attendees []string
		if attendeesStr != "" {
			for _, a := range strings.Split(attendeesStr, ",") {
				attendees = append(attendees, strings.TrimSpace(a))
			}
		}

		event, err := calendarService.CreateEvent(calendarID, summary, description, startTime, endTime, attendees)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create event: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created event: %s (ID: %s)", event.Summary, event.Id)), nil
	})

	// Tool: Calendar Delete Event
	s.AddTool(mcp.NewTool("calendar_delete_event",
		mcp.WithDescription("Delete an event from Google Calendar"),
		mcp.WithString("event_id", mcp.Required(), mcp.Description("ID of the event to delete")),
		mcp.WithString("calendar_id", mcp.Description("Calendar ID (default: 'primary')")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		eventID, err := request.RequireString("event_id")
		if err != nil {
			return mcp.NewToolResultError("event_id is required"), nil
		}
		calendarID := request.GetString("calendar_id", "primary")

		if err := calendarService.DeleteEvent(calendarID, eventID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete event: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Deleted event: %s", eventID)), nil
	})

	// Tool: Sheets Create Spreadsheet
	s.AddTool(mcp.NewTool("sheets_create_spreadsheet",
		mcp.WithDescription("Create a new Google Sheet"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Title of the spreadsheet")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}

		sp, err := sheetsService.CreateSpreadsheet(title)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create spreadsheet: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created spreadsheet: %s (ID: %s)\nURL: %s", sp.Properties.Title, sp.SpreadsheetId, sp.SpreadsheetUrl)), nil
	})

	// Tool: Sheets Read Values
	s.AddTool(mcp.NewTool("sheets_read_values",
		mcp.WithDescription("Read values from a Google Sheet range"),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("ID of the spreadsheet")),
		mcp.WithString("range", mcp.Required(), mcp.Description("A1 notation range (e.g. 'Sheet1!A1:C10')")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		spreadsheetID, err := request.RequireString("spreadsheet_id")
		if err != nil {
			return mcp.NewToolResultError("spreadsheet_id is required"), nil
		}
		rangeName, err := request.RequireString("range")
		if err != nil {
			return mcp.NewToolResultError("range is required"), nil
		}

		values, err := sheetsService.ReadValues(spreadsheetID, rangeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read values: %v", err)), nil
		}

		if len(values) == 0 {
			return mcp.NewToolResultText("No data found."), nil
		}

		// JSON output is often best for structured data analysis by AI
		jsonBytes, _ := json.MarshalIndent(values, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// Tool: Sheets Append Values
	s.AddTool(mcp.NewTool("sheets_append_values",
		mcp.WithDescription("Append values to a Google Sheet (new rows)"),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("ID of the spreadsheet")),
		mcp.WithString("range", mcp.Required(), mcp.Description("A1 notation range (e.g. 'Sheet1!A1')")),
		mcp.WithString("values_json", mcp.Required(), mcp.Description("JSON array of arrays (e.g. '[[\"A\", \"B\"]]') or single array for one row")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		spreadsheetID, err := request.RequireString("spreadsheet_id")
		if err != nil {
			return mcp.NewToolResultError("spreadsheet_id is required"), nil
		}
		rangeName, err := request.RequireString("range")
		if err != nil {
			return mcp.NewToolResultError("range is required"), nil
		}
		valuesJSON, err := request.RequireString("values_json")
		if err != nil {
			return mcp.NewToolResultError("values_json is required"), nil
		}

		resp, err := sheetsService.AppendValues(spreadsheetID, rangeName, valuesJSON)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to append values: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Appended %d cells.", resp.Updates.UpdatedCells)), nil
	})

	// Tool: Sheets Update Values
	s.AddTool(mcp.NewTool("sheets_update_values",
		mcp.WithDescription("Update values in a Google Sheet range (overwrite)"),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("ID of the spreadsheet")),
		mcp.WithString("range", mcp.Required(), mcp.Description("A1 notation range (e.g. 'Sheet1!A1')")),
		mcp.WithString("values_json", mcp.Required(), mcp.Description("JSON array of arrays")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		spreadsheetID, err := request.RequireString("spreadsheet_id")
		if err != nil {
			return mcp.NewToolResultError("spreadsheet_id is required"), nil
		}
		rangeName, err := request.RequireString("range")
		if err != nil {
			return mcp.NewToolResultError("range is required"), nil
		}
		valuesJSON, err := request.RequireString("values_json")
		if err != nil {
			return mcp.NewToolResultError("values_json is required"), nil
		}

		resp, err := sheetsService.UpdateValues(spreadsheetID, rangeName, valuesJSON)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update values: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Updated %d cells.", resp.UpdatedCells)), nil
	})

	// Tool: People List Connections
	s.AddTool(mcp.NewTool("people_list_connections",
		mcp.WithDescription("List contacts (connections)"),
		mcp.WithNumber("limit", mcp.Description("Max contacts to return (default 10)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int64(request.GetInt("limit", 10))

		connections, err := peopleService.ListConnections(limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list connections: %v", err)), nil
		}

		var result string
		for _, p := range connections {
			name := "Unknown"
			if len(p.Names) > 0 {
				name = p.Names[0].DisplayName
			}
			email := ""
			if len(p.EmailAddresses) > 0 {
				email = p.EmailAddresses[0].Value
			}
			result += fmt.Sprintf("Name: %s | Email: %s | ResourceName: %s\n", name, email, p.ResourceName)
		}
		if len(connections) == 0 {
			result = "No connections found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: People Create Contact
	s.AddTool(mcp.NewTool("people_create_contact",
		mcp.WithDescription("Create a new contact"),
		mcp.WithString("given_name", mcp.Required(), mcp.Description("First name")),
		mcp.WithString("family_name", mcp.Description("Last name")),
		mcp.WithString("email", mcp.Description("Email address")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		givenName, err := request.RequireString("given_name")
		if err != nil {
			return mcp.NewToolResultError("given_name is required"), nil
		}
		familyName := request.GetString("family_name", "")
		email := request.GetString("email", "")

		person, err := peopleService.CreateContact(givenName, familyName, email)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create contact: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created contact: %s (ID: %s)", givenName, person.ResourceName)), nil
	})

	// Tool: Docs Create Document
	s.AddTool(mcp.NewTool("docs_create_document",
		mcp.WithDescription("Create a new Google Doc"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Document title")),
		mcp.WithString("initial_text", mcp.Description("Initial text content to insert")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}
		initialText := request.GetString("initial_text", "")

		doc, err := docsService.CreateDocument(title)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create document: %v", err)), nil
		}

		if initialText != "" {
			if err := docsService.InsertText(doc.DocumentId, initialText); err != nil {
				// We still return success for creation, but note the error
				return mcp.NewToolResultText(fmt.Sprintf("Created document: %s (ID: %s)\nWarning: Failed to insert initial text: %v", doc.Title, doc.DocumentId, err)), nil
			}
		}

		return mcp.NewToolResultText(fmt.Sprintf("Created document: %s (ID: %s)", doc.Title, doc.DocumentId)), nil
	})

	// Tool: Docs Read Document
	s.AddTool(mcp.NewTool("docs_read_document",
		mcp.WithDescription("Read a Google Doc"),
		mcp.WithString("document_id", mcp.Required(), mcp.Description("ID of the document")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docID, err := request.RequireString("document_id")
		if err != nil {
			return mcp.NewToolResultError("document_id is required"), nil
		}

		doc, err := docsService.GetDocument(docID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read document: %v", err)), nil
		}

		// Very basic text extraction
		var text string
		if doc.Body != nil {
			for _, elem := range doc.Body.Content {
				if elem.Paragraph != nil {
					for _, paraElem := range elem.Paragraph.Elements {
						if paraElem.TextRun != nil {
							text += paraElem.TextRun.Content
						}
					}
				}
			}
		}

		return mcp.NewToolResultText(fmt.Sprintf("Title: %s\n\n%s", doc.Title, text)), nil
	})

	// Tool: Tasks List Task Lists
	s.AddTool(mcp.NewTool("tasks_list_tasklists",
		mcp.WithDescription("List the user's Google Tasks task lists. Call this first to get task_list_id for other tasks operations."),
		mcp.WithNumber("max_results", mcp.Description("Max task lists to return (default 100)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		maxResults := int64(request.GetInt("max_results", 100))

		lists, err := tasksService.ListTaskLists(maxResults)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list task lists: %v", err)), nil
		}

		var result string
		for _, l := range lists {
			result += fmt.Sprintf("ID: %s | Title: %s\n", l.Id, l.Title)
		}
		if len(lists) == 0 {
			result = "No task lists found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Tasks List Tasks
	s.AddTool(mcp.NewTool("tasks_list_tasks",
		mcp.WithDescription("List tasks in a Google Tasks list. Use tasks_list_tasklists first to get task_list_id."),
		mcp.WithString("task_list_id", mcp.Required(), mcp.Description("ID of the task list")),
		mcp.WithString("show_completed", mcp.Description("Include completed tasks: 'true' or 'false' (default: false to reduce output)")),
		mcp.WithNumber("max_results", mcp.Description("Max tasks to return (default 20, max 100)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskListID, err := request.RequireString("task_list_id")
		if err != nil {
			return mcp.NewToolResultError("task_list_id is required"), nil
		}
		showCompleted := request.GetString("show_completed", "false") == "true"
		maxResults := int64(request.GetInt("max_results", 20))

		taskList, err := tasksService.ListTasks(taskListID, taskssvc.ListTasksOptions{
			ShowCompleted: showCompleted,
			MaxResults:    maxResults,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
		}

		var result string
		for _, t := range taskList {
			status := t.Status
			if status == "" {
				status = "needsAction"
			}
			due := ""
			if t.Due != "" {
				due = " | Due: " + t.Due
			}
			result += fmt.Sprintf("[%s] %s | Status: %s%s\n", t.Id, t.Title, status, due)
		}
		if len(taskList) == 0 {
			result = "No tasks found."
		}
		return mcp.NewToolResultText(result), nil
	})

	// Tool: Tasks Insert Task
	s.AddTool(mcp.NewTool("tasks_insert_task",
		mcp.WithDescription("Create a new task in a Google Tasks list"),
		mcp.WithString("task_list_id", mcp.Required(), mcp.Description("ID of the task list")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("notes", mcp.Description("Optional notes")),
		mcp.WithString("due", mcp.Description("Due date (RFC3339 date, e.g. 2025-02-01)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskListID, err := request.RequireString("task_list_id")
		if err != nil {
			return mcp.NewToolResultError("task_list_id is required"), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}
		notes := request.GetString("notes", "")
		due := request.GetString("due", "")

		task, err := tasksService.InsertTask(taskListID, title, notes, due)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to insert task: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created task: %s (ID: %s)", task.Title, task.Id)), nil
	})

	// Tool: Tasks Update Task
	s.AddTool(mcp.NewTool("tasks_update_task",
		mcp.WithDescription("Update an existing task (title, notes, due date, or status)"),
		mcp.WithString("task_list_id", mcp.Required(), mcp.Description("ID of the task list")),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("ID of the task")),
		mcp.WithString("title", mcp.Description("New title (optional)")),
		mcp.WithString("notes", mcp.Description("New notes (optional)")),
		mcp.WithString("due", mcp.Description("New due date RFC3339 (optional)")),
		mcp.WithString("status", mcp.Description("'needsAction' or 'completed' (optional)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskListID, err := request.RequireString("task_list_id")
		if err != nil {
			return mcp.NewToolResultError("task_list_id is required"), nil
		}
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		title := request.GetString("title", "")
		notes := request.GetString("notes", "")
		due := request.GetString("due", "")
		status := request.GetString("status", "")

		in := taskssvc.UpdateTaskInput{}
		if title != "" {
			in.Title = &title
		}
		if notes != "" {
			in.Notes = &notes
		}
		if due != "" {
			in.Due = &due
		}
		if status != "" {
			in.Status = &status
		}

		task, err := tasksService.UpdateTask(taskListID, taskID, in)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated task: %s (ID: %s)", task.Title, task.Id)), nil
	})

	// Tool: Tasks Delete Task
	s.AddTool(mcp.NewTool("tasks_delete_task",
		mcp.WithDescription("Delete a task from a Google Tasks list"),
		mcp.WithString("task_list_id", mcp.Required(), mcp.Description("ID of the task list")),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("ID of the task to delete")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskListID, err := request.RequireString("task_list_id")
		if err != nil {
			return mcp.NewToolResultError("task_list_id is required"), nil
		}
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		if err := tasksService.DeleteTask(taskListID, taskID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Deleted task: %s", taskID)), nil
	})

	// Start server (stdio)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func handleAuthCommand() {
	// We parse subcommands manually since "auth" is the command
	if len(os.Args) < 3 {
		fmt.Println("Usage: gogo-mcp auth login --secrets <path>")
		os.Exit(1)
	}

	if os.Args[2] == "login" {
		loginCmd := flag.NewFlagSet("login", flag.ExitOnError)
		secretsPath := loginCmd.String("secrets", "", "Path to client_secrets.json")
		loginCmd.Parse(os.Args[3:])

		if *secretsPath == "" {
			fmt.Println("Error: --secrets flag is required")
			loginCmd.Usage()
			os.Exit(1)
		}

		// Read secrets
		secrets, err := os.ReadFile(*secretsPath)
		if err != nil {
			fmt.Printf("Error reading secrets file: %v\n", err)
			os.Exit(1)
		}

		// Perform login
		fmt.Println("Starting OAuth 2.0 flow...")
		scopes := []string{
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/spreadsheets",
			"https://www.googleapis.com/auth/contacts",
			"https://www.googleapis.com/auth/documents",
			tasks.TasksScope,
		}
		if err := auth.Login(context.Background(), secrets, scopes); err != nil {
			fmt.Printf("Login failed: %v\n", err)
			os.Exit(1)
		}

		// Save secrets for future use
		if err := auth.SaveSecrets(*secretsPath); err != nil {
			fmt.Printf("Warning: Failed to save secrets file for future use: %v\n", err)
		}

		fmt.Println("Setup complete! You can now run 'gogo-mcp' without arguments.")
	} else {
		fmt.Printf("Unknown auth command: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func pingHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError("message argument is required and must be a string"), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Pong: %s", message)), nil
}