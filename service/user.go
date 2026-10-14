package service

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	userNameMinLength = 3
	userNameMaxLength = 200
	usernameMinLength = 3
	usernameMaxLength = 50
	passwordMinLength = 8
	// bcrypt ignores everything after 72 bytes.
	passwordMaxBytes = 72
)

type UserInput struct {
	// Name is the display name.
	Name string
	// Username is used to log in.
	Username string
	// Password is required on create. On update an empty value keeps the current password.
	Password string
	Active   bool
	Role     domain.Role
}

// UserService manages the users of the current tenant. A SuperAdmin that is
// not impersonating a tenant manages the SuperAdmin users (which have no tenant).
type UserService struct {
	authn  authenticator
	hasher passwordHasher
	users  repository.UserRepository
}

func (s *UserService) Create(ctx context.Context, input UserInput) (*domain.User, error) {
	p, err := s.authn.authorizeWrite(ctx, userWriters...)
	if err != nil {
		return nil, err
	}

	tenantId := p.EffectiveTenantId()
	user := &domain.User{
		Id:           newId(),
		Name:         strings.TrimSpace(input.Name),
		Username:     normalizeUsername(input.Username),
		TenantId:     tenantId,
		Active:       input.Active,
		CreatedAtUtc: nowUtc(),
		Role:         input.Role,
	}
	if err := s.validate(ctx, p.Principal, user, input.Password, true); err != nil {
		return nil, err
	}
	if user.Password, err = s.hasher.hash(input.Password); err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, mapRepoError(err)
	}
	return withoutPassword(user), nil
}

func (s *UserService) Update(ctx context.Context, id string, input UserInput) (*domain.User, error) {
	p, err := s.authn.authorizeWrite(ctx, userWriters...)
	if err != nil {
		return nil, err
	}

	tenantId := p.EffectiveTenantId()
	user, err := s.users.GetById(ctx, tenantId, id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	user.Name = strings.TrimSpace(input.Name)
	user.Username = normalizeUsername(input.Username)
	user.Active = input.Active
	user.Role = input.Role

	if err := s.validate(ctx, p.Principal, user, input.Password, false); err != nil {
		return nil, err
	}
	if input.Password != "" {
		if user.Password, err = s.hasher.hash(input.Password); err != nil {
			return nil, err
		}
	}
	if err := s.users.Update(ctx, tenantId, user); err != nil {
		return nil, mapRepoError(err)
	}
	return withoutPassword(user), nil
}

func (s *UserService) GetById(ctx context.Context, id string) (*domain.User, error) {
	p, err := s.authn.authorize(ctx, userReaders...)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetById(ctx, p.EffectiveTenantId(), id)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return withoutPassword(user), nil
}

func (s *UserService) List(ctx context.Context, page Page) (*PagedResult[domain.User], error) {
	p, err := s.authn.authorize(ctx, userReaders...)
	if err != nil {
		return nil, err
	}
	items, total, err := s.users.List(ctx, p.EffectiveTenantId(), page.toPagination())
	if err != nil {
		return nil, mapRepoError(err)
	}
	for i := range items {
		items[i].Password = ""
	}
	return newPagedResult(items, total, page), nil
}

func (s *UserService) validate(ctx context.Context, p auth.Principal, user *domain.User, password string, creating bool) error {
	// Only a SuperAdmin may grant the SuperAdmin role.
	if user.Role == domain.SuperAdmin && !p.IsSuperAdmin() {
		return ErrForbidden
	}

	v := &validator{}

	validateUserName(v, user.Name)
	validateUsername(v, user.Username)
	if creating || password != "" {
		validatePassword(v, password)
	}
	validateRole(v, user)

	if user.Id == p.UserId && !user.Active {
		v.add("active", "you cannot deactivate your own user")
	}

	if err := v.err(); err != nil {
		return err
	}

	taken, err := s.users.ExistsByUsername(ctx, user.Username, user.Id)
	if err != nil {
		return err
	}
	v.check(!taken, "username", "is already in use")
	return v.err()
}

// normalizeUsername makes usernames case-insensitive in every database:
// they are stored and looked up in lowercase.
func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validateUserName(v *validator, name string) {
	nameLength := utf8.RuneCountInString(name)
	v.check(nameLength >= userNameMinLength && nameLength <= userNameMaxLength,
		"name", "must have between 3 and 200 characters")
}

func validateUsername(v *validator, username string) {
	length := utf8.RuneCountInString(username)
	v.check(length >= usernameMinLength && length <= usernameMaxLength,
		"username", "must have between 3 and 50 characters")
	v.check(!strings.ContainsFunc(username, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }),
		"username", "must not contain spaces")
}

func validatePassword(v *validator, password string) {
	v.check(utf8.RuneCountInString(password) >= passwordMinLength, "password", "must have at least 8 characters")
	v.check(len(password) <= passwordMaxBytes, "password", "must have at most 72 bytes")
}

func validateRole(v *validator, user *domain.User) {
	if !user.Role.IsValid() {
		v.add("role", "must be one of SuperAdmin, Admin, ContentManager or API")
		return
	}
	// SuperAdmins are global users; everyone else belongs to a tenant.
	if user.Role == domain.SuperAdmin && user.TenantId != "" {
		v.add("role", "SuperAdmin users cannot belong to a tenant; stop impersonating to manage them")
	}
	if user.Role != domain.SuperAdmin && user.TenantId == "" {
		v.add("role", "users without a tenant must be SuperAdmin; impersonate a tenant to manage its users")
	}
}

func withoutPassword(user *domain.User) *domain.User {
	u := *user
	u.Password = ""
	return &u
}
