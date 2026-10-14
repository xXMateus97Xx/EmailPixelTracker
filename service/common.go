package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Page struct {
	Number int // 1-based
	Size   int
}

func (p Page) normalized() Page {
	if p.Number < 1 {
		p.Number = 1
	}
	if p.Size < 1 {
		p.Size = DefaultPageSize
	}
	if p.Size > MaxPageSize {
		p.Size = MaxPageSize
	}
	return p
}

func (p Page) toPagination() repository.Pagination {
	p = p.normalized()
	return repository.Pagination{Offset: (p.Number - 1) * p.Size, Limit: p.Size}
}

type PagedResult[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}

func newPagedResult[T any](items []T, total int64, page Page) *PagedResult[T] {
	page = page.normalized()
	if items == nil {
		items = []T{}
	}
	return &PagedResult[T]{Items: items, Total: total, Page: page.Number, PageSize: page.Size}
}

var nowUtc = func() time.Time { return time.Now().UTC() }

func newId() string { return uuid.NewString() }

// Session is the resolved state of the current operation.
type Session struct {
	auth.Principal
	// ReadOnly is true when a SuperAdmin impersonates an inactive tenant: its
	// data can be read but not changed.
	ReadOnly bool
	// Tenant is the effective tenant (own or impersonated); nil for a
	// SuperAdmin that is not impersonating.
	Tenant *domain.Tenant
}

// authenticator resolves the session of the current operation. The principal
// in the context (e.g. from a token) only identifies the user: the user and
// its tenant are reloaded on every operation so deactivations and role changes
// apply at once.
type authenticator struct {
	users   repository.UserRepository
	tenants repository.TenantRepository
}

func (a authenticator) current(ctx context.Context) (Session, error) {
	p, ok := auth.FromContext(ctx)
	if !ok || p.Validate() != nil {
		return Session{}, ErrUnauthenticated
	}

	// Looking the user up by its own tenant also rejects forged tenant claims.
	user, err := a.users.GetById(ctx, p.TenantId, p.UserId)
	if errors.Is(err, repository.ErrNotFound) {
		return Session{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, err
	}
	if !user.Active {
		return Session{}, fmt.Errorf("%w: user is inactive", ErrUnauthenticated)
	}

	p.Role = user.Role
	if !p.IsSuperAdmin() {
		p.ImpersonatedTenantId = ""
	}
	if p.Validate() != nil {
		return Session{}, ErrUnauthenticated
	}

	session := Session{Principal: p}
	if tenantId := p.EffectiveTenantId(); tenantId != "" {
		tenant, err := a.tenants.GetById(ctx, tenantId)
		if errors.Is(err, repository.ErrNotFound) {
			return Session{}, ErrUnauthenticated
		}
		if err != nil {
			return Session{}, err
		}
		if !tenant.Active {
			if !p.IsImpersonating() {
				return Session{}, ErrTenantInactive
			}
			session.ReadOnly = true
		}
		session.Tenant = tenant
	}
	return session, nil
}

// authorize returns the current session if it has at least one of the roles.
// Use it for reads and for operations that are not scoped to a tenant.
func (a authenticator) authorize(ctx context.Context, roles ...domain.Role) (Session, error) {
	s, err := a.current(ctx)
	if err != nil {
		return s, err
	}
	if !s.HasAnyRole(roles...) {
		return s, ErrForbidden
	}
	return s, nil
}

// authorizeWrite is authorize for operations that change tenant data, which
// are refused while impersonating an inactive tenant.
func (a authenticator) authorizeWrite(ctx context.Context, roles ...domain.Role) (Session, error) {
	s, err := a.authorize(ctx, roles...)
	if err != nil {
		return s, err
	}
	if s.ReadOnly {
		return s, fmt.Errorf("%w: impersonated tenant is read-only", ErrTenantInactive)
	}
	return s, nil
}

// tenantScope authorizes the session and returns the tenant all queries must
// be filtered by, for data that always belongs to a tenant (campaigns and
// their views). A SuperAdmin must impersonate a tenant first.
func (a authenticator) tenantScope(ctx context.Context, roles []domain.Role, write bool) (string, error) {
	authorize := a.authorize
	if write {
		authorize = a.authorizeWrite
	}
	s, err := authorize(ctx, roles...)
	if err != nil {
		return "", err
	}
	tenantId := s.EffectiveTenantId()
	if tenantId == "" {
		return "", ErrImpersonationRequired
	}
	return tenantId, nil
}

// Roles allowed to create and edit each entity. API users can never write.
var (
	tenantWriters   = []domain.Role{domain.SuperAdmin}
	userWriters     = []domain.Role{domain.SuperAdmin, domain.Admin}
	campaignWriters = []domain.Role{domain.SuperAdmin, domain.Admin, domain.ContentManager}
	userReaders     = userWriters
	campaignReaders = []domain.Role{domain.SuperAdmin, domain.Admin, domain.ContentManager, domain.API}
)

// passwordHasher hashes passwords with the configured bcrypt cost.
type passwordHasher struct {
	cost int
}

func (h passwordHasher) hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
