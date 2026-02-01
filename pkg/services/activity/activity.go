package activity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/driveactivity/v2"
	"google.golang.org/api/option"
)

// Service wraps the Drive Activity API.
type Service struct {
	srv *driveactivity.Service
}

// New creates a new Service.
func New(ctx context.Context, opts ...option.ClientOption) (*Service, error) {
	srv, err := driveactivity.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive Activity client: %w", err)
	}
	return &Service{srv: srv}, nil
}

// ActivitySummary is a human-readable summary of a Drive activity (metadata-only, for low token usage).
type ActivitySummary struct {
	Timestamp string // RFC3339
	Action    string // e.g. "Edit", "Move", "Rename", "Create", "Comment", etc.
	Actor     string // e.g. "you" or "user@example.com"
	Target    string // e.g. file/folder title or "items/FILE_ID"
}

// GetRecentActivity returns recent Drive activity as human-readable summaries.
// timeRangeHours: how many hours back (default 24). itemName: optional "items/FILE_ID" to filter by file.
func (s *Service) GetRecentActivity(timeRangeHours int, pageSize int64, itemName string) ([]ActivitySummary, error) {
	if timeRangeHours <= 0 {
		timeRangeHours = 24
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	since := time.Now().Add(-time.Duration(timeRangeHours) * time.Hour)
	filter := fmt.Sprintf("time >= \"%s\"", since.UTC().Format(time.RFC3339))

	req := &driveactivity.QueryDriveActivityRequest{
		Filter:   filter,
		PageSize: pageSize,
	}
	if itemName != "" {
		if !strings.HasPrefix(itemName, "items/") {
			itemName = "items/" + itemName
		}
		req.ItemName = itemName
	}

	resp, err := s.srv.Activity.Query(req).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to query Drive activity: %w", err)
	}

	var out []ActivitySummary
	for _, a := range resp.Activities {
		sum := summarizeActivity(a)
		if sum != nil {
			out = append(out, *sum)
		}
	}
	return out, nil
}

func summarizeActivity(a *driveactivity.DriveActivity) *ActivitySummary {
	timestamp := a.Timestamp
	if timestamp == "" && a.TimeRange != nil {
		timestamp = a.TimeRange.StartTime
	}
	action := primaryActionDetail(a)
	actor := primaryActor(a)
	target := primaryTarget(a)
	if action == "" && actor == "" && target == "" {
		return nil
	}
	return &ActivitySummary{
		Timestamp: timestamp,
		Action:    action,
		Actor:     actor,
		Target:    target,
	}
}

func primaryActionDetail(a *driveactivity.DriveActivity) string {
	if a.PrimaryActionDetail == nil {
		return ""
	}
	d := a.PrimaryActionDetail
	switch {
	case d.Edit != nil:
		return "Edit"
	case d.Move != nil:
		return "Move"
	case d.Rename != nil:
		return "Rename"
	case d.Create != nil:
		return "Create"
	case d.Delete != nil:
		return "Delete"
	case d.Restore != nil:
		return "Restore"
	case d.PermissionChange != nil:
		return "Permission change"
	case d.Comment != nil:
		return "Comment"
	case d.Reference != nil:
		return "Reference"
	default:
		return "Activity"
	}
}

func primaryActor(a *driveactivity.DriveActivity) string {
	if len(a.Actors) == 0 {
		return ""
	}
	ac := a.Actors[0]
	if ac.User != nil {
		if ac.User.KnownUser != nil {
			if ac.User.KnownUser.IsCurrentUser {
				return "you"
			}
			return ac.User.KnownUser.PersonName
		}
	}
	return "unknown"
}

func primaryTarget(a *driveactivity.DriveActivity) string {
	if len(a.Targets) == 0 {
		return ""
	}
	t := a.Targets[0]
	if t.DriveItem != nil {
		if t.DriveItem.Title != "" {
			return t.DriveItem.Title
		}
		return t.DriveItem.Name
	}
	if t.Drive != nil {
		return t.Drive.Title
	}
	return ""
}
