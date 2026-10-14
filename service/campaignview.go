package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	externalUserIdMaxLength = 200
	ipAddressMaxLength      = 45
	userAgentMaxLength      = 512
	// maxDailyReportDays bounds the size of the daily report.
	maxDailyReportDays = 366
	eachViewBatchSize  = 5000
)

// TrackInput is what the pixel request tells about the view.
type TrackInput struct {
	CampaignId string
	// ExternalUserId is optional.
	ExternalUserId string
	IpAddress      string
	// UserAgent is optional; it is truncated, never rejected.
	UserAgent string
}

// CampaignViewService records the views of the tracking pixel and reports on
// them. Reports have the same access rules as reading a campaign.
type CampaignViewService struct {
	authn     authenticator
	campaigns repository.CampaignRepository
	tenants   repository.TenantRepository
	views     repository.CampaignViewRepository
}

// Track records a view of the campaign pixel. It is public (no user), so
// anything that should not be tracked (an unknown, inactive or expired
// campaign, or one whose tenant is inactive) matches ErrNotFound; the specific
// error (ErrCampaignNotFound, ErrCampaignInactive, ...) tells the reason.
func (s *CampaignViewService) Track(ctx context.Context, input TrackInput) (*domain.Campaign, error) {
	externalUserId := strings.TrimSpace(input.ExternalUserId)

	v := &validator{}
	v.check(utf8.RuneCountInString(externalUserId) <= externalUserIdMaxLength,
		"externalUserId", "must have at most 200 characters")
	v.check(input.IpAddress != "" && len(input.IpAddress) <= ipAddressMaxLength, "ipAddress", "is invalid")
	if err := v.err(); err != nil {
		return nil, err
	}

	campaign, err := s.campaigns.GetByIdAnyTenant(ctx, input.CampaignId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCampaignNotFound
	}
	if err != nil {
		return nil, mapRepoError(err)
	}
	now := nowUtc()
	if !campaign.Active {
		return nil, ErrCampaignInactive
	}
	if !now.Before(campaign.ExpireInUtc) {
		return nil, ErrCampaignExpired
	}

	tenant, err := s.tenants.GetById(ctx, campaign.TenantId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCampaignTenantInactive // a tenant is never deleted; treat as inactive
	}
	if err != nil {
		return nil, mapRepoError(err)
	}
	if !tenant.Active {
		return nil, ErrCampaignTenantInactive
	}

	view := &domain.CampaignView{
		Id:             newId(),
		CampaignId:     campaign.Id,
		ViewedOnUtc:    now,
		ExternalUserId: externalUserId,
		IpAddress:      input.IpAddress,
		UserAgent:      sanitizeUserAgent(input.UserAgent),
	}
	if err := s.views.Create(ctx, view); err != nil {
		return nil, mapRepoError(err)
	}
	return campaign, nil
}

// sanitizeUserAgent makes the client-controlled header safe to store: valid
// UTF-8 (PostgreSQL rejects invalid sequences and NUL), at most
// userAgentMaxLength characters.
func sanitizeUserAgent(userAgent string) string {
	userAgent = strings.ToValidUTF8(userAgent, "�")
	userAgent = strings.ReplaceAll(userAgent, "\x00", "")
	userAgent = strings.TrimSpace(userAgent)
	if utf8.RuneCountInString(userAgent) > userAgentMaxLength {
		userAgent = string([]rune(userAgent)[:userAgentMaxLength])
	}
	return userAgent
}

func (s *CampaignViewService) Summary(ctx context.Context, campaignId string, period domain.Period) (*domain.CampaignViewSummary, error) {
	if err := s.authorizeCampaign(ctx, campaignId, period); err != nil {
		return nil, err
	}
	summary, err := s.views.Summary(ctx, campaignId, period)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return &summary, nil
}

// Daily returns one entry per UTC day of the period, including days without
// views. Without a period start (or end), the first (or last) day with views is used.
func (s *CampaignViewService) Daily(ctx context.Context, campaignId string, period domain.Period) ([]domain.DailyCampaignViews, error) {
	if err := s.authorizeCampaign(ctx, campaignId, period); err != nil {
		return nil, err
	}
	days, err := s.views.DailySummary(ctx, campaignId, period)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return fillMissingDays(days, period)
}

// UserAgents groups the views by e-mail client, operating system, device and
// raw user agent, as classified by the useragent package.
func (s *CampaignViewService) UserAgents(ctx context.Context, campaignId string, period domain.Period) (*domain.UserAgentReport, error) {
	if err := s.authorizeCampaign(ctx, campaignId, period); err != nil {
		return nil, err
	}
	rows, err := s.views.UserAgentVisitors(ctx, campaignId, period)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return buildUserAgentReport(rows), nil
}

func (s *CampaignViewService) List(ctx context.Context, campaignId string, period domain.Period, page Page) (*PagedResult[domain.CampaignView], error) {
	if err := s.authorizeCampaign(ctx, campaignId, period); err != nil {
		return nil, err
	}
	items, total, err := s.views.List(ctx, campaignId, period, page.toPagination())
	if err != nil {
		return nil, mapRepoError(err)
	}
	return newPagedResult(items, total, page), nil
}

// EachView calls fn for every view of the period, from the most recent, and
// stops at the first error fn returns. The views are read in batches, so a
// view tracked meanwhile can shift them: give a period that ends before the
// call to read each view exactly once.
func (s *CampaignViewService) EachView(ctx context.Context, campaignId string, period domain.Period, fn func(domain.CampaignView) error) error {
	if err := s.authorizeCampaign(ctx, campaignId, period); err != nil {
		return err
	}
	page := repository.Pagination{Limit: eachViewBatchSize}
	for {
		items, _, err := s.views.List(ctx, campaignId, period, page)
		if err != nil {
			return mapRepoError(err)
		}
		for _, v := range items {
			if err := fn(v); err != nil {
				return err
			}
		}
		if len(items) < page.Limit {
			return nil
		}
		page.Offset += len(items)
	}
}

// authorizeCampaign checks the read access and that the campaign belongs to
// the current tenant, since views are not stored with a tenant.
func (s *CampaignViewService) authorizeCampaign(ctx context.Context, campaignId string, period domain.Period) error {
	tenantId, err := s.authn.tenantScope(ctx, campaignReaders, false)
	if err != nil {
		return err
	}
	if _, err := s.campaigns.GetById(ctx, tenantId, campaignId); err != nil {
		return mapRepoError(err)
	}

	v := &validator{}
	v.check(period.FromUtc.IsZero() || period.ToUtc.IsZero() || period.FromUtc.Before(period.ToUtc),
		"toUtc", "must be after fromUtc")
	return v.err()
}

func fillMissingDays(days []domain.DailyCampaignViews, period domain.Period) ([]domain.DailyCampaignViews, error) {
	first, last := truncateToDay(period.FromUtc), truncateToDay(period.ToUtc.Add(-time.Nanosecond))
	if period.FromUtc.IsZero() {
		if len(days) == 0 {
			return []domain.DailyCampaignViews{}, nil
		}
		first = days[0].DateUtc
	}
	if period.ToUtc.IsZero() {
		if len(days) == 0 {
			return []domain.DailyCampaignViews{}, nil
		}
		last = days[len(days)-1].DateUtc
	}
	if last.Before(first) {
		return []domain.DailyCampaignViews{}, nil
	}
	if last.Sub(first) >= maxDailyReportDays*24*time.Hour {
		return nil, &ValidationError{Errors: []FieldError{{
			Field: "period", Message: "the daily report covers at most 366 days; narrow fromUtc/toUtc",
		}}}
	}

	byDay := make(map[time.Time]domain.CampaignViewSummary, len(days))
	for _, d := range days {
		byDay[d.DateUtc] = d.CampaignViewSummary
	}

	var filled []domain.DailyCampaignViews
	for day := first; !day.After(last); day = day.AddDate(0, 0, 1) {
		filled = append(filled, domain.DailyCampaignViews{DateUtc: day, CampaignViewSummary: byDay[day]})
	}
	return filled, nil
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
