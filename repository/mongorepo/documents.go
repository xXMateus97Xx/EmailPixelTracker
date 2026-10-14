package mongorepo

import (
	"time"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

type tenantDocument struct {
	Id           string    `bson:"_id"`
	Name         string    `bson:"name"`
	Active       bool      `bson:"active"`
	CreatedAtUtc time.Time `bson:"createdAtUtc"`
}

func newTenantDocument(t *domain.Tenant) tenantDocument {
	return tenantDocument{Id: t.Id, Name: t.Name, Active: t.Active, CreatedAtUtc: t.CreatedAtUtc}
}

func (d tenantDocument) toDomain() domain.Tenant {
	return domain.Tenant{Id: d.Id, Name: d.Name, Active: d.Active, CreatedAtUtc: d.CreatedAtUtc.UTC()}
}

type userDocument struct {
	Id           string      `bson:"_id"`
	Name         string      `bson:"name"`
	Username     string      `bson:"username"`
	TenantId     string      `bson:"tenantId"`
	Active       bool        `bson:"active"`
	Password     string      `bson:"password"`
	CreatedAtUtc time.Time   `bson:"createdAtUtc"`
	Role         domain.Role `bson:"role"`
}

func newUserDocument(u *domain.User) userDocument {
	return userDocument{
		Id:           u.Id,
		Name:         u.Name,
		Username:     u.Username,
		TenantId:     u.TenantId,
		Active:       u.Active,
		Password:     u.Password,
		CreatedAtUtc: u.CreatedAtUtc,
		Role:         u.Role,
	}
}

func (d userDocument) toDomain() domain.User {
	return domain.User{
		Id:           d.Id,
		Name:         d.Name,
		Username:     d.Username,
		TenantId:     d.TenantId,
		Active:       d.Active,
		Password:     d.Password,
		CreatedAtUtc: d.CreatedAtUtc.UTC(),
		Role:         d.Role,
	}
}

type campaignDocument struct {
	Id           string    `bson:"_id"`
	Name         string    `bson:"name"`
	TenantId     string    `bson:"tenantId"`
	Active       bool      `bson:"active"`
	CreatedAtUtc time.Time `bson:"createdAtUtc"`
	ExpireInUtc  time.Time `bson:"expireInUtc"`
	PixelColorR  int       `bson:"pixelColorR"`
	PixelColorG  int       `bson:"pixelColorG"`
	PixelColorB  int       `bson:"pixelColorB"`
	PixelColorA  float32   `bson:"pixelColorA"`
}

func newCampaignDocument(c *domain.Campaign) campaignDocument {
	return campaignDocument{
		Id:           c.Id,
		Name:         c.Name,
		TenantId:     c.TenantId,
		Active:       c.Active,
		CreatedAtUtc: c.CreatedAtUtc,
		ExpireInUtc:  c.ExpireInUtc,
		PixelColorR:  c.PixelColorR,
		PixelColorG:  c.PixelColorG,
		PixelColorB:  c.PixelColorB,
		PixelColorA:  c.PixelColorA,
	}
}

func (d campaignDocument) toDomain() domain.Campaign {
	return domain.Campaign{
		Id:           d.Id,
		Name:         d.Name,
		TenantId:     d.TenantId,
		Active:       d.Active,
		CreatedAtUtc: d.CreatedAtUtc.UTC(),
		ExpireInUtc:  d.ExpireInUtc.UTC(),
		PixelColorR:  d.PixelColorR,
		PixelColorG:  d.PixelColorG,
		PixelColorB:  d.PixelColorB,
		PixelColorA:  d.PixelColorA,
	}
}
