package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository/memrepo"
)

type fixture struct {
	repos   *repository.Repositories
	svc     *Services
	tenantA *domain.Tenant
	tenantB *domain.Tenant
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	repos := memrepo.New()
	f := &fixture{repos: repos, svc: newServices(t, repos)}

	var err error
	ctx := f.asSuperAdmin("")
	if f.tenantA, err = f.svc.Tenants().Create(ctx, TenantInput{Name: "Tenant A", Active: true}); err != nil {
		t.Fatal(err)
	}
	if f.tenantB, err = f.svc.Tenants().Create(ctx, TenantInput{Name: "Tenant B", Active: true}); err != nil {
		t.Fatal(err)
	}
	return f
}

func newServices(t *testing.T, repos *repository.Repositories) *Services {
	t.Helper()
	svc, err := New(repos, Config{BcryptCost: bcrypt.MinCost})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// seedUser stores an active user directly in the repository, bypassing the services.
func (f *fixture) seedUser(id, tenantId string, role domain.Role) {
	_ = f.repos.Users().Create(context.Background(), &domain.User{
		Id:       id,
		Name:     id,
		Username: id,
		TenantId: tenantId,
		Active:   true,
		Role:     role,
	})
}

func (f *fixture) asSuperAdmin(impersonatedTenantId string) context.Context {
	f.seedUser("root", "", domain.SuperAdmin)
	return auth.WithPrincipal(context.Background(), auth.Principal{
		UserId:               "root",
		Role:                 domain.SuperAdmin,
		ImpersonatedTenantId: impersonatedTenantId,
	})
}

func (f *fixture) as(role domain.Role, tenantId string) context.Context {
	id := "user-" + string(role) + "-" + tenantId
	f.seedUser(id, tenantId, role)
	return auth.WithPrincipal(context.Background(), auth.Principal{
		UserId:   id,
		TenantId: tenantId,
		Role:     role,
	})
}

func validCampaign(name string) CampaignInput {
	return CampaignInput{
		Name:        name,
		Active:      true,
		ExpireInUtc: time.Now().Add(24 * time.Hour),
		PixelColorA: 1,
	}
}

func TestWritePermissionsByRole(t *testing.T) {
	f := newFixture(t)
	tenant := f.tenantA.Id

	tests := []struct {
		name string
		ctx  context.Context
		// expected errors for creating a tenant, a user and a campaign
		tenantErr, userErr, campaignErr error
	}{
		{"SuperAdmin impersonating", f.asSuperAdmin(tenant), nil, nil, nil},
		{"Admin", f.as(domain.Admin, tenant), ErrForbidden, nil, nil},
		{"ContentManager", f.as(domain.ContentManager, tenant), ErrForbidden, ErrForbidden, nil},
		{"API", f.as(domain.API, tenant), ErrForbidden, ErrForbidden, ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.svc.Tenants().Create(tt.ctx, TenantInput{Name: "New " + tt.name, Active: true})
			assertErr(t, "tenant", err, tt.tenantErr)

			_, err = f.svc.Users().Create(tt.ctx, UserInput{
				Name: "user " + tt.name, Username: strings.ReplaceAll("user-"+tt.name, " ", "-"),
				Password: "secret123", Active: true, Role: domain.ContentManager,
			})
			assertErr(t, "user", err, tt.userErr)

			_, err = f.svc.Campaigns().Create(tt.ctx, validCampaign("campaign "+tt.name))
			assertErr(t, "campaign", err, tt.campaignErr)
		})
	}
}

func TestCampaignsAreIsolatedByTenant(t *testing.T) {
	f := newFixture(t)

	c, err := f.svc.Campaigns().Create(f.as(domain.ContentManager, f.tenantA.Id), validCampaign("Black Friday"))
	if err != nil {
		t.Fatal(err)
	}
	if c.TenantId != f.tenantA.Id {
		t.Fatalf("campaign tenant = %q, want %q", c.TenantId, f.tenantA.Id)
	}

	ctxB := f.as(domain.Admin, f.tenantB.Id)
	if _, err := f.svc.Campaigns().GetById(ctxB, c.Id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetById from other tenant: got %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Campaigns().Update(ctxB, c.Id, validCampaign("hijack")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update from other tenant: got %v, want ErrNotFound", err)
	}
	list, err := f.svc.Campaigns().List(ctxB, Page{})
	if err != nil || list.Total != 0 {
		t.Fatalf("List from other tenant: got %d items, err %v", list.Total, err)
	}

	// The same name is allowed in another tenant.
	if _, err := f.svc.Campaigns().Create(ctxB, validCampaign("Black Friday")); err != nil {
		t.Fatalf("same name in other tenant: %v", err)
	}
	// But not twice in the same tenant.
	_, err = f.svc.Campaigns().Create(f.as(domain.Admin, f.tenantA.Id), validCampaign("Black Friday"))
	assertErr(t, "duplicated campaign", err, ErrValidation)

	// API users can read their tenant's campaigns.
	if _, err := f.svc.Campaigns().GetById(f.as(domain.API, f.tenantA.Id), c.Id); err != nil {
		t.Fatalf("API read: %v", err)
	}
}

func TestSuperAdminNeedsImpersonationForCampaigns(t *testing.T) {
	f := newFixture(t)

	_, err := f.svc.Campaigns().Create(f.asSuperAdmin(""), validCampaign("x"))
	assertErr(t, "without impersonation", err, ErrImpersonationRequired)

	impersonated, err := f.svc.Auth().Impersonate(f.asSuperAdmin(""), f.tenantB.Id)
	if err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithPrincipal(context.Background(), impersonated)
	c, err := f.svc.Campaigns().Create(ctx, validCampaign("x"))
	if err != nil {
		t.Fatal(err)
	}
	if c.TenantId != f.tenantB.Id {
		t.Fatalf("campaign tenant = %q, want %q", c.TenantId, f.tenantB.Id)
	}

	_, err = f.svc.Auth().Impersonate(f.as(domain.Admin, f.tenantA.Id), f.tenantB.Id)
	assertErr(t, "admin impersonating", err, ErrForbidden)

	_, err = f.svc.Auth().Impersonate(f.asSuperAdmin(""), "missing")
	assertErr(t, "missing tenant", err, ErrNotFound)
}

func TestUserRoleRules(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)

	_, err := f.svc.Users().Create(admin, UserInput{Name: "evil", Username: "evil", Password: "secret123", Active: true, Role: domain.SuperAdmin})
	assertErr(t, "admin granting SuperAdmin", err, ErrForbidden)

	_, err = f.svc.Users().Create(admin, UserInput{Name: "bad", Username: "bad", Password: "secret123", Active: true, Role: "Root"})
	assertErr(t, "invalid role", err, ErrValidation)

	_, err = f.svc.Users().Create(admin, UserInput{Name: "no role", Username: "norole", Password: "secret123", Active: true})
	assertErr(t, "missing role", err, ErrValidation)

	_, err = f.svc.Users().Create(admin, UserInput{Name: "short", Username: "short", Password: "123", Active: true, Role: domain.API})
	assertErr(t, "short password", err, ErrValidation)

	// SuperAdmin without impersonation manages global SuperAdmins only.
	root := f.asSuperAdmin("")
	sa, err := f.svc.Users().Create(root, UserInput{Name: "root2", Username: "root2", Password: "secret123", Active: true, Role: domain.SuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if sa.TenantId != "" || sa.Password != "" {
		t.Fatalf("SuperAdmin user: tenant %q, password exposed: %v", sa.TenantId, sa.Password != "")
	}
	_, err = f.svc.Users().Create(root, UserInput{Name: "orphan", Username: "orphan", Password: "secret123", Active: true, Role: domain.Admin})
	assertErr(t, "tenantless admin", err, ErrValidation)

	_, err = f.svc.Users().Create(f.asSuperAdmin(f.tenantA.Id), UserInput{Name: "root3", Username: "root3", Password: "secret123", Active: true, Role: domain.SuperAdmin})
	assertErr(t, "SuperAdmin inside tenant", err, ErrValidation)

	// Admins cannot see SuperAdmins nor users from other tenants.
	if _, err := f.svc.Users().GetById(admin, sa.Id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin reading SuperAdmin: got %v, want ErrNotFound", err)
	}

	u, err := f.svc.Users().Create(admin, UserInput{Name: "maria", Username: "maria", Password: "secret123", Active: true, Role: domain.ContentManager})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Users().Update(f.as(domain.Admin, f.tenantB.Id), u.Id, UserInput{Name: "maria", Username: "maria", Active: true, Role: domain.Admin})
	assertErr(t, "cross-tenant update", err, ErrNotFound)

	_, err = f.svc.Users().Create(f.as(domain.Admin, f.tenantB.Id), UserInput{Name: "maria", Username: "maria", Password: "secret123", Active: true, Role: domain.API})
	assertErr(t, "duplicated username", err, ErrValidation)
}

func TestUsername(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)
	create := func(name, username string) (*domain.User, error) {
		return f.svc.Users().Create(admin, UserInput{
			Name: name, Username: username, Password: "secret123", Active: true, Role: domain.ContentManager,
		})
	}

	for _, username := range []string{"ana maria", "ana\tmaria", "ana\nmaria", "ab", strings.Repeat("a", 51), ""} {
		_, err := create("Ana Maria", username)
		assertErr(t, "username "+strconv.Quote(username), err, ErrValidation)
	}

	// Surrounding spaces are trimmed and the username is stored in lowercase.
	u, err := create("Ana Maria", "  Ana.Maria  ")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "ana.maria" || u.Name != "Ana Maria" {
		t.Fatalf("username %q, name %q", u.Username, u.Name)
	}

	// Usernames are unique regardless of case; names are not unique.
	_, err = create("Other", "ANA.MARIA")
	assertErr(t, "same username, other case", err, ErrValidation)
	if _, err := create("Ana Maria", "ana.maria2"); err != nil {
		t.Fatalf("same name, other username: %v", err)
	}

	// Login uses the username (case-insensitive), not the name.
	if _, err := f.svc.Auth().Login(context.Background(), "ANA.Maria", "secret123"); err != nil {
		t.Fatalf("login by username: %v", err)
	}
	_, err = f.svc.Auth().Login(context.Background(), "Ana Maria", "secret123")
	assertErr(t, "login by name", err, ErrInvalidCredentials)
}

func TestCurrentUserMustExistAndBeActive(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)

	u, err := f.svc.Users().Create(admin, UserInput{Name: "joao", Username: "joao", Password: "secret123", Active: true, Role: domain.ContentManager})
	if err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserId: u.Id, TenantId: f.tenantA.Id, Role: u.Role})
	if _, err := f.svc.Campaigns().List(ctx, Page{}); err != nil {
		t.Fatalf("active user: %v", err)
	}

	// Roles come from the database, not from the principal (e.g. an old token).
	forged := auth.WithPrincipal(context.Background(), auth.Principal{UserId: u.Id, TenantId: f.tenantA.Id, Role: domain.Admin})
	_, err = f.svc.Users().List(forged, Page{})
	assertErr(t, "stale roles", err, ErrForbidden)

	// A principal claiming another tenant does not match the user.
	wrongTenant := auth.WithPrincipal(context.Background(), auth.Principal{UserId: u.Id, TenantId: f.tenantB.Id, Role: u.Role})
	_, err = f.svc.Campaigns().List(wrongTenant, Page{})
	assertErr(t, "wrong tenant", err, ErrUnauthenticated)

	if _, err := f.svc.Users().Update(admin, u.Id, UserInput{Name: "joao", Username: "joao", Active: false, Role: u.Role}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Campaigns().List(ctx, Page{})
	assertErr(t, "inactive user", err, ErrUnauthenticated)

	_, err = f.svc.Auth().Login(context.Background(), "joao", "secret123")
	assertErr(t, "inactive login", err, ErrInvalidCredentials)

	missing := auth.WithPrincipal(context.Background(), auth.Principal{UserId: "ghost", TenantId: f.tenantA.Id, Role: domain.Admin})
	_, err = f.svc.Campaigns().List(missing, Page{})
	assertErr(t, "missing user", err, ErrUnauthenticated)
}

func TestLoginAndBootstrap(t *testing.T) {
	svc := newServices(t, memrepo.New())
	ctx := context.Background()

	created, err := svc.Auth().EnsureSuperAdmin(ctx, "admin", "Admin", "supersecret")
	if err != nil || !created {
		t.Fatalf("EnsureSuperAdmin: created=%v err=%v", created, err)
	}
	if created, _ := svc.Auth().EnsureSuperAdmin(ctx, "admin", "Admin", "supersecret"); created {
		t.Fatal("EnsureSuperAdmin created a second user")
	}

	_, err = svc.Auth().Login(ctx, "admin", "wrong-password")
	assertErr(t, "wrong password", err, ErrInvalidCredentials)
	_, err = svc.Auth().Login(ctx, "nobody", "supersecret")
	assertErr(t, "unknown user", err, ErrInvalidCredentials)

	p, err := svc.Auth().Login(ctx, "admin", "supersecret")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsSuperAdmin() || p.TenantId != "" {
		t.Fatalf("unexpected principal %+v", p)
	}
}

func TestCampaignValidation(t *testing.T) {
	f := newFixture(t)
	ctx := f.as(domain.Admin, f.tenantA.Id)

	in := validCampaign("")
	in.ExpireInUtc = time.Now().Add(-time.Hour)
	in.PixelColorR = 256
	in.PixelColorA = 1.5

	_, err := f.svc.Campaigns().Create(ctx, in)
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("got %v, want ValidationError", err)
	}
	fields := map[string]bool{}
	for _, fe := range verr.Errors {
		fields[fe.Field] = true
	}
	for _, field := range []string{"name", "expireInUtc", "pixelColorR", "pixelColorA"} {
		if !fields[field] {
			t.Errorf("missing validation error for %s (got %v)", field, verr.Errors)
		}
	}
}

func TestInactiveTenant(t *testing.T) {
	f := newFixture(t)
	admin := f.as(domain.Admin, f.tenantA.Id)

	c, err := f.svc.Campaigns().Create(admin, validCampaign("before"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Users().Create(admin, UserInput{Name: "carla", Username: "carla", Password: "secret123", Active: true, Role: domain.API}); err != nil {
		t.Fatal(err)
	}

	root := f.asSuperAdmin("")
	if _, err := f.svc.Tenants().Update(root, f.tenantA.Id, TenantInput{Name: "Tenant A", Active: false}); err != nil {
		t.Fatal(err)
	}

	// Users of the tenant cannot do anything, not even read or log in.
	_, err = f.svc.Campaigns().List(admin, Page{})
	assertErr(t, "tenant user read", err, ErrTenantInactive)
	_, err = f.svc.Campaigns().Create(admin, validCampaign("after"))
	assertErr(t, "tenant user write", err, ErrTenantInactive)
	_, err = f.svc.Tenants().GetById(admin, f.tenantA.Id)
	assertErr(t, "tenant user reads tenant", err, ErrTenantInactive)
	_, _, err = f.svc.Auth().Me(admin)
	assertErr(t, "tenant user me", err, ErrTenantInactive)
	_, err = f.svc.Auth().Login(context.Background(), "carla", "secret123")
	assertErr(t, "tenant user login", err, ErrTenantInactive)

	// Other tenants are not affected.
	if _, err := f.svc.Campaigns().List(f.as(domain.Admin, f.tenantB.Id), Page{}); err != nil {
		t.Fatalf("other tenant: %v", err)
	}

	// A SuperAdmin can impersonate it in read-only mode.
	impersonated, err := f.svc.Auth().Impersonate(root, f.tenantA.Id)
	if err != nil {
		t.Fatal(err)
	}
	imp := auth.WithPrincipal(context.Background(), impersonated)

	if _, session, err := f.svc.Auth().Me(imp); err != nil || !session.ReadOnly {
		t.Fatalf("Me while impersonating inactive tenant: readOnly=%v err=%v", session.ReadOnly, err)
	}
	if _, err := f.svc.Campaigns().GetById(imp, c.Id); err != nil {
		t.Fatalf("read campaign: %v", err)
	}
	if list, err := f.svc.Users().List(imp, Page{}); err != nil || list.Total == 0 {
		t.Fatalf("list users: %v", err)
	}
	_, err = f.svc.Campaigns().Create(imp, validCampaign("after"))
	assertErr(t, "impersonated campaign create", err, ErrTenantInactive)
	_, err = f.svc.Campaigns().Update(imp, c.Id, validCampaign("renamed"))
	assertErr(t, "impersonated campaign update", err, ErrTenantInactive)
	_, err = f.svc.Users().Create(imp, UserInput{Name: "dave", Username: "dave", Password: "secret123", Active: true, Role: domain.API})
	assertErr(t, "impersonated user create", err, ErrTenantInactive)

	// Tenants are global, so the SuperAdmin can reactivate it while impersonating.
	if _, err := f.svc.Tenants().Update(imp, f.tenantA.Id, TenantInput{Name: "Tenant A", Active: true}); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if _, err := f.svc.Campaigns().Create(imp, validCampaign("after")); err != nil {
		t.Fatalf("write after reactivation: %v", err)
	}
	if _, err := f.svc.Campaigns().List(admin, Page{}); err != nil {
		t.Fatalf("tenant user after reactivation: %v", err)
	}
}

func TestUnauthenticated(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Campaigns().List(context.Background(), Page{})
	assertErr(t, "no principal", err, ErrUnauthenticated)

	// A non-SuperAdmin principal without tenant is rejected.
	_, err = f.svc.Campaigns().List(f.as(domain.Admin, ""), Page{})
	assertErr(t, "admin without tenant", err, ErrUnauthenticated)
}

func TestServicesAreLazyAndCreatedOnce(t *testing.T) {
	var tenantsCalls, usersCalls, campaignsCalls int
	repos := repository.NewRepositories(repository.Constructors{
		Tenants:       func() repository.TenantRepository { tenantsCalls++; return memrepo.New().Tenants() },
		Users:         func() repository.UserRepository { usersCalls++; return memrepo.New().Users() },
		Campaigns:     func() repository.CampaignRepository { campaignsCalls++; return memrepo.New().Campaigns() },
		CampaignViews: func() repository.CampaignViewRepository { return memrepo.New().CampaignViews() },
	})
	svc := newServices(t, repos)

	if tenantsCalls+usersCalls+campaignsCalls != 0 {
		t.Fatal("repositories were created by service.New")
	}

	first := svc.Campaigns()
	if svc.Campaigns() != first {
		t.Fatal("Campaigns() returned a different instance")
	}
	if usersCalls != 1 || tenantsCalls != 1 || campaignsCalls != 1 {
		t.Fatalf("constructor calls: users=%d tenants=%d campaigns=%d, want 1 each", usersCalls, tenantsCalls, campaignsCalls)
	}

	// Other services reuse the repositories already created.
	svc.Users()
	svc.Tenants()
	svc.Auth()
	if usersCalls != 1 || tenantsCalls != 1 {
		t.Fatalf("repositories recreated: users=%d tenants=%d", usersCalls, tenantsCalls)
	}
}

func assertErr(t *testing.T, what string, got, want error) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("%s: unexpected error %v", what, got)
		}
		return
	}
	if !errors.Is(got, want) {
		t.Errorf("%s: got error %v, want %v", what, got, want)
	}
}
