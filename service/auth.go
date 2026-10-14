package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

// ErrInvalidCredentials is returned for unknown users, wrong passwords and
// inactive users alike, so the response does not reveal which one failed.
var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	authn   authenticator
	hasher  passwordHasher
	users   repository.UserRepository
	tenants repository.TenantRepository
	// dummyHash is compared against when the user does not exist, so a login
	// for an unknown name takes as long as one with a wrong password. It is
	// computed once, on first use, since bcrypt is deliberately slow.
	dummyHash func() (string, error)
}

func newAuthService(authn authenticator, hasher passwordHasher, users repository.UserRepository, tenants repository.TenantRepository) *AuthService {
	return &AuthService{
		authn:     authn,
		hasher:    hasher,
		users:     users,
		tenants:   tenants,
		dummyHash: sync.OnceValues(func() (string, error) { return hasher.hash("dummy-password") }),
	}
}

// Login checks the credentials and returns the principal to be issued to the
// user. Users of an inactive tenant cannot log in.
func (s *AuthService) Login(ctx context.Context, username, password string) (auth.Principal, error) {
	user, err := s.users.GetByUsername(ctx, normalizeUsername(username))
	if errors.Is(err, repository.ErrNotFound) {
		dummyHash, err := s.dummyHash()
		if err != nil {
			return auth.Principal{}, err
		}
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return auth.Principal{}, ErrInvalidCredentials
	}
	if err != nil {
		return auth.Principal{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil || !user.Active {
		return auth.Principal{}, ErrInvalidCredentials
	}

	if user.TenantId != "" {
		tenant, err := s.tenants.GetById(ctx, user.TenantId)
		if err != nil {
			return auth.Principal{}, mapRepoError(err)
		}
		if !tenant.Active {
			return auth.Principal{}, ErrTenantInactive
		}
	}
	return auth.Principal{UserId: user.Id, TenantId: user.TenantId, Role: user.Role}, nil
}

// Me returns the current (active) user and its up-to-date session.
func (s *AuthService) Me(ctx context.Context) (*domain.User, Session, error) {
	session, err := s.authn.current(ctx)
	if err != nil {
		return nil, Session{}, err
	}
	user, err := s.users.GetById(ctx, session.TenantId, session.UserId)
	if err != nil {
		return nil, Session{}, mapRepoError(err)
	}
	return withoutPassword(user), session, nil
}

// Impersonate returns a copy of the current SuperAdmin principal scoped to the
// given tenant. The caller is responsible for persisting it (e.g. in a new
// token). Inactive tenants can be impersonated, but only in read-only mode.
func (s *AuthService) Impersonate(ctx context.Context, tenantId string) (auth.Principal, error) {
	session, err := s.authn.authorize(ctx, domain.SuperAdmin)
	if err != nil {
		return auth.Principal{}, err
	}
	if _, err := s.tenants.GetById(ctx, tenantId); err != nil {
		return auth.Principal{}, mapRepoError(err)
	}
	p := session.Principal
	p.ImpersonatedTenantId = tenantId
	return p, nil
}

func (s *AuthService) StopImpersonation(ctx context.Context) (auth.Principal, error) {
	session, err := s.authn.authorize(ctx, domain.SuperAdmin)
	if err != nil {
		return auth.Principal{}, err
	}
	p := session.Principal
	p.ImpersonatedTenantId = ""
	return p, nil
}

// EnsureSuperAdmin creates the first SuperAdmin when no user with that
// username exists. It bypasses authorization and must only be called at startup.
func (s *AuthService) EnsureSuperAdmin(ctx context.Context, username, name, password string) (created bool, err error) {
	username = normalizeUsername(username)
	name = strings.TrimSpace(name)

	v := &validator{}
	validateUsername(v, username)
	validateUserName(v, name)
	validatePassword(v, password)
	if err := v.err(); err != nil {
		return false, err
	}

	exists, err := s.users.ExistsByUsername(ctx, username, "")
	if err != nil || exists {
		return false, err
	}

	hash, err := s.hasher.hash(password)
	if err != nil {
		return false, err
	}
	user := &domain.User{
		Id:           newId(),
		Name:         name,
		Username:     username,
		Active:       true,
		Password:     hash,
		CreatedAtUtc: nowUtc(),
		Role:         domain.SuperAdmin,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return false, mapRepoError(err)
	}
	return true, nil
}
