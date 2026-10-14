package recaptcha

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeGoogle answers siteverify with the response registered for each token.
func fakeGoogle(t *testing.T, answers map[string]siteVerifyResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.FormValue("secret") != "secret" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		answer, ok := answers[r.FormValue("response")]
		if !ok {
			answer = siteVerifyResponse{ErrorCodes: []string{"invalid-input-response"}}
		}
		_ = json.NewEncoder(w).Encode(answer)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDisabled(t *testing.T) {
	v, err := New(Config{Enabled: false, SiteKey: "ignored"})
	if err != nil || v != nil {
		t.Fatalf("disabled: %v, %v", v, err)
	}
	if v.Enabled() || v.SiteKey() != "" {
		t.Fatal("a nil verifier must be disabled")
	}
	if err := v.Verify(context.Background(), "", LoginAction); err != nil {
		t.Fatalf("disabled verifier rejected: %v", err)
	}
}

func TestConfig(t *testing.T) {
	for name, cfg := range map[string]Config{
		"no secret":  {Enabled: true, SiteKey: "site"},
		"no site":    {Enabled: true, SecretKey: "secret"},
		"high score": {Enabled: true, SiteKey: "site", SecretKey: "secret", MinScore: 1.5},
		"low score":  {Enabled: true, SiteKey: "site", SecretKey: "secret", MinScore: -0.1},
	} {
		if _, err := New(cfg); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	v, err := New(Config{Enabled: true, SiteKey: "site", SecretKey: "secret"})
	if err != nil || !v.Enabled() || v.SiteKey() != "site" || v.minScore != DefaultMinScore {
		t.Fatalf("enabled: %+v, %v", v, err)
	}
}

func TestVerify(t *testing.T) {
	srv := fakeGoogle(t, map[string]siteVerifyResponse{
		"person":       {Success: true, Score: 0.9, Action: LoginAction},
		"borderline":   {Success: true, Score: 0.5, Action: LoginAction},
		"bot":          {Success: true, Score: 0.1, Action: LoginAction},
		"other action": {Success: true, Score: 0.9, Action: "signup"},
		"expired":      {Success: false, ErrorCodes: []string{"timeout-or-duplicate"}},
	})
	v, err := New(Config{Enabled: true, SiteKey: "site", SecretKey: "secret", VerifyURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		token string
		want  error
	}{
		{"person", nil},
		{"borderline", nil},
		{"bot", ErrRejected},
		{"other action", ErrRejected},
		{"expired", ErrRejected},
		{"unknown", ErrRejected},
		{"", ErrMissingToken},
		{"  ", ErrMissingToken},
	}
	for _, tt := range tests {
		err := v.Verify(context.Background(), tt.token, LoginAction)
		if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
			t.Errorf("Verify(%q) = %v, want %v", tt.token, err, tt.want)
		}
	}
}

func TestVerifyUnavailable(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(down.Close)
	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>"))
	}))
	t.Cleanup(garbage.Close)
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	for name, url := range map[string]string{"5xx": down.URL, "not JSON": garbage.URL, "unreachable": closed.URL} {
		v, err := New(Config{Enabled: true, SiteKey: "site", SecretKey: "secret", VerifyURL: url})
		if err != nil {
			t.Fatal(err)
		}
		if err := v.Verify(context.Background(), "token", LoginAction); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: %v, want ErrUnavailable", name, err)
		}
	}
}
