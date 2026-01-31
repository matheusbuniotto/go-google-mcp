package people

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// PeopleService wraps the Google People API.
type PeopleService struct {
	srv *people.Service
}

// New creates a new PeopleService.
func New(ctx context.Context, opts ...option.ClientOption) (*PeopleService, error) {
	srv, err := people.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve People client: %w", err)
	}
	return &PeopleService{srv: srv}, nil
}

// CreateContact creates a new contact.
func (p *PeopleService) CreateContact(givenName string, familyName string, email string) (*people.Person, error) {
	contact := &people.Person{
		Names: []*people.Name{
			{
				GivenName:  givenName,
				FamilyName: familyName,
			},
		},
	}
	if email != "" {
		contact.EmailAddresses = []*people.EmailAddress{
			{
				Value: email,
			},
		}
	}

	resp, err := p.srv.People.CreateContact(contact).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create contact: %w", err)
	}
	return resp, nil
}

// SearchContacts searches for contacts.
func (p *PeopleService) SearchContacts(query string) ([]*people.Person, error) {
	// People API search is a bit complex. 
	// Simplest is SearchContacts method if enabled, or listing "people/me" and filtering.
	// Let's use SearchContacts.
	
	call := p.srv.People.SearchContacts().
		Query(query).
		ReadMask("names,emailAddresses")
	
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to search contacts: %w", err)
	}
	
	var results []*people.Person
	for _, res := range resp.Results {
		if res.Person != nil {
			results = append(results, res.Person)
		}
	}
	return results, nil
}

// ListConnections lists the authenticated user's contacts.
func (p *PeopleService) ListConnections(limit int64) ([]*people.Person, error) {
	if limit <= 0 {
		limit = 10
	}
	resp, err := p.srv.People.Connections.List("people/me").
		PageSize(limit).
		PersonFields("names,emailAddresses").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list connections: %w", err)
	}
	return resp.Connections, nil
}
