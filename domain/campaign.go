package domain

import "time"

type Campaign struct {
	Id           string
	Name         string
	TenantId     string
	Active       bool
	CreatedAtUtc time.Time
	ExpireInUtc  time.Time
	PixelColorR  int
	PixelColorG  int
	PixelColorB  int
	PixelColorA  float32
}
