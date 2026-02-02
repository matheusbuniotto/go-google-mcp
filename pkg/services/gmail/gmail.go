package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailService wraps the Google Gmail API.
type GmailService struct {
	srv *gmail.Service
}

// New creates a new GmailService.
func New(ctx context.Context, opts ...option.ClientOption) (*GmailService, error) {
	srv, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %w", err)
	}
	return &GmailService{srv: srv}, nil
}

// ListThreads lists threads matching the query.
func (g *GmailService) ListThreads(query string, limit int64) ([]*gmail.Thread, error) {
	if limit <= 0 {
		limit = 10
	}
	// 'me' is the special user ID for the authenticated user
	call := g.srv.Users.Threads.List("me").MaxResults(limit)
	if query != "" {
		call.Q(query)
	}

	r, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve threads: %w", err)
	}
	return r.Threads, nil
}

// GetThread retrieves a thread by ID.
func (g *GmailService) GetThread(threadID string) (*gmail.Thread, error) {
	t, err := g.srv.Users.Threads.Get("me", threadID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve thread: %w", err)
	}
	return t, nil
}

// SendEmail sends an email.
func (g *GmailService) SendEmail(to string, subject string, body string) (*gmail.Message, error) {
	msgStr := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	msg := &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(msgStr)),
	}

	m, err := g.srv.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to send message: %w", err)
	}
	return m, nil
}

// CreateDraft creates a draft email.
func (g *GmailService) CreateDraft(to string, subject string, body string) (*gmail.Draft, error) {
	msgStr := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	msg := &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(msgStr)),
	}

	draft := &gmail.Draft{
		Message: msg,
	}

	d, err := g.srv.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create draft: %w", err)
	}
	return d, nil
}

// TrashThread moves a thread to trash.
func (g *GmailService) TrashThread(threadID string) error {
	_, err := g.srv.Users.Threads.Trash("me", threadID).Do()
	return err
}

// ListLabels lists all labels.
func (g *GmailService) ListLabels() ([]*gmail.Label, error) {
	r, err := g.srv.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list labels: %w", err)
	}
	return r.Labels, nil
}

// Helper to extract text from a message payload
func ExtractMessageBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}
	var body string
	if payload.Body != nil && payload.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(payload.Body.Data)
		body += string(data)
	}

	for _, part := range payload.Parts {
		body += ExtractMessageBody(part)
	}
	return body
}

// Helper to find headers
func GetHeader(headers []*gmail.MessagePartHeader, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}
