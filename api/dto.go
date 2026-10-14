package api

import (
	"time"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

type pageResponse[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func toPageResponse[D any, T any](result *service.PagedResult[D], convert func(*D) T) pageResponse[T] {
	items := make([]T, len(result.Items))
	for i := range result.Items {
		items[i] = convert(&result.Items[i])
	}
	return pageResponse[T]{Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}
}

// Auth

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// RecaptchaToken is required when reCAPTCHA is enabled.
	RecaptchaToken string `json:"recaptchaToken,omitempty"`
}

type recaptchaConfigResponse struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"siteKey,omitempty"`
	Action  string `json:"action,omitempty"`
}

type impersonateRequest struct {
	TenantId string `json:"tenantId"`
}

type tokenResponse struct {
	Token        string    `json:"token"`
	ExpiresAtUtc time.Time `json:"expiresAtUtc"`
}

type meResponse struct {
	User                 userResponse `json:"user"`
	ImpersonatedTenantId string       `json:"impersonatedTenantId,omitempty"`
	// ReadOnly is true while impersonating an inactive tenant.
	ReadOnly bool `json:"readOnly"`
	// Tenant is the effective tenant (own or impersonated); absent for a
	// SuperAdmin that is not impersonating.
	Tenant      *tenantResponse     `json:"tenant,omitempty"`
	Permissions permissionsResponse `json:"permissions"`
}

type permissionResponse struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type permissionsResponse struct {
	Tenants     permissionResponse `json:"tenants"`
	Users       permissionResponse `json:"users"`
	Campaigns   permissionResponse `json:"campaigns"`
	Impersonate bool               `json:"impersonate"`
}

func newPermissionsResponse(p service.Permissions) permissionsResponse {
	return permissionsResponse{
		Tenants:     permissionResponse{Read: p.Tenants.Read, Write: p.Tenants.Write},
		Users:       permissionResponse{Read: p.Users.Read, Write: p.Users.Write},
		Campaigns:   permissionResponse{Read: p.Campaigns.Read, Write: p.Campaigns.Write},
		Impersonate: p.Impersonate,
	}
}

type roleResponse struct {
	Name       domain.Role `json:"name"`
	Assignable bool        `json:"assignable"`
}

type rolesResponse struct {
	Items []roleResponse `json:"items"`
}

// Tenants

type tenantRequest struct {
	Name string `json:"name"`
	// Optional on create (defaults to true), required on update.
	Active *bool `json:"active"`
}

type tenantResponse struct {
	Id           string    `json:"id"`
	Name         string    `json:"name"`
	Active       bool      `json:"active"`
	CreatedAtUtc time.Time `json:"createdAtUtc"`
}

func newTenantResponse(t *domain.Tenant) tenantResponse {
	return tenantResponse{Id: t.Id, Name: t.Name, Active: t.Active, CreatedAtUtc: t.CreatedAtUtc}
}

// Users

type userRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	// Required on create; on update an empty value keeps the current password.
	Password string      `json:"password"`
	Active   *bool       `json:"active"`
	Role     domain.Role `json:"role"`
}

type userResponse struct {
	Id           string      `json:"id"`
	Name         string      `json:"name"`
	Username     string      `json:"username"`
	TenantId     string      `json:"tenantId,omitempty"`
	Active       bool        `json:"active"`
	CreatedAtUtc time.Time   `json:"createdAtUtc"`
	Role         domain.Role `json:"role"`
}

func newUserResponse(u *domain.User) userResponse {
	return userResponse{
		Id:           u.Id,
		Name:         u.Name,
		Username:     u.Username,
		TenantId:     u.TenantId,
		Active:       u.Active,
		CreatedAtUtc: u.CreatedAtUtc,
		Role:         u.Role,
	}
}

// Campaigns

type campaignRequest struct {
	Name        string    `json:"name"`
	Active      *bool     `json:"active"`
	ExpireInUtc time.Time `json:"expireInUtc"`
	PixelColorR int       `json:"pixelColorR"`
	PixelColorG int       `json:"pixelColorG"`
	PixelColorB int       `json:"pixelColorB"`
	PixelColorA float32   `json:"pixelColorA"`
}

type campaignResponse struct {
	Id           string    `json:"id"`
	Name         string    `json:"name"`
	TenantId     string    `json:"tenantId"`
	Active       bool      `json:"active"`
	CreatedAtUtc time.Time `json:"createdAtUtc"`
	ExpireInUtc  time.Time `json:"expireInUtc"`
	PixelColorR  int       `json:"pixelColorR"`
	PixelColorG  int       `json:"pixelColorG"`
	PixelColorB  int       `json:"pixelColorB"`
	PixelColorA  float32   `json:"pixelColorA"`
}

// Campaign views

type viewSummaryResponse struct {
	CampaignId  string     `json:"campaignId"`
	FromUtc     *time.Time `json:"fromUtc,omitempty"`
	ToUtc       *time.Time `json:"toUtc,omitempty"`
	TotalViews  int64      `json:"totalViews"`
	UniqueViews int64      `json:"uniqueViews"`
}

type dailyViewsItem struct {
	Date        string `json:"date"` // YYYY-MM-DD, UTC
	TotalViews  int64  `json:"totalViews"`
	UniqueViews int64  `json:"uniqueViews"`
}

type dailyViewsResponse struct {
	CampaignId string           `json:"campaignId"`
	Days       []dailyViewsItem `json:"days"`
}

type viewGroupResponse struct {
	Name        string `json:"name"`
	TotalViews  int64  `json:"totalViews"`
	UniqueViews int64  `json:"uniqueViews"`
}

type userAgentViewsResponse struct {
	UserAgent   string `json:"userAgent"`
	Client      string `json:"client"`
	OS          string `json:"os"`
	Device      string `json:"device"`
	TotalViews  int64  `json:"totalViews"`
	UniqueViews int64  `json:"uniqueViews"`
}

type userAgentReportResponse struct {
	CampaignId       string                   `json:"campaignId"`
	FromUtc          *time.Time               `json:"fromUtc,omitempty"`
	ToUtc            *time.Time               `json:"toUtc,omitempty"`
	Clients          []viewGroupResponse      `json:"clients"`
	OperatingSystems []viewGroupResponse      `json:"operatingSystems"`
	Devices          []viewGroupResponse      `json:"devices"`
	UserAgents       []userAgentViewsResponse `json:"userAgents"`
}

func newViewGroupsResponse(groups []domain.ViewGroup) []viewGroupResponse {
	res := make([]viewGroupResponse, len(groups))
	for i, g := range groups {
		res[i] = viewGroupResponse{Name: g.Name, TotalViews: g.TotalViews, UniqueViews: g.UniqueViews}
	}
	return res
}

func newUserAgentReportResponse(campaignId string, period domain.Period, r *domain.UserAgentReport) userAgentReportResponse {
	res := userAgentReportResponse{
		CampaignId:       campaignId,
		FromUtc:          optionalTime(period.FromUtc),
		ToUtc:            optionalTime(period.ToUtc),
		Clients:          newViewGroupsResponse(r.Clients),
		OperatingSystems: newViewGroupsResponse(r.OperatingSystems),
		Devices:          newViewGroupsResponse(r.Devices),
		UserAgents:       make([]userAgentViewsResponse, len(r.UserAgents)),
	}
	for i, ua := range r.UserAgents {
		res.UserAgents[i] = userAgentViewsResponse{
			UserAgent:   ua.UserAgent,
			Client:      ua.Client,
			OS:          ua.OS,
			Device:      ua.Device,
			TotalViews:  ua.TotalViews,
			UniqueViews: ua.UniqueViews,
		}
	}
	return res
}

type campaignViewResponse struct {
	Id             string    `json:"id"`
	ViewedOnUtc    time.Time `json:"viewedOnUtc"`
	ExternalUserId string    `json:"externalUserId,omitempty"`
	IpAddress      string    `json:"ipAddress"`
	UserAgent      string    `json:"userAgent,omitempty"`
}

func newCampaignViewResponse(v *domain.CampaignView) campaignViewResponse {
	return campaignViewResponse{
		Id:             v.Id,
		ViewedOnUtc:    v.ViewedOnUtc,
		ExternalUserId: v.ExternalUserId,
		IpAddress:      v.IpAddress,
		UserAgent:      v.UserAgent,
	}
}

func newCampaignResponse(c *domain.Campaign) campaignResponse {
	return campaignResponse{
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
