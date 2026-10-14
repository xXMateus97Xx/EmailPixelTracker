// Package service holds the business rules: authorization by role, tenant
// isolation and validation. The current user must be attached to the context
// with auth.WithPrincipal before calling any service; it and its tenant are
// reloaded from the repository on every operation and rejected if missing or
// inactive.
package service

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type Config struct {
	// BcryptCost is the cost used to hash new passwords (bcrypt.MinCost to
	// bcrypt.MaxCost). Zero means bcrypt.DefaultCost. Changing it does not
	// affect existing hashes, which keep working.
	BcryptCost int
}

// Services gives access to every service. Each one (and the repositories it
// depends on) is created lazily, on the first call to its accessor, and reused
// afterwards. It is safe for concurrent use.
type Services struct {
	auth          func() *AuthService
	tenants       func() *TenantService
	users         func() *UserService
	campaigns     func() *CampaignService
	campaignViews func() *CampaignViewService
}

func New(repos *repository.Repositories, cfg Config) (*Services, error) {
	cost := cfg.BcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	hasher := passwordHasher{cost: cost}
	authn := func() authenticator {
		return authenticator{users: repos.Users(), tenants: repos.Tenants()}
	}

	return &Services{
		auth: sync.OnceValue(func() *AuthService {
			return newAuthService(authn(), hasher, repos.Users(), repos.Tenants())
		}),
		tenants: sync.OnceValue(func() *TenantService {
			return &TenantService{authn: authn(), tenants: repos.Tenants()}
		}),
		users: sync.OnceValue(func() *UserService {
			return &UserService{authn: authn(), hasher: hasher, users: repos.Users()}
		}),
		campaigns: sync.OnceValue(func() *CampaignService {
			return &CampaignService{authn: authn(), campaigns: repos.Campaigns()}
		}),
		campaignViews: sync.OnceValue(func() *CampaignViewService {
			return &CampaignViewService{
				authn:     authn(),
				campaigns: repos.Campaigns(),
				tenants:   repos.Tenants(),
				views:     repos.CampaignViews(),
			}
		}),
	}, nil
}

func (s *Services) Auth() *AuthService                  { return s.auth() }
func (s *Services) Tenants() *TenantService             { return s.tenants() }
func (s *Services) Users() *UserService                 { return s.users() }
func (s *Services) Campaigns() *CampaignService         { return s.campaigns() }
func (s *Services) CampaignViews() *CampaignViewService { return s.campaignViews() }
