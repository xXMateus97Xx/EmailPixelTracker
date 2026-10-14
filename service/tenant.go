package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	tenantNameMinLength = 2
	tenantNameMaxLength = 200
)

type TenantInput struct {
	Name   string
	Active bool
}

// TenantService manages tenants. Tenants are global (not tenant data), so a
// SuperAdmin can change them even while impersonating an inactive tenant,
// e.g. to reactivate it.
type TenantService struct {
	authn   authenticator
	tenants repository.TenantRepository
}

func (s *TenantService) Create(ctx context.Context, input TenantInput) (*domain.Tenant, error) {
	if _, err := s.authn.authorize(ctx, tenantWriters...); err != nil {
		return nil, err
	}

	tenant := &domain.Tenant{
		Id:           newId(),
		Name:         strings.TrimSpace(input.Name),
		Active:       input.Active,
		CreatedAtUtc: nowUtc(),
	}
	if err := s.validate(ctx, tenant); err != nil {
		return nil, err
	}
	if err := s.tenants.Create(ctx, tenant); err != nil {
		return nil, mapRepoError(err)
	}
	return tenant, nil
}

func (s *TenantService) Update(ctx context.Context, id string, input TenantInput) (*domain.Tenant, error) {
	if _, err := s.authn.authorize(ctx, tenantWriters...); err != nil {
		return nil, err
	}

	tenant, err := s.tenants.GetById(ctx, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	tenant.Name = strings.TrimSpace(input.Name)
	tenant.Active = input.Active

	if err := s.validate(ctx, tenant); err != nil {
		return nil, err
	}
	if err := s.tenants.Update(ctx, tenant); err != nil {
		return nil, mapRepoError(err)
	}
	return tenant, nil
}

// GetById lets a SuperAdmin read any tenant; other users can only read their own.
func (s *TenantService) GetById(ctx context.Context, id string) (*domain.Tenant, error) {
	p, err := s.authn.current(ctx)
	if err != nil {
		return nil, err
	}
	if !p.IsSuperAdmin() && p.TenantId != id {
		return nil, ErrNotFound
	}
	tenant, err := s.tenants.GetById(ctx, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return tenant, nil
}

func (s *TenantService) List(ctx context.Context, page Page) (*PagedResult[domain.Tenant], error) {
	if _, err := s.authn.authorize(ctx, domain.SuperAdmin); err != nil {
		return nil, err
	}
	items, total, err := s.tenants.List(ctx, page.toPagination())
	if err != nil {
		return nil, mapRepoError(err)
	}
	return newPagedResult(items, total, page), nil
}

func (s *TenantService) validate(ctx context.Context, tenant *domain.Tenant) error {
	v := &validator{}
	nameLength := utf8.RuneCountInString(tenant.Name)
	v.check(nameLength >= tenantNameMinLength && nameLength <= tenantNameMaxLength,
		"name", "must have between 2 and 200 characters")
	if err := v.err(); err != nil {
		return err
	}

	taken, err := s.tenants.ExistsByName(ctx, tenant.Name, tenant.Id)
	if err != nil {
		return err
	}
	v.check(!taken, "name", "is already in use")
	return v.err()
}
