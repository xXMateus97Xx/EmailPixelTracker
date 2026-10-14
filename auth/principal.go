package auth

import (
	"context"
	"errors"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

var ErrInvalidPrincipal = errors.New("auth: invalid principal")

// Principal is the authenticated user performing the current operation.
// It is built by the API layer (e.g. from a token) and attached to the request
// context with WithPrincipal.
type Principal struct {
	UserId   string
	TenantId string
	Role     domain.Role
	// ImpersonatedTenantId is only honoured for SuperAdmin principals.
	ImpersonatedTenantId string
}

func (p Principal) HasAnyRole(roles ...domain.Role) bool {
	for _, role := range roles {
		if p.Role == role {
			return true
		}
	}
	return false
}

func (p Principal) IsSuperAdmin() bool {
	return p.Role == domain.SuperAdmin
}

func (p Principal) IsImpersonating() bool {
	return p.IsSuperAdmin() && p.ImpersonatedTenantId != ""
}

// EffectiveTenantId is the tenant every tenant-scoped query must be filtered by.
// For a SuperAdmin that is not impersonating it is empty, which is the scope
// of the master (SuperAdmin) users.
func (p Principal) EffectiveTenantId() string {
	if p.IsSuperAdmin() {
		return p.ImpersonatedTenantId
	}
	return p.TenantId
}

// Validate guarantees the invariants the services rely on to isolate tenants.
func (p Principal) Validate() error {
	if p.UserId == "" || !p.Role.IsValid() {
		return ErrInvalidPrincipal
	}
	if p.IsSuperAdmin() {
		if p.TenantId != "" {
			return ErrInvalidPrincipal
		}
		return nil
	}
	if p.TenantId == "" || p.ImpersonatedTenantId != "" {
		return ErrInvalidPrincipal
	}
	return nil
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
