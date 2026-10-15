// Package api exposes the services over HTTP with gin.
package api

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/recaptcha"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

type handler struct {
	svc       *service.Services
	tokens    *TokenManager
	pixels    *pixelCache
	metrics   *telemetry.Metrics
	readiness func(ctx context.Context) error
	recaptcha *recaptcha.Verifier
}

type RouterConfig struct {
	// Metrics records the HTTP and business metrics. Optional.
	Metrics *telemetry.Metrics
	// Readiness checks the dependencies (the database) for /health/ready.
	// Optional; without it the API is always ready.
	Readiness func(ctx context.Context) error
	// Recaptcha protects the login and the password change. Optional; nil disables it.
	Recaptcha *recaptcha.Verifier
	// TrustedProxies are the IPs/CIDRs of the reverse proxies allowed to set
	// X-Forwarded-For. Empty means the client IP is the connection address.
	TrustedProxies []string
	// DocsEnabled registers the Swagger UI and the OpenAPI specification.
	DocsEnabled bool
	// DocsPath is where the docs are served; empty means the application root.
	DocsPath string
}

func NewRouter(svc *service.Services, tokens *TokenManager, cfg RouterConfig) (*gin.Engine, error) {
	h := &handler{
		svc:       svc,
		tokens:    tokens,
		pixels:    &pixelCache{metrics: cfg.Metrics},
		metrics:   cfg.Metrics,
		readiness: cfg.Readiness,
		recaptcha: cfg.Recaptcha,
	}

	router := gin.New()
	router.Use(observability(cfg.Metrics)...)
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}

	router.GET("/health/live", h.live)
	router.GET("/health/ready", h.ready)

	router.GET("/pixel/:campaignId", h.trackPixel)

	public := router.Group("/api")
	public.POST("/auth/login", h.login)
	public.GET("/auth/recaptcha", h.recaptchaConfig)

	private := router.Group("/api", authenticate(tokens))
	private.GET("/auth/me", h.me)
	private.POST("/auth/change-password", h.changePassword)
	private.POST("/auth/impersonate", h.impersonate)
	private.POST("/auth/stop-impersonation", h.stopImpersonation)

	private.GET("/roles", h.listRoles)

	private.GET("/tenants", h.listTenants)
	private.POST("/tenants", h.createTenant)
	private.GET("/tenants/:id", h.getTenant)
	private.PUT("/tenants/:id", h.updateTenant)

	private.GET("/users", h.listUsers)
	private.POST("/users", h.createUser)
	private.GET("/users/:id", h.getUser)
	private.PUT("/users/:id", h.updateUser)

	private.GET("/campaigns", h.listCampaigns)
	private.POST("/campaigns", h.createCampaign)
	private.GET("/campaigns/:id", h.getCampaign)
	private.PUT("/campaigns/:id", h.updateCampaign)
	private.GET("/campaigns/:id/views", h.listCampaignViews)
	private.GET("/campaigns/:id/reports/summary", h.campaignViewSummary)
	private.GET("/campaigns/:id/reports/daily", h.campaignViewDaily)
	private.GET("/campaigns/:id/reports/user-agents", h.campaignViewUserAgents)
	private.GET("/campaigns/:id/reports/export", h.exportCampaignReports)

	// Registered last, so a docs path that clashes with an API route is
	// reported as an error instead of shadowing it.
	if cfg.DocsEnabled {
		if err := registerDocs(router, cfg.DocsPath); err != nil {
			return nil, err
		}
	}

	return router, nil
}

// activeFlag defaults "active" to true on create and requires it on update,
// so a PUT that omits it never silently changes the current value.
func activeFlag(c *gin.Context, active *bool, creating bool) (bool, bool) {
	if active != nil {
		return *active, true
	}
	if creating {
		return true, true
	}
	badRequest(c, service.ErrValidation.Error(), fieldErrorResponse{Field: "active", Message: "is required"})
	return false, false
}
