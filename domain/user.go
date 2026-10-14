package domain

import "time"

type User struct {
	Id string
	// Name is the display name.
	Name string
	// Username is used to log in. It is unique, lowercase and has no spaces.
	Username string
	// TenantId is empty only for SuperAdmin users.
	TenantId     string
	Active       bool
	Password     string
	CreatedAtUtc time.Time
	Role         Role
}
