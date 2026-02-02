package keep

import (
	"context"
	"fmt"

	"google.golang.org/api/keep/v1"
	"google.golang.org/api/option"
)

// Service wraps the Google Keep API (google-api-go-client keep/v1).
type Service struct {
	srv *keep.Service
}

// New creates a new Service using the given client options (e.g. from auth).
func New(ctx context.Context, opts ...option.ClientOption) (*Service, error) {
	srv, err := keep.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Keep client: %w", err)
	}
	return &Service{srv: srv}, nil
}

// ListNotesOptions configures list behavior.
type ListNotesOptions struct {
	PageSize  int64 // Max notes per page (0 = server default)
	PageToken string
	Filter    string // e.g. "trashed = false" (AIP-160)
}

// ListNotes lists notes. Use filter "trashed = false" to exclude trashed.
func (s *Service) ListNotes(opts ListNotesOptions) (*keep.ListNotesResponse, error) {
	call := s.srv.Notes.List()
	if opts.PageSize > 0 {
		call = call.PageSize(opts.PageSize)
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	if opts.Filter != "" {
		call = call.Filter(opts.Filter)
	}
	return call.Do()
}

// CreateNote creates a new note. Body can be text-only, list-only, or nil.
// For list notes, pass listItems; each item can have text and checked.
func (s *Service) CreateNote(title string, bodyText string, listItems []*keep.ListItem) (*keep.Note, error) {
	note := &keep.Note{Title: title}
	if bodyText != "" {
		note.Body = &keep.Section{
			Text: &keep.TextContent{Text: bodyText},
		}
	} else if len(listItems) > 0 {
		note.Body = &keep.Section{
			List: &keep.ListContent{ListItems: listItems},
		}
	}
	return s.srv.Notes.Create(note).Do()
}

// GetNote returns a note by name (e.g. "notes/abc123" or id "abc123").
func (s *Service) GetNote(name string) (*keep.Note, error) {
	if name == "" {
		return nil, fmt.Errorf("note name is required")
	}
	if len(name) < 6 || name[:6] != "notes/" {
		name = "notes/" + name
	}
	return s.srv.Notes.Get(name).Do()
}

// DeleteNote deletes a note by name. Caller must be owner.
func (s *Service) DeleteNote(name string) error {
	if name == "" {
		return fmt.Errorf("note name is required")
	}
	if len(name) < 6 || name[:6] != "notes/" {
		name = "notes/" + name
	}
	_, err := s.srv.Notes.Delete(name).Do()
	return err
}

// UpdateNoteInput holds optional fields for editing a note.
// The Keep API has no update endpoint; we get the note, create a new one with merged content, then delete the old one (note ID will change).
type UpdateNoteInput struct {
	Title     string           // If non-empty, replace note title
	BodyText  string           // If non-empty, replace body with this text (clears list)
	ListItems []*keep.ListItem // If non-nil and len > 0, replace body with this list (clears text)
}

// UpdateNote "edits" a note by creating a new note with merged content and deleting the old one. Returns the new note (new name/id).
func (s *Service) UpdateNote(name string, in UpdateNoteInput) (*keep.Note, error) {
	existing, err := s.GetNote(name)
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}
	title := existing.Title
	if in.Title != "" {
		title = in.Title
	}
	var body *keep.Section
	if in.BodyText != "" {
		body = &keep.Section{Text: &keep.TextContent{Text: in.BodyText}}
	} else if len(in.ListItems) > 0 {
		body = &keep.Section{List: &keep.ListContent{ListItems: in.ListItems}}
	} else if existing.Body != nil {
		body = existing.Body
	}
	newNote := &keep.Note{Title: title, Body: body}
	created, err := s.srv.Notes.Create(newNote).Do()
	if err != nil {
		return nil, fmt.Errorf("create updated note: %w", err)
	}
	if err := s.DeleteNote(name); err != nil {
		return nil, fmt.Errorf("delete old note (new note %s was created): %w", created.Name, err)
	}
	return created, nil
}
