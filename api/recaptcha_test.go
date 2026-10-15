package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xXMateus97Xx/email-pixel-tracker/recaptcha"
)

func TestLoginWithoutRecaptcha(t *testing.T) {
	c := newClient(t)
	var cfg recaptchaConfigResponse
	c.do(http.MethodGet, "/api/auth/recaptcha", "", nil, http.StatusOK, &cfg)
	if cfg != (recaptchaConfigResponse{}) {
		t.Fatalf("disabled config %+v", cfg)
	}
	// The token is ignored.
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "root", Password: "rootpassword", RecaptchaToken: "anything"}, http.StatusOK, nil)
}

func TestLoginWithRecaptcha(t *testing.T) {
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.FormValue("response") {
		case "person":
			fmt.Fprint(w, `{"success": true, "score": 0.9, "action": "login"}`)
		case "bot":
			fmt.Fprint(w, `{"success": true, "score": 0.1, "action": "login"}`)
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
	if cfg != (recaptchaConfigResponse{Enabled: true, SiteKey: "site-key", Action: "login", ChangePasswordAction: "change_password"}) {
		t.Fatalf("enabled config %+v", cfg)
	}

	login := func(password, token string, want int) errorResponse {
		t.Helper()
		var res errorResponse
		out := any(&res)
		if want == http.StatusOK {
			out = nil
		}
		c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "root", Password: password, RecaptchaToken: token}, want, out)
		return res
	}

	if res := login("rootpassword", "", http.StatusBadRequest); len(res.Fields) != 1 || res.Fields[0].Field != "recaptchaToken" {
		t.Errorf("missing token: %+v", res)
	}
	if res := login("rootpassword", "bot", http.StatusForbidden); res.Error != "recaptcha verification failed" {
		t.Errorf("low score: %+v", res)
	}
	login("rootpassword", "expired", http.StatusForbidden)
	if res := login("rootpassword", "down", http.StatusServiceUnavailable); res.Error != "recaptcha verification unavailable" {
		t.Errorf("Google unavailable: %+v", res)
	}
	// The captcha passes, the password is still checked.
	login("wrong", "person", http.StatusUnauthorized)
	login("rootpassword", "person", http.StatusOK)
}
