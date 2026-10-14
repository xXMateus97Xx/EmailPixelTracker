package memrepo

import (
	"context"
	"slices"
	"time"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type CampaignViewRepository struct{ store[domain.CampaignView] }

func (r *CampaignViewRepository) Create(_ context.Context, v *domain.CampaignView) error {
	r.put(v.Id, *v)
	return nil
}

func inPeriod(v domain.CampaignView, campaignId string, period domain.Period) bool {
	return v.CampaignId == campaignId &&
		(period.FromUtc.IsZero() || !v.ViewedOnUtc.Before(period.FromUtc)) &&
		(period.ToUtc.IsZero() || v.ViewedOnUtc.Before(period.ToUtc))
}

func visitor(v domain.CampaignView) string {
	if v.ExternalUserId != "" {
		return "e:" + v.ExternalUserId
	}
	return "i:" + v.IpAddress
}

func (r *CampaignViewRepository) matching(campaignId string, period domain.Period) []domain.CampaignView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var views []domain.CampaignView
	for _, v := range r.items {
		if inPeriod(v, campaignId, period) {
			views = append(views, v)
		}
	}
	return views
}

func summarize(views []domain.CampaignView) domain.CampaignViewSummary {
	visitors := map[string]bool{}
	for _, v := range views {
		visitors[visitor(v)] = true
	}
	return domain.CampaignViewSummary{TotalViews: int64(len(views)), UniqueViews: int64(len(visitors))}
}

func (r *CampaignViewRepository) Summary(_ context.Context, campaignId string, period domain.Period) (domain.CampaignViewSummary, error) {
	return summarize(r.matching(campaignId, period)), nil
}

func (r *CampaignViewRepository) DailySummary(_ context.Context, campaignId string, period domain.Period) ([]domain.DailyCampaignViews, error) {
	byDay := map[time.Time][]domain.CampaignView{}
	for _, v := range r.matching(campaignId, period) {
		day := v.ViewedOnUtc.UTC().Truncate(24 * time.Hour)
		byDay[day] = append(byDay[day], v)
	}

	days := make([]domain.DailyCampaignViews, 0, len(byDay))
	for day, views := range byDay {
		days = append(days, domain.DailyCampaignViews{DateUtc: day, CampaignViewSummary: summarize(views)})
	}
	slices.SortFunc(days, func(a, b domain.DailyCampaignViews) int { return a.DateUtc.Compare(b.DateUtc) })
	return days, nil
}

func (r *CampaignViewRepository) UserAgentVisitors(_ context.Context, campaignId string, period domain.Period) ([]domain.VisitorUserAgentViews, error) {
	type key struct{ userAgent, visitor string }
	counts := map[key]int64{}
	for _, v := range r.matching(campaignId, period) {
		counts[key{v.UserAgent, visitor(v)}]++
	}

	result := make([]domain.VisitorUserAgentViews, 0, len(counts))
	for k, views := range counts {
		result = append(result, domain.VisitorUserAgentViews{UserAgent: k.userAgent, Visitor: k.visitor, Views: views})
	}
	return result, nil
}

func (r *CampaignViewRepository) List(_ context.Context, campaignId string, period domain.Period, page repository.Pagination) ([]domain.CampaignView, int64, error) {
	items, total := r.list(
		func(v domain.CampaignView) bool { return inPeriod(v, campaignId, period) },
		func(v domain.CampaignView) int64 { return v.ViewedOnUtc.UnixNano() },
		func(v domain.CampaignView) string { return v.Id },
		page)
	return items, total, nil
}
