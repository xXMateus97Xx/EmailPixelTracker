package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/recaptcha"
)

func TestChangePassword(t *testing.T) {
	c := newClient(t)
	root := c.login("root", "rootpassword")

	var tenant tenantResponse
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusCreated, &tenant)
	var imp tokenResponse
	c.do(http.MethodPost, "/api/auth/impersonate", root, impersonateRequest{TenantId: tenant.Id}, http.StatusOK, &imp)
	// The API role is read-only, but its users still own their password.
	c.do(http.MethodPost, "/api/users", imp.Token, userRequest{Name: "Integração", Username: "bot", Password: "botpassword", Role: domain.API}, http.StatusCreated, nil)
	bot := c.login("bot", "botpassword")

	change := func(token string, req changePasswordRequest, want int) errorResponse {
		t.Helper()
		var res errorResponse
		out := any(&res)
		if want == http.StatusNoContent || want == http.StatusUnauthorized {
			out = nil
		}
		c.do(http.MethodPost, "/api/auth/change-password", token, req, want, out)
		return res
	}
	field := func(res errorResponse) string {
		if len(res.Fields) != 1 {
			return fmt.Sprintf("%+v", res)
		}
		return res.Fields[0].Field + " " + res.Fields[0].Message
	}

	change("", changePasswordRequest{CurrentPassword: "botpassword", NewPassword: "newpassword"}, http.StatusUnauthorized)
	for _, tc := range []struct {
		req  changePasswordRequest
		want string
	}{
		{changePasswordRequest{NewPassword: "newpassword"}, "currentPassword is required"},
		{changePasswordRequest{CurrentPassword: "botpassword", NewPassword: "short"}, "newPassword must have at least 8 characters"},
		// A wrong password is a field error, not a 401 that would end the session.
		{changePasswordRequest{CurrentPassword: "wrongpassword", NewPassword: "newpassword"}, "currentPassword is incorrect"},
		{changePasswordRequest{CurrentPassword: "botpassword", NewPassword: "botpassword"}, "newPassword must be different from the current password"},
	} {
		if got := field(change(bot, tc.req, http.StatusBadRequest)); got != tc.want {
			t.Errorf("%+v: got %q, want %q", tc.req, got, tc.want)
		}
	}

	change(bot, changePasswordRequest{CurrentPassword: "botpassword", NewPassword: "newpassword"}, http.StatusNoContent)
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "bot", Password: "botpassword"}, http.StatusUnauthorized, nil)
	c.login("bot", "newpassword")
	// The token used for the change keeps working until it expires.
	c.do(http.MethodGet, "/api/auth/me", bot, nil, http.StatusOK, nil)

	// A SuperAdmin impersonating an inactive (read-only) tenant still changes its own password.
	c.do(http.MethodPut, "/api/tenants/"+tenant.Id, root, map[string]any{"name": "Acme", "active": false}, http.StatusOK, nil)
	change(imp.Token, changePasswordRequest{CurrentPassword: "rootpassword", NewPassword: "rootpassword2"}, http.StatusNoContent)
	c.login("root", "rootpassword2")
	// Users of an inactive tenant cannot.
	if res := change(bot, changePasswordRequest{CurrentPassword: "newpassword", NewPassword: "newpassword2"}, http.StatusForbidden); res.Error != "tenant is inactive" {
		t.Errorf("inactive tenant: %+v", res)
	}
}

func TestChangePasswordWithRecaptcha(t *testing.T) {
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.FormValue("response") {
		case "login-token":
			fmt.Fprint(w, `{"success": true, "score": 0.9, "action": "login"}`)
		case "change-token":
			fmt.Fprint(w, `{"success": true, "score": 0.9, "action": "change_password"}`)
		case "down":
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			fmt.Fprint(w, `{"success": false, "error-codes": ["invalid-input-response"]}`)
		}
	}))
	t.Cleanup(google.Close)
	verifier, err := recaptcha.New(recaptcha.Config{Enabled: true, SiteKey: "site-key", SecretKey: "secret", VerifyURL: google.URL})
	if err != nil {
		t.Fatal(err)
	}
	c := newClientWith(t, RouterConfig{Recaptcha: verifier})

	var cfg recaptchaConfigResponse
	c.do(http.MethodGet, "/api/auth/recaptcha", "", nil, http.StatusOK, &cfg)
	if cfg.ChangePasswordAction != "change_password" {
		t.Fatalf("config %+v", cfg)
	}

	var token tokenResponse
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "root", Password: "rootpassword", RecaptchaToken: "login-token"}, http.StatusOK, &token)

	change := func(current, recaptchaToken string, want int) errorResponse {
		t.Helper()
		var res errorResponse
		out := any(&res)
		if want == http.StatusNoContent {
			out = nil
		}
		req := changePasswordRequest{CurrentPassword: current, NewPassword: "newpassword", RecaptchaToken: recaptchaToken}
		c.do(http.MethodPost, "/api/auth/change-password", token.Token, req, want, out)
		return res
	}

	if res := change("rootpassword", "", http.StatusBadRequest); len(res.Fields) != 1 || res.Fields[0].Field != "recaptchaToken" {
		t.Errorf("missing token: %+v", res)
	}
	// A login token cannot be reused for the password change.
	if res := change("rootpassword", "login-token", http.StatusForbidden); res.Error != "recaptcha verification failed" {
		t.Errorf("login token: %+v", res)
	}
	change("rootpassword", "down", http.StatusServiceUnavailable)
	// The captcha passes, the current password is still checked.
	change("wrongpassword", "change-token", http.StatusBadRequest)
	change("rootpassword", "change-token", http.StatusNoContent)
}
