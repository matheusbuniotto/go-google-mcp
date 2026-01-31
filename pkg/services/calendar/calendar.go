package calendar

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarService wraps the Google Calendar API.
type CalendarService struct {
	srv *calendar.Service
}

// New creates a new CalendarService.
func New(ctx context.Context, opts ...option.ClientOption) (*CalendarService, error) {
	srv, err := calendar.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}
	return &CalendarService{srv: srv}, nil
}

// ListEvents lists upcoming events.
func (c *CalendarService) ListEvents(calendarId string, maxResults int64, timeMin string, timeMax string) ([]*calendar.Event, error) {
	if calendarId == "" {
		calendarId = "primary"
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	
	call := c.srv.Events.List(calendarId).
		ShowDeleted(false).
		SingleEvents(true).
		MaxResults(maxResults).
		OrderBy("startTime")

	if timeMin != "" {
		call.TimeMin(timeMin)
	} else {
		// Default to now if not specified? 
		// Actually, standard behavior is usually from now if not specified for "upcoming".
		call.TimeMin(time.Now().Format(time.RFC3339))
	}
	if timeMax != "" {
		call.TimeMax(timeMax)
	}

	events, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events: %w", err)
	}
	return events.Items, nil
}

// CreateEvent creates a new event.
func (c *CalendarService) CreateEvent(calendarId string, summary string, description string, startTime string, endTime string, attendees []string) (*calendar.Event, error) {
	if calendarId == "" {
		calendarId = "primary"
	}

	event := &calendar.Event{
		Summary:     summary,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: startTime,
			TimeZone: "UTC", // Or infer?
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
			TimeZone: "UTC",
		},
	}

	if len(attendees) > 0 {
		var atts []*calendar.EventAttendee
		for _, email := range attendees {
			atts = append(atts, &calendar.EventAttendee{Email: email})
		}
		event.Attendees = atts
	}

	e, err := c.srv.Events.Insert(calendarId, event).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create event: %w", err)
	}
	return e, nil
}

// DeleteEvent deletes an event.
func (c *CalendarService) DeleteEvent(calendarId string, eventId string) error {
	if calendarId == "" {
		calendarId = "primary"
	}
	return c.srv.Events.Delete(calendarId, eventId).Do()
}
