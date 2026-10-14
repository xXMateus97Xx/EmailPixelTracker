package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

// trackPixel is public: it records the view and answers the campaign pixel.
// Anything that should not be tracked answers 404 without a body.
func (h *handler) trackPixel(c *gin.Context) {
	ctx := c.Request.Context()
	campaignId := c.Param("campaignId")
	campaign, err := h.svc.CampaignViews().Track(ctx, service.TrackInput{
		CampaignId:     campaignId,
		ExternalUserId: c.Query("externalUserId"),
		IpAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	})
	result := pixelResult(err)
	h.metrics.PixelRequest(result)
	if errors.Is(err, service.ErrNotFound) {
		// The caller only sees 404; the reason answers "why is my pixel not counted?".
		slog.DebugContext(ctx, "pixel view not tracked", "campaign_id", campaignId, "reason", result)
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(c, err)
		return
	}

	img, err := h.pixels.get(campaign)
	if err != nil {
		writeError(c, err)
		return
	}

	// Every open must reach the server to be counted.
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Data(http.StatusOK, "image/png", img)
}

func (h *handler) campaignViewSummary(c *gin.Context) {
	period, ok := periodFromQuery(c)
	if !ok {
		return
	}
	campaignId := c.Param("id")
	summary, err := h.svc.CampaignViews().Summary(c.Request.Context(), campaignId, period)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, viewSummaryResponse{
		CampaignId:  campaignId,
		FromUtc:     optionalTime(period.FromUtc),
		ToUtc:       optionalTime(period.ToUtc),
		TotalViews:  summary.TotalViews,
		UniqueViews: summary.UniqueViews,
	})
}

func (h *handler) campaignViewDaily(c *gin.Context) {
	period, ok := periodFromQuery(c)
	if !ok {
		return
	}
	campaignId := c.Param("id")
	days, err := h.svc.CampaignViews().Daily(c.Request.Context(), campaignId, period)
	if err != nil {
		writeError(c, err)
		return
	}

	res := dailyViewsResponse{CampaignId: campaignId, Days: make([]dailyViewsItem, len(days))}
	for i, d := range days {
		res.Days[i] = dailyViewsItem{
			Date:        d.DateUtc.Format(time.DateOnly),
			TotalViews:  d.TotalViews,
			UniqueViews: d.UniqueViews,
		}
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) campaignViewUserAgents(c *gin.Context) {
	period, ok := periodFromQuery(c)
	if !ok {
		return
	}
	campaignId := c.Param("id")
	report, err := h.svc.CampaignViews().UserAgents(c.Request.Context(), campaignId, period)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserAgentReportResponse(campaignId, period, report))
}

func (h *handler) listCampaignViews(c *gin.Context) {
	period, ok := periodFromQuery(c)
	if !ok {
		return
	}
	page, ok := pageFromQuery(c)
	if !ok {
		return
	}
	result, err := h.svc.CampaignViews().List(c.Request.Context(), c.Param("id"), period, page)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPageResponse(result, newCampaignViewResponse))
}

// periodFromQuery reads the optional ?fromUtc=&toUtc= (RFC 3339) filters.
func periodFromQuery(c *gin.Context) (domain.Period, bool) {
	var period domain.Period
	for _, q := range []struct {
		name string
		dst  *time.Time
	}{{"fromUtc", &period.FromUtc}, {"toUtc", &period.ToUtc}} {
		raw := c.Query(q.name)
		if raw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			badRequest(c, "invalid query parameter", fieldErrorResponse{
				Field: q.name, Message: "must be an RFC 3339 date, e.g. 2026-10-01T00:00:00Z",
			})
			return period, false
		}
		*q.dst = t.UTC()
	}
	return period, true
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
