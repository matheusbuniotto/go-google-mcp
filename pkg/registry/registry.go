package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/matheusbuniotto/go-google-mcp/pkg/auth"
	activitysvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/activity"
	calendarsvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/calendar"
	docssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/docs"
	drivesvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/drive"
	gmailsvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/gmail"
	keepsvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/keep"
	peoplesvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/people"
	sheetssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/sheets"
	taskssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/tasks"
	"google.golang.org/api/option"
)

// ServiceSet holds all Google service instances for one account.
type ServiceSet struct {
	Drive    *drivesvc.DriveService
	Gmail    *gmailsvc.GmailService
	Calendar *calendarsvc.CalendarService
	Sheets   *sheetssvc.SheetsService
	People   *peoplesvc.PeopleService
	Docs     *docssvc.DocsService
	Tasks    *taskssvc.Service
	Activity *activitysvc.Service
	Keep     *keepsvc.Service
}

// NewServiceSet creates all 9 Google services with the given auth options.
func NewServiceSet(ctx context.Context, opts ...option.ClientOption) (*ServiceSet, error) {
	driveSvc, err := drivesvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("drive: %w", err)
	}
	gmailSvc, err := gmailsvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gmail: %w", err)
	}
	calendarSvc, err := calendarsvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("calendar: %w", err)
	}
	sheetsSvc, err := sheetssvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("sheets: %w", err)
	}
	peopleSvc, err := peoplesvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("people: %w", err)
	}
	docsSvc, err := docssvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("docs: %w", err)
	}
	tasksSvc, err := taskssvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("tasks: %w", err)
	}
	activitySvc, err := activitysvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("activity: %w", err)
	}
	keepSvc, err := keepsvc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("keep: %w", err)
	}
	return &ServiceSet{
		Drive:    driveSvc,
		Gmail:    gmailSvc,
		Calendar: calendarSvc,
		Sheets:   sheetsSvc,
		People:   peopleSvc,
		Docs:     docsSvc,
		Tasks:    tasksSvc,
		Activity: activitySvc,
		Keep:     keepSvc,
	}, nil
}

// Registry manages multiple account ServiceSets with lazy initialization.
type Registry struct {
	mu       sync.Mutex
	accounts map[string]*ServiceSet
	scopes   []string

	// legacy is the pre-existing single-account ServiceSet (backward compat).
	legacy *ServiceSet

	// multiAccount indicates whether accounts/ directory was detected.
	multiAccount bool
}

// NewLegacyRegistry creates a registry wrapping a single pre-initialized ServiceSet.
// Used when no accounts/ directory exists (backward compatible mode).
func NewLegacyRegistry(ss *ServiceSet) *Registry {
	return &Registry{
		legacy:       ss,
		multiAccount: false,
	}
}

// NewMultiAccountRegistry creates a registry for multi-account mode.
// ServiceSets are created lazily on first use per account.
func NewMultiAccountRegistry(scopes []string) *Registry {
	return &Registry{
		accounts:     make(map[string]*ServiceSet),
		scopes:       scopes,
		multiAccount: true,
	}
}

// IsMultiAccount returns whether the registry is in multi-account mode.
func (r *Registry) IsMultiAccount() bool {
	return r.multiAccount
}

// Resolve returns the ServiceSet for a tool call.
//
// Resolution rules:
//   - Legacy mode: always returns the legacy ServiceSet (account param ignored).
//   - Multi-account, account provided: returns that account's ServiceSet (lazy init).
//   - Multi-account, account empty, 1 account: auto-selects the single account.
//   - Multi-account, account empty, N accounts: returns error with account list.
func (r *Registry) Resolve(account string) (*ServiceSet, error) {
	if !r.multiAccount {
		return r.legacy, nil
	}

	if account == "" {
		accounts, err := auth.ListAccounts()
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}
		switch len(accounts) {
		case 0:
			return nil, fmt.Errorf("no accounts configured; run: go-google-mcp auth login --account <email> --secrets <path>")
		case 1:
			account = accounts[0]
		default:
			return nil, fmt.Errorf("multiple accounts available, please specify 'account' parameter; available: %v", accounts)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if ss, ok := r.accounts[account]; ok {
		return ss, nil
	}

	// Lazy init: create service set for this account.
	ctx := context.Background()
	opts, err := auth.GetClientOptionsForAccount(ctx, account, r.scopes)
	if err != nil {
		return nil, fmt.Errorf("auth for account %q: %w", account, err)
	}
	ss, err := NewServiceSet(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("services for account %q: %w", account, err)
	}
	r.accounts[account] = ss
	return ss, nil
}
