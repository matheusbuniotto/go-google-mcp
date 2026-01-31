package docs

import (
	"context"
	"fmt"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

// DocsService wraps the Google Docs API.
type DocsService struct {
	srv *docs.Service
}

// New creates a new DocsService.
func New(ctx context.Context, opts ...option.ClientOption) (*DocsService, error) {
	srv, err := docs.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Docs client: %w", err)
	}
	return &DocsService{srv: srv}, nil
}

// CreateDocument creates a new document.
func (d *DocsService) CreateDocument(title string) (*docs.Document, error) {
	doc := &docs.Document{
		Title: title,
	}
	resp, err := d.srv.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create document: %w", err)
	}
	return resp, nil
}

// GetDocument reads a document.
func (d *DocsService) GetDocument(documentId string) (*docs.Document, error) {
	doc, err := d.srv.Documents.Get(documentId).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve document: %w", err)
	}
	return doc, nil
}

// InsertText inserts text at an index (or end if index=0, though Docs API is precise).
// Simpler: Insert at end using EndOfSegmentLocation.
func (d *DocsService) InsertText(documentId string, text string) error {
	req := &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Text: text,
			EndOfSegmentLocation: &docs.EndOfSegmentLocation{
				SegmentId: "", // Body
			},
		},
	}
	
	batchUpdate := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{req},
	}

	_, err := d.srv.Documents.BatchUpdate(documentId, batchUpdate).Do()
	return err
}
