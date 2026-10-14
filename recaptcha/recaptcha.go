// Package recaptcha verifies Google reCAPTCHA v3 tokens.
//
// The page runs grecaptcha.execute(siteKey, {action}) and sends the token
// along with the protected request; the API asks Google whether the token is
// valid, was made for that action and scored high enough to be a person.
package recaptcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// LoginAction is the action of the tokens made on the login page.
	LoginAction = "login"
	// DefaultMinScore is Google's suggested threshold: 1.0 is very likely a
	// person, 0.0 very likely a bot.
	DefaultMinScore = 0.5

	defaultVerifyURL = "https://www.google.com/recaptcha/api/siteverify"
)

var (
	ErrMissingToken = errors.New("recaptcha token is required")
	// ErrRejected means Google answered and the token did not pass: invalid,
	// expired, already used, made for another action or a low score.
	ErrRejected = errors.New("recaptcha verification failed")
	// ErrUnavailable means Google could not be asked; the token is not trusted.
	ErrUnavailable = errors.New("recaptcha verification unavailable")
)

type Config struct {
	Enabled   bool
	SiteKey   string
	SecretKey string
	// MinScore is the lowest accepted score, from 0 to 1. Zero means DefaultMinScore.
	MinScore float64
	// VerifyURL and HTTPClient are for tests; they default to Google and a
	// client with a 5 second timeout.
	VerifyURL  string
	HTTPClient *http.Client
}

// Verifier is nil-safe: a nil *Verifier is disabled.
type Verifier struct {
	siteKey, secretKey string
	minScore           float64
	verifyURL          string
	client             *http.Client
}

// New returns nil (disabled) when cfg.Enabled is false.
func New(cfg Config) (*Verifier, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.SiteKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("recaptcha: the site key and the secret key are required when it is enabled")
	}
	if cfg.MinScore == 0 {
		cfg.MinScore = DefaultMinScore
	}
	if cfg.MinScore < 0 || cfg.MinScore > 1 {
		return nil, errors.New("recaptcha: the minimum score must be between 0 and 1")
	}
	if cfg.VerifyURL == "" {
		cfg.VerifyURL = defaultVerifyURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Verifier{
		siteKey:   cfg.SiteKey,
		secretKey: cfg.SecretKey,
		minScore:  cfg.MinScore,
		verifyURL: cfg.VerifyURL,
		client:    cfg.HTTPClient,
	}, nil
}

func (v *Verifier) Enabled() bool { return v != nil }

// SiteKey is public: the page needs it to load reCAPTCHA.
func (v *Verifier) SiteKey() string {
	if v == nil {
		return ""
	}
	return v.siteKey
}

type siteVerifyResponse struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify checks a token made for action. It always passes when disabled.
// The errors wrap ErrMissingToken, ErrRejected or ErrUnavailable, with the
// reason (score, error codes) in the message, meant for the logs.
func (v *Verifier) Verify(ctx context.Context, token, action string) error {
	if v == nil {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrMissingToken
	}

	// remoteip is optional and left out: the API is called by the admin
	// server, so the client IP it sees is not the person's.
	form := url.Values{"secret": {v.secretKey}, "response": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUnavailable, res.StatusCode)
	}
	var body siteVerifyResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return fmt.Errorf("%w: decoding the answer: %v", ErrUnavailable, err)
	}

	switch {
	case !body.Success:
		return fmt.Errorf("%w: %s", ErrRejected, strings.Join(body.ErrorCodes, ", "))
	case body.Action != action:
		return fmt.Errorf("%w: token made for action %q, not %q", ErrRejected, body.Action, action)
	case body.Score < v.minScore:
		return fmt.Errorf("%w: score %.1f below %.1f", ErrRejected, body.Score, v.minScore)
	}
	return nil
}
