package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/recaptcha"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

// recaptchaConfig tells the login page whether to send a reCAPTCHA token.
func (h *handler) recaptchaConfig(c *gin.Context) {
	res := recaptchaConfigResponse{Enabled: h.recaptcha.Enabled()}
	if res.Enabled {
		res.SiteKey = h.recaptcha.SiteKey()
		res.Action = recaptcha.LoginAction
		res.ChangePasswordAction = recaptcha.ChangePasswordAction
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		h.metrics.Login("invalid_request")
		return
	}
	ctx := c.Request.Context()

	// Checked before the password, so a bot does not get to try passwords
	// (nor make the API spend a bcrypt comparison on them).
	if err := h.recaptcha.Verify(ctx, req.RecaptchaToken, recaptcha.LoginAction); err != nil {
		h.writeRecaptchaError(c, err, "login failed", h.metrics.Login)
		return
	}

	p, err := h.svc.Auth().Login(ctx, req.Username, req.Password)
	result := loginResult(err)
	h.metrics.Login(result)
	if err != nil {
		// The attempted username is not logged: people type passwords in it.
		slog.InfoContext(ctx, "login failed", "reason", result)
		writeError(c, err)
		return
	}
	attrs := []any{"user_id", p.UserId, "role", p.Role.String()}
	if p.TenantId != "" {
		attrs = append(attrs, "tenant_id", p.TenantId)
	}
	slog.InfoContext(ctx, "login succeeded", attrs...)
	h.writeToken(c, p)
}

// writeRecaptchaError answers a failed reCAPTCHA verification. failure is the
// log message and record counts the result in the operation's metric.
func (h *handler) writeRecaptchaError(c *gin.Context, err error, failure string, record func(result string)) {
	ctx := c.Request.Context()
	switch {
	case errors.Is(err, recaptcha.ErrMissingToken):
		record("invalid_request")
		badRequest(c, service.ErrValidation.Error(), fieldErrorResponse{Field: "recaptchaToken", Message: "is required"})
	case errors.Is(err, recaptcha.ErrRejected):
		record("recaptcha_rejected")
		slog.InfoContext(ctx, failure, "reason", "recaptcha_rejected", "detail", err.Error())
		c.JSON(http.StatusForbidden, errorResponse{Error: recaptcha.ErrRejected.Error()})
	default:
		// Without an answer from Google the token cannot be trusted: fail closed.
		record("recaptcha_unavailable")
		slog.WarnContext(ctx, failure, "reason", "recaptcha_unavailable", "error", err)
		c.JSON(http.StatusServiceUnavailable, errorResponse{Error: recaptcha.ErrUnavailable.Error()})
	}
}

func (h *handler) changePassword(c *gin.Context) {
	var req changePasswordRequest
	if !bindJSON(c, &req) {
		h.metrics.PasswordChange("invalid_request")
		return
	}
	ctx := c.Request.Context()

	// As in the login: a bot does not get to guess the current password.
	if err := h.recaptcha.Verify(ctx, req.RecaptchaToken, recaptcha.ChangePasswordAction); err != nil {
		h.writeRecaptchaError(c, err, "password change failed", h.metrics.PasswordChange)
		return
	}

	err := h.svc.Auth().ChangePassword(ctx, req.CurrentPassword, req.NewPassword)
	result := passwordChangeResult(err)
	h.metrics.PasswordChange(result)
	if err != nil {
		if result != "error" {
			slog.InfoContext(ctx, "password change failed", "reason", result)
		}
		writeError(c, err)
		return
	}
	if p, ok := auth.FromContext(ctx); ok {
		slog.InfoContext(ctx, "password changed", "user_id", p.UserId)
	}
	c.Status(http.StatusNoContent)
}

func passwordChangeResult(err error) string {
	var verr *service.ValidationError
	switch {
	case err == nil:
		return "success"
	case errors.As(err, &verr):
		for _, fe := range verr.Errors {
			if fe.Field == "currentPassword" && fe.Message == "is incorrect" {
				return "wrong_password"
			}
		}
		return "invalid_request"
	case errors.Is(err, service.ErrUnauthenticated), errors.Is(err, service.ErrTenantInactive):
		return "unauthenticated"
	default:
		return "error"
	}
}

func loginResult(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, service.ErrInvalidCredentials):
		return "invalid_credentials"
	case errors.Is(err, service.ErrTenantInactive):
		return "tenant_inactive"
	case errors.Is(err, service.ErrValidation):
		return "invalid_request"
	default:
		return "error"
	}
}

func (h *handler) me(c *gin.Context) {
	user, session, err := h.svc.Auth().Me(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	res := meResponse{
		User:                 newUserResponse(user),
		ImpersonatedTenantId: session.ImpersonatedTenantId,
		ReadOnly:             session.ReadOnly,
		Permissions:          newPermissionsResponse(session.Permissions()),
	}
	if session.Tenant != nil {
		tenant := newTenantResponse(session.Tenant)
		res.Tenant = &tenant
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) listRoles(c *gin.Context) {
	roles, err := h.svc.Users().Roles(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	res := rolesResponse{Items: make([]roleResponse, len(roles))}
	for i, r := range roles {
		res.Items[i] = roleResponse{Name: r.Role, Assignable: r.Assignable}
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) impersonate(c *gin.Context) {
	var req impersonateRequest
	if !bindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	p, err := h.svc.Auth().Impersonate(ctx, req.TenantId)
	if err != nil {
		writeError(c, err)
		return
	}
	// SuperAdmins reaching into a tenant is worth a trail in the logs.
	h.metrics.Impersonation("start")
	slog.InfoContext(ctx, "impersonation started", "user_id", p.UserId, "impersonated_tenant_id", p.ImpersonatedTenantId)
	h.writeToken(c, p)
}

func (h *handler) stopImpersonation(c *gin.Context) {
	ctx := c.Request.Context()
	p, err := h.svc.Auth().StopImpersonation(ctx)
	if err != nil {
		writeError(c, err)
		return
	}
	h.metrics.Impersonation("stop")
	slog.InfoContext(ctx, "impersonation stopped", "user_id", p.UserId)
	h.writeToken(c, p)
}

func (h *handler) writeToken(c *gin.Context, p auth.Principal) {
	token, expiresAt, err := h.tokens.Issue(p)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, tokenResponse{Token: token, ExpiresAtUtc: expiresAt})
}
