package service

import (
	"context"
	"slices"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

type Permission struct {
	Read  bool
	Write bool
}

// Permissions tells what the session can do, so a client (the admin panel)
// can show only what will be accepted. The services still enforce every rule.
type Permissions struct {
	// Tenants.Read is listing every tenant; any user can read its own tenant.
	Tenants   Permission
	Users     Permission
	Campaigns Permission
	// Impersonate is starting and stopping the impersonation of a tenant.
	Impersonate bool
}

// Permissions mirrors the authorization of the services for this session.
func (s Session) Permissions() Permissions {
	superAdmin := s.IsSuperAdmin()
	inTenant := s.EffectiveTenantId() != ""
	writable := !s.ReadOnly
	return Permissions{
		// Tenants are global, so they stay writable while impersonating an
		// inactive tenant (e.g. to reactivate it).
		Tenants: Permission{Read: superAdmin, Write: superAdmin},
		Users: Permission{
			Read:  s.HasAnyRole(userReaders...),
			Write: s.HasAnyRole(userWriters...) && writable,
		},
		Campaigns: Permission{
			Read:  s.HasAnyRole(campaignReaders...) && inTenant,
			Write: s.HasAnyRole(campaignWriters...) && inTenant && writable,
		},
		Impersonate: superAdmin,
	}
}

// assignableRoles are the roles the session can give to the users it
// manages: SuperAdmin outside a tenant, the tenant roles inside one.
func (s Session) assignableRoles() []domain.Role {
	if !s.Permissions().Users.Write {
		return nil
	}
	if s.EffectiveTenantId() == "" {
		return []domain.Role{domain.SuperAdmin}
	}
	return []domain.Role{domain.Admin, domain.ContentManager, domain.API}
}

type RoleInfo struct {
	Role domain.Role
	// Assignable tells whether the current user can give this role to a user
	// (create or edit) in the current scope.
	Assignable bool
}

// Roles lists every role, flagging the ones the current user can assign.
func (s *UserService) Roles(ctx context.Context) ([]RoleInfo, error) {
	session, err := s.authn.current(ctx)
	if err != nil {
		return nil, err
	}
	assignable := session.assignableRoles()
	roles := domain.AllRoles()
	infos := make([]RoleInfo, len(roles))
	for i, role := range roles {
		infos[i] = RoleInfo{Role: role, Assignable: slices.Contains(assignable, role)}
	}
	return infos, nil
}
