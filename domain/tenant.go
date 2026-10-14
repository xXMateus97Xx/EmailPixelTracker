package domain

import "time"

type Tenant struct {
	Id           string
	Name         string
	Active       bool
	CreatedAtUtc time.Time
}
