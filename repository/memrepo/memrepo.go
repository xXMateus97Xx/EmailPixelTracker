// Package memrepo is an in-memory implementation of the repositories, meant
// for tests.
package memrepo

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

func New() *repository.Repositories {
	return repository.NewRepositories(repository.Constructors{
		Tenants: func() repository.TenantRepository {
			return &TenantRepository{store[domain.Tenant]{items: map[string]domain.Tenant{}}}
		},
		Users: func() repository.UserRepository {
			return &UserRepository{store[domain.User]{items: map[string]domain.User{}}}
		},
		Campaigns: func() repository.CampaignRepository {
			return &CampaignRepository{store[domain.Campaign]{items: map[string]domain.Campaign{}}}
		},
		CampaignViews: func() repository.CampaignViewRepository {
			return &CampaignViewRepository{store[domain.CampaignView]{items: map[string]domain.CampaignView{}}}
		},
	})
}

type store[T any] struct {
	mu    sync.RWMutex
	items map[string]T
}

func (s *store[T]) find(match func(T) bool) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if match(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

func (s *store[T]) put(id string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = item
}

// replace overwrites the item only if the current one matches.
func (s *store[T]) replace(id string, item T, match func(T) bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cur, ok := s.items[id]; !ok || !match(cur) {
		return repository.ErrNotFound
	}
	s.items[id] = item
	return nil
}

func (s *store[T]) list(match func(T) bool, createdAt func(T) int64, id func(T) string, page repository.Pagination) ([]T, int64) {
	s.mu.RLock()
	var all []T
	for _, item := range s.items {
		if match(item) {
			all = append(all, item)
		}
	}
	s.mu.RUnlock()

	slices.SortFunc(all, func(a, b T) int {
		if c := createdAt(b) - createdAt(a); c != 0 {
			if c > 0 {
				return 1
			}
			return -1
		}
		return strings.Compare(id(a), id(b))
	})

	total := int64(len(all))
	start := min(page.Offset, len(all))
	end := min(start+page.Limit, len(all))
	return all[start:end], total
}

type TenantRepository struct{ store[domain.Tenant] }

func (r *TenantRepository) Create(_ context.Context, t *domain.Tenant) error {
	if _, dup := r.find(func(x domain.Tenant) bool { return x.Name == t.Name }); dup {
		return repository.ErrDuplicate
	}
	r.put(t.Id, *t)
	return nil
}

func (r *TenantRepository) Update(_ context.Context, t *domain.Tenant) error {
	return r.replace(t.Id, *t, func(domain.Tenant) bool { return true })
}

func (r *TenantRepository) GetById(_ context.Context, id string) (*domain.Tenant, error) {
	t, ok := r.find(func(x domain.Tenant) bool { return x.Id == id })
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &t, nil
}

func (r *TenantRepository) ExistsByName(_ context.Context, name, excludeId string) (bool, error) {
	_, ok := r.find(func(x domain.Tenant) bool { return x.Name == name && x.Id != excludeId })
	return ok, nil
}

func (r *TenantRepository) List(_ context.Context, page repository.Pagination) ([]domain.Tenant, int64, error) {
	items, total := r.list(
		func(domain.Tenant) bool { return true },
		func(x domain.Tenant) int64 { return x.CreatedAtUtc.UnixNano() },
		func(x domain.Tenant) string { return x.Id },
		page)
	return items, total, nil
}

type UserRepository struct{ store[domain.User] }

func (r *UserRepository) Create(_ context.Context, u *domain.User) error {
	if _, dup := r.find(func(x domain.User) bool { return x.Username == u.Username }); dup {
		return repository.ErrDuplicate
	}
	r.put(u.Id, *u)
	return nil
}

func (r *UserRepository) Update(_ context.Context, tenantId string, u *domain.User) error {
	return r.replace(u.Id, *u, func(x domain.User) bool { return x.TenantId == tenantId })
}

func (r *UserRepository) GetById(_ context.Context, tenantId, id string) (*domain.User, error) {
	u, ok := r.find(func(x domain.User) bool { return x.Id == id && x.TenantId == tenantId })
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	u, ok := r.find(func(x domain.User) bool { return x.Username == username })
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (r *UserRepository) ExistsByUsername(_ context.Context, username, excludeId string) (bool, error) {
	_, ok := r.find(func(x domain.User) bool { return x.Username == username && x.Id != excludeId })
	return ok, nil
}

func (r *UserRepository) List(_ context.Context, tenantId string, page repository.Pagination) ([]domain.User, int64, error) {
	items, total := r.list(
		func(x domain.User) bool { return x.TenantId == tenantId },
		func(x domain.User) int64 { return x.CreatedAtUtc.UnixNano() },
		func(x domain.User) string { return x.Id },
		page)
	return items, total, nil
}

type CampaignRepository struct{ store[domain.Campaign] }

func (r *CampaignRepository) Create(_ context.Context, c *domain.Campaign) error {
	if _, dup := r.find(func(x domain.Campaign) bool { return x.TenantId == c.TenantId && x.Name == c.Name }); dup {
		return repository.ErrDuplicate
	}
	r.put(c.Id, *c)
	return nil
}

func (r *CampaignRepository) Update(_ context.Context, tenantId string, c *domain.Campaign) error {
	return r.replace(c.Id, *c, func(x domain.Campaign) bool { return x.TenantId == tenantId })
}

func (r *CampaignRepository) GetById(_ context.Context, tenantId, id string) (*domain.Campaign, error) {
	c, ok := r.find(func(x domain.Campaign) bool { return x.Id == id && x.TenantId == tenantId })
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &c, nil
}

func (r *CampaignRepository) GetByIdAnyTenant(_ context.Context, id string) (*domain.Campaign, error) {
	c, ok := r.find(func(x domain.Campaign) bool { return x.Id == id })
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &c, nil
}

func (r *CampaignRepository) ExistsByName(_ context.Context, tenantId, name, excludeId string) (bool, error) {
	_, ok := r.find(func(x domain.Campaign) bool {
		return x.TenantId == tenantId && x.Name == name && x.Id != excludeId
	})
	return ok, nil
}

func (r *CampaignRepository) List(_ context.Context, tenantId string, page repository.Pagination) ([]domain.Campaign, int64, error) {
	items, total := r.list(
		func(x domain.Campaign) bool { return x.TenantId == tenantId },
		func(x domain.Campaign) int64 { return x.CreatedAtUtc.UnixNano() },
		func(x domain.Campaign) string { return x.Id },
		page)
	return items, total, nil
}
