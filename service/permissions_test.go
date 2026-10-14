package service

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

func TestPermissionsAndAssignableRoles(t *testing.T) {
	f := newFixture(t)

	inactive, err := f.svc.Tenants().Create(f.asSuperAdmin(""), TenantInput{Name: "Inactive", Active: false})
	if err != nil {
		t.Fatal(err)
	}

	tenantRoles := []domain.Role{domain.Admin, domain.ContentManager, domain.API}
	rw := func(read, write bool) Permission { return Permission{Read: read, Write: write} }
	tests := []struct {
		name       string
		ctx        context.Context
		want       Permissions
		assignable []domain.Role
		tenantId   string
	}{
		{"SuperAdmin", f.asSuperAdmin(""),
			Permissions{rw(true, true), rw(true, true), rw(false, false), true}, []domain.Role{domain.SuperAdmin}, ""},
		{"SuperAdmin impersonating", f.asSuperAdmin(f.tenantA.Id),
			Permissions{rw(true, true), rw(true, true), rw(true, true), true}, tenantRoles, f.tenantA.Id},
		{"SuperAdmin impersonating inactive tenant", f.asSuperAdmin(inactive.Id),
			Permissions{rw(true, true), rw(true, false), rw(true, false), true}, nil, inactive.Id},
		{"Admin", f.as(domain.Admin, f.tenantA.Id),
			Permissions{rw(false, false), rw(true, true), rw(true, true), false}, tenantRoles, f.tenantA.Id},
		{"ContentManager", f.as(domain.ContentManager, f.tenantA.Id),
			Permissions{rw(false, false), rw(false, false), rw(true, true), false}, nil, f.tenantA.Id},
		{"API", f.as(domain.API, f.tenantA.Id),
			Permissions{rw(false, false), rw(false, false), rw(true, false), false}, nil, f.tenantA.Id},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, session, err := f.svc.Auth().Me(tt.ctx)
			if err != nil {
				t.Fatal(err)
			}
			got := session.Permissions()
			if got != tt.want {
				t.Errorf("permissions %+v, want %+v", got, tt.want)
			}
			if (session.Tenant == nil) != (tt.tenantId == "") || (session.Tenant != nil && session.Tenant.Id != tt.tenantId) {
				t.Errorf("session tenant %+v, want %q", session.Tenant, tt.tenantId)
			}

			roles, err := f.svc.Users().Roles(tt.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(roles) != len(domain.AllRoles()) {
				t.Fatalf("got %d roles", len(roles))
			}
			for _, r := range roles {
				if r.Assignable != slices.Contains(tt.assignable, r.Role) {
					t.Errorf("role %s assignable = %v", r.Role, r.Assignable)
				}
			}

			// The flags must match what the services actually accept.
			_, err = f.svc.Tenants().List(tt.ctx, Page{})
			checkAllowed(t, "list tenants", err, got.Tenants.Read)
			_, err = f.svc.Users().List(tt.ctx, Page{})
			checkAllowed(t, "list users", err, got.Users.Read)
			_, err = f.svc.Campaigns().List(tt.ctx, Page{})
			checkAllowed(t, "list campaigns", err, got.Campaigns.Read)
			_, err = f.svc.Campaigns().Create(tt.ctx, validCampaign(fmt.Sprintf("campaign %d", i)))
			checkAllowed(t, "create campaign", err, got.Campaigns.Write)
			for _, role := range domain.AllRoles() {
				username := fmt.Sprintf("user-%d-%s", i, role)
				_, err = f.svc.Users().Create(tt.ctx, UserInput{Name: username, Username: username, Password: "secret123", Active: true, Role: role})
				checkAllowed(t, "create "+role.String(), err, slices.Contains(tt.assignable, role))
			}
		})
	}
}

func checkAllowed(t *testing.T, action string, err error, allowed bool) {
	t.Helper()
	if allowed && err != nil {
		t.Errorf("%s: flagged as allowed but failed: %v", action, err)
	}
	if !allowed && err == nil {
		t.Errorf("%s: flagged as not allowed but succeeded", action)
	}
}
