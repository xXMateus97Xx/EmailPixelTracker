package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

var (
	ErrNotFound  = errors.New("repository: not found")
	ErrDuplicate = errors.New("repository: duplicate key")
)

type Pagination struct {
	Offset int
	Limit  int
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	Update(ctx context.Context, tenant *domain.Tenant) error
	GetById(ctx context.Context, id string) (*domain.Tenant, error)
	// ExistsByName ignores the tenant with id excludeId (use "" to ignore none).
	ExistsByName(ctx context.Context, name, excludeId string) (bool, error)
	List(ctx context.Context, page Pagination) ([]domain.Tenant, int64, error)
}

// Every UserRepository method that reads or writes existing users is scoped by
// tenantId. An empty tenantId is the scope of SuperAdmin users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, tenantId string, user *domain.User) error
	GetById(ctx context.Context, tenantId, id string) (*domain.User, error)
	// GetByUsername searches all tenants, since usernames are used to log in.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	// ExistsByUsername checks usernames globally, since they are used to log in.
	ExistsByUsername(ctx context.Context, username, excludeId string) (bool, error)
	List(ctx context.Context, tenantId string, page Pagination) ([]domain.User, int64, error)
}

// Every CampaignRepository method is scoped by tenantId.
type CampaignRepository interface {
	Create(ctx context.Context, campaign *domain.Campaign) error
	Update(ctx context.Context, tenantId string, campaign *domain.Campaign) error
	GetById(ctx context.Context, tenantId, id string) (*domain.Campaign, error)
	// GetByIdAnyTenant is NOT scoped by tenant. It exists only for the public
	// tracking pixel, which has no authenticated user; never use it elsewhere.
	GetByIdAnyTenant(ctx context.Context, id string) (*domain.Campaign, error)
	ExistsByName(ctx context.Context, tenantId, name, excludeId string) (bool, error)
	List(ctx context.Context, tenantId string, page Pagination) ([]domain.Campaign, int64, error)
}

// CampaignViewRepository is scoped by campaign: callers must check that the
// campaign belongs to the current tenant before reading its views.
type CampaignViewRepository interface {
	Create(ctx context.Context, view *domain.CampaignView) error
	Summary(ctx context.Context, campaignId string, period domain.Period) (domain.CampaignViewSummary, error)
	// DailySummary returns only the days with views, ordered by date.
	DailySummary(ctx context.Context, campaignId string, period domain.Period) ([]domain.DailyCampaignViews, error)
	// UserAgentVisitors returns one entry per distinct user agent and visitor,
	// in no particular order. The user agent is parsed by the caller, so the
	// unique views of a group of user agents can still be counted exactly.
	UserAgentVisitors(ctx context.Context, campaignId string, period domain.Period) ([]domain.VisitorUserAgentViews, error)
	// List returns the views ordered from the most recent.
	List(ctx context.Context, campaignId string, period domain.Period, page Pagination) ([]domain.CampaignView, int64, error)
}

// Repositories gives access to every repository. Each one is created lazily,
// on the first call to its accessor, and reused afterwards. It is safe for
// concurrent use.
type Repositories struct {
	tenants       func() TenantRepository
	users         func() UserRepository
	campaigns     func() CampaignRepository
	campaignViews func() CampaignViewRepository
	ping          func(ctx context.Context) error
	close         func(ctx context.Context) error
}

// Constructors holds the constructor of each repository; they are called at
// most once, when the repository is first needed.
type Constructors struct {
	Tenants       func() TenantRepository
	Users         func() UserRepository
	Campaigns     func() CampaignRepository
	CampaignViews func() CampaignViewRepository
	// Ping checks that the database answers. Optional.
	Ping func(ctx context.Context) error
	// Close releases the underlying connection. Optional.
	Close func(ctx context.Context) error
}

func NewRepositories(c Constructors) *Repositories {
	return &Repositories{
		tenants:       sync.OnceValue(c.Tenants),
		users:         sync.OnceValue(c.Users),
		campaigns:     sync.OnceValue(c.Campaigns),
		campaignViews: sync.OnceValue(c.CampaignViews),
		ping:          c.Ping,
		close:         c.Close,
	}
}

func (r *Repositories) Tenants() TenantRepository             { return r.tenants() }
func (r *Repositories) Users() UserRepository                 { return r.users() }
func (r *Repositories) Campaigns() CampaignRepository         { return r.campaigns() }
func (r *Repositories) CampaignViews() CampaignViewRepository { return r.campaignViews() }

// Ping checks that the database answers; used by the readiness check.
func (r *Repositories) Ping(ctx context.Context) error {
	if r.ping == nil {
		return nil
	}
	return r.ping(ctx)
}

// Close releases the underlying database connection.
func (r *Repositories) Close(ctx context.Context) error {
	if r.close == nil {
		return nil
	}
	return r.close(ctx)
}
