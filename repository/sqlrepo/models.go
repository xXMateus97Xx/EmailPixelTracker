package sqlrepo

import (
	"time"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

type tenantModel struct {
	Id           string    `gorm:"primaryKey;size:36"`
	Name         string    `gorm:"size:200;not null;uniqueIndex"`
	Active       bool      `gorm:"not null"`
	CreatedAtUtc time.Time `gorm:"not null"`
}

func (tenantModel) TableName() string { return "tenants" }

func newTenantModel(t *domain.Tenant) tenantModel {
	return tenantModel{Id: t.Id, Name: t.Name, Active: t.Active, CreatedAtUtc: t.CreatedAtUtc}
}

func (m tenantModel) toDomain() domain.Tenant {
	return domain.Tenant{Id: m.Id, Name: m.Name, Active: m.Active, CreatedAtUtc: m.CreatedAtUtc.UTC()}
}

type userModel struct {
	Id       string `gorm:"primaryKey;size:36"`
	Name     string `gorm:"size:200;not null"`
	Username string `gorm:"size:50;not null;uniqueIndex"`
	// Empty for SuperAdmin users, so it is not a foreign key.
	TenantId     string    `gorm:"size:36;not null;index"`
	Active       bool      `gorm:"not null"`
	Password     string    `gorm:"size:100;not null"`
	CreatedAtUtc time.Time `gorm:"not null"`
	Role         string    `gorm:"size:20;not null"`
}

func (userModel) TableName() string { return "users" }

func newUserModel(u *domain.User) userModel {
	return userModel{
		Id:           u.Id,
		Name:         u.Name,
		Username:     u.Username,
		TenantId:     u.TenantId,
		Active:       u.Active,
		Password:     u.Password,
		CreatedAtUtc: u.CreatedAtUtc,
		Role:         string(u.Role),
	}
}

func (m userModel) toDomain() domain.User {
	return domain.User{
		Id:           m.Id,
		Name:         m.Name,
		Username:     m.Username,
		TenantId:     m.TenantId,
		Active:       m.Active,
		Password:     m.Password,
		CreatedAtUtc: m.CreatedAtUtc.UTC(),
		Role:         domain.Role(m.Role),
	}
}

type campaignModel struct {
	Id           string    `gorm:"primaryKey;size:36"`
	Name         string    `gorm:"size:200;not null;uniqueIndex:ux_campaigns_tenant_name,priority:2"`
	TenantId     string    `gorm:"size:36;not null;uniqueIndex:ux_campaigns_tenant_name,priority:1"`
	Active       bool      `gorm:"not null"`
	CreatedAtUtc time.Time `gorm:"not null"`
	ExpireInUtc  time.Time `gorm:"not null"`
	PixelColorR  int       `gorm:"not null"`
	PixelColorG  int       `gorm:"not null"`
	PixelColorB  int       `gorm:"not null"`
	PixelColorA  float32   `gorm:"not null"`
}

func (campaignModel) TableName() string { return "campaigns" }

func newCampaignModel(c *domain.Campaign) campaignModel {
	return campaignModel{
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

type campaignViewModel struct {
	Id          string    `gorm:"primaryKey;size:36"`
	CampaignId  string    `gorm:"size:36;not null;index:ix_campaign_views_campaign_date,priority:1"`
	ViewedOnUtc time.Time `gorm:"not null;index:ix_campaign_views_campaign_date,priority:2"`
	// NULL when not informed.
	ExternalUserId *string `gorm:"size:200"`
	// 45 characters fit any IPv6 address.
	IpAddress string `gorm:"size:45;not null"`
	UserAgent string `gorm:"size:512;not null;default:''"`
}

func (campaignViewModel) TableName() string { return "campaign_views" }

func newCampaignViewModel(v *domain.CampaignView) campaignViewModel {
	var externalUserId *string
	if v.ExternalUserId != "" {
		externalUserId = &v.ExternalUserId
	}
	return campaignViewModel{
		Id:             v.Id,
		CampaignId:     v.CampaignId,
		ViewedOnUtc:    v.ViewedOnUtc,
		ExternalUserId: externalUserId,
		IpAddress:      v.IpAddress,
		UserAgent:      v.UserAgent,
	}
}

func (m campaignViewModel) toDomain() domain.CampaignView {
	v := domain.CampaignView{
		Id:          m.Id,
		CampaignId:  m.CampaignId,
		ViewedOnUtc: m.ViewedOnUtc.UTC(),
		IpAddress:   m.IpAddress,
		UserAgent:   m.UserAgent,
	}
	if m.ExternalUserId != nil {
		v.ExternalUserId = *m.ExternalUserId
	}
	return v
}

func (m campaignModel) toDomain() domain.Campaign {
	return domain.Campaign{
		Id:           m.Id,
		Name:         m.Name,
		TenantId:     m.TenantId,
		Active:       m.Active,
		CreatedAtUtc: m.CreatedAtUtc.UTC(),
		ExpireInUtc:  m.ExpireInUtc.UTC(),
		PixelColorR:  m.PixelColorR,
		PixelColorG:  m.PixelColorG,
		PixelColorB:  m.PixelColorB,
		PixelColorA:  m.PixelColorA,
	}
}
