package drive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveService wraps the Google Drive API.
type DriveService struct {
	srv *drive.Service
}

// New creates a new DriveService.
func New(ctx context.Context, opts ...option.ClientOption) (*DriveService, error) {
	srv, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %w", err)
	}
	return &DriveService{srv: srv}, nil
}

// ListFiles lists the first n files.
func (d *DriveService) ListFiles(limit int64) ([]*drive.File, error) {
	if limit <= 0 {
		limit = 10
	}
	r, err := d.srv.Files.List().
		PageSize(limit).
		Fields("nextPageToken, files(id, name, mimeType, parents)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %w", err)
	}
	return r.Files, nil
}

// SearchFiles searches for files using specific criteria.
// Use empty query to list non-trashed files (account-wide). Default filter is trashed = false.
func (d *DriveService) SearchFiles(query string, limit int64) ([]*drive.File, error) {
	if limit <= 0 {
		limit = 10
	}
	if query == "" {
		query = "trashed = false"
	} else if !strings.Contains(query, "trashed") {
		query = fmt.Sprintf("(%s) and trashed = false", query)
	}

	r, err := d.srv.Files.List().
		Q(query).
		PageSize(limit).
		Fields("nextPageToken, files(id, name, mimeType, parents)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to search files: %w", err)
	}
	return r.Files, nil
}

// SearchFileResult holds a file and an optional content snippet (e.g. first N bytes).
type SearchFileResult struct {
	File    *drive.File
	Snippet string
}

// SearchFilesWithSnippets runs SearchFiles and optionally fetches a short content snippet per file.
// maxSnippetBytes limits snippet length per file; 0 disables snippets. Snippet fetch errors are ignored.
func (d *DriveService) SearchFilesWithSnippets(query string, limit int64, maxSnippetBytes int64) ([]SearchFileResult, error) {
	files, err := d.SearchFiles(query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]SearchFileResult, len(files))
	for i, f := range files {
		out[i] = SearchFileResult{File: f}
		if maxSnippetBytes <= 0 {
			continue
		}
		snippet, err := d.ReadFileContent(f.Id, maxSnippetBytes)
		if err != nil {
			continue // leave Snippet empty on error
		}
		out[i].Snippet = snippet
	}
	return out, nil
}

// findFilesQuery builds the Drive fullText query for a search term (escapes ' and \).
func findFilesQuery(searchTerm string) string {
	escaped := strings.ReplaceAll(searchTerm, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return fmt.Sprintf("fullText contains '%s' and trashed = false", escaped)
}

// FindFiles runs an account-wide fullText search. Use for discovery when you know a phrase to search for.
func (d *DriveService) FindFiles(searchTerm string, limit int64) ([]*drive.File, error) {
	if searchTerm == "" {
		return d.SearchFiles("", limit)
	}
	return d.SearchFiles(findFilesQuery(searchTerm), limit)
}

// FindFilesWithSnippets runs FindFiles and optionally fetches a short content snippet per file.
func (d *DriveService) FindFilesWithSnippets(searchTerm string, limit int64, maxSnippetBytes int64) ([]SearchFileResult, error) {
	if searchTerm == "" {
		return d.SearchFilesWithSnippets("trashed = false", limit, maxSnippetBytes)
	}
	return d.SearchFilesWithSnippets(findFilesQuery(searchTerm), limit, maxSnippetBytes)
}

// ReadFileContent downloads and reads the content of a file.
// limitBytes limits the number of bytes read. -1 for no limit (use with caution).
func (d *DriveService) ReadFileContent(fileID string, limitBytes int64) (string, error) {
	// Check file metadata first to see if we need to export
	f, err := d.srv.Files.Get(fileID).Fields("mimeType").Do()
	if err != nil {
		return "", fmt.Errorf("unable to get file metadata: %w", err)
	}

	var resp *http.Response

	// Handle Google Workspace documents by Exporting
	if strings.HasPrefix(f.MimeType, "application/vnd.google-apps.") {
		// Default export formats:
		// Docs -> text/plain
		// Sheets -> application/pdf (no text export), or csv? Sheets CSV export is usually via "text/csv"
		// Slides -> text/plain

		exportMime := "text/plain"
		if f.MimeType == "application/vnd.google-apps.spreadsheet" {
			exportMime = "text/csv"
		}
		// Try export
		resp, err = d.srv.Files.Export(fileID, exportMime).Download()
		if err != nil {
			// Fallback or specific error handling
			// If text/plain isn't supported for this type, return error
			return "", fmt.Errorf("unable to export file (mime: %s) as %s: %w", f.MimeType, exportMime, err)
		}
	} else {
		// Standard binary download
		resp, err = d.srv.Files.Get(fileID).Download()
		if err != nil {
			return "", fmt.Errorf("unable to download file: %w", err)
		}
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if limitBytes > 0 {
		reader = io.LimitReader(resp.Body, limitBytes)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("unable to read file content: %w", err)
	}
	return string(content), nil
}

// CreateFolder creates a new folder.
func (d *DriveService) CreateFolder(name string, parentID string) (*drive.File, error) {
	f := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}
	if parentID != "" {
		f.Parents = []string{parentID}
	}

	file, err := d.srv.Files.Create(f).Fields("id", "name", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create folder: %w", err)
	}
	return file, nil
}

// CreateFile creates a new file with content.
func (d *DriveService) CreateFile(name string, parentID string, content string, mimeType string) (*drive.File, error) {
	f := &drive.File{
		Name: name,
	}
	if parentID != "" {
		f.Parents = []string{parentID}
	}

	media := strings.NewReader(content)

	call := d.srv.Files.Create(f).Media(media)
	if mimeType != "" {
		f.MimeType = mimeType
	}

	file, err := call.Fields("id", "name", "mimeType", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	return file, nil
}

// UpdateFile updates a file's name, parent, or content.
func (d *DriveService) UpdateFile(fileID string, name string, addParents string, removeParents string, content *string) (*drive.File, error) {
	f := &drive.File{}
	if name != "" {
		f.Name = name
	}

	call := d.srv.Files.Update(fileID, f)

	if addParents != "" {
		call.AddParents(addParents)
	}
	if removeParents != "" {
		call.RemoveParents(removeParents)
	}

	if content != nil {
		media := strings.NewReader(*content)
		call.Media(media)
	}

	file, err := call.Fields("id", "name", "mimeType", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to update file: %w", err)
	}
	return file, nil
}

// DeleteFile deletes a file or folder (Trash).
// Renaming to TrashFile for clarity, but standard Delete usually means trash in Drive UI unless 'delete' API is used.
// API: Files.Delete permanently deletes. Files.Update(trashed=true) moves to trash.
// We should probably implement explicit Trash and explicit Delete.
// Current implementation uses Files.Delete which is PERMANENT. This is dangerous.
// Recommendation: Change DeleteFile to TrashFile.
func (d *DriveService) TrashFile(fileID string) error {
	f := &drive.File{Trashed: true}
	_, err := d.srv.Files.Update(fileID, f).Do()
	return err
}

// AddPermission shares a file.
func (d *DriveService) AddPermission(fileID string, role string, type_ string, emailAddress string) error {
	perm := &drive.Permission{
		Role:         role,
		Type:         type_,
		EmailAddress: emailAddress,
	}
	_, err := d.srv.Permissions.Create(fileID, perm).Do()
	return err
}

// ListComments lists comments on a Drive file (e.g. Doc, Sheet).
func (d *DriveService) ListComments(fileID string, pageSize int64) ([]*drive.Comment, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	resp, err := d.srv.Comments.List(fileID).PageSize(pageSize).Fields("comments(id,content,createdTime,author,resolved)").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list comments: %w", err)
	}
	return resp.Comments, nil
}

// CreateComment adds a comment to a Drive file.
func (d *DriveService) CreateComment(fileID string, content string) (*drive.Comment, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	comment := &drive.Comment{Content: content}
	c, err := d.srv.Comments.Create(fileID, comment).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create comment: %w", err)
	}
	return c, nil
}
