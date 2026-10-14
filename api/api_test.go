package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository/memrepo"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

type client struct {
	t      *testing.T
	router *gin.Engine
	// spec checks every JSON response against openapi.yaml.
	spec routers.Router
}

func newClient(t *testing.T) *client {
	t.Helper()
	return newClientWith(t, RouterConfig{})
}

// newClientWith uses cfg, with the docs always enabled at /docs.
func newClientWith(t *testing.T, cfg RouterConfig) *client {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc, err := service.New(memrepo.New(), service.Config{BcryptCost: bcrypt.MinCost})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Auth().EnsureSuperAdmin(context.Background(), "root", "Root", "rootpassword"); err != nil {
		t.Fatal(err)
	}
	tokens, err := NewTokenManager("0123456789abcdef0123456789abcdef", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DocsEnabled, cfg.DocsPath = true, "/docs"
	router, err := NewRouter(svc, tokens, cfg)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := gorillamux.NewRouter(loadSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	return &client{t: t, router: router, spec: spec}
}

// validateResponse fails the test when the response does not match the
// status codes and schemas documented in openapi.yaml.
func (c *client) validateResponse(req *http.Request, rec *httptest.ResponseRecorder) {
	c.t.Helper()
	route, pathParams, err := c.spec.FindRoute(req)
	if err != nil {
		c.t.Fatalf("%s %s: not found in openapi.yaml: %v", req.Method, req.URL.Path, err)
	}
	err = openapi3filter.ValidateResponse(req.Context(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status:  rec.Code,
		Header:  rec.Header(),
		Body:    io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	})
	if err != nil {
		c.t.Fatalf("%s %s: response %d does not match openapi.yaml: %v\nbody: %s",
			req.Method, req.URL.Path, rec.Code, err, rec.Body.String())
	}
}

// do sends the request and decodes the JSON response into out (if not nil).
func (c *client) do(method, path, token string, body any, wantStatus int, out any) {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d; body: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	c.validateResponse(req, rec)
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("%s %s: decoding %s: %v", method, path, rec.Body.String(), err)
		}
	}
}

func (c *client) login(name, password string) string {
	c.t.Helper()
	var res tokenResponse
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: name, Password: password}, http.StatusOK, &res)
	return res.Token
}

func TestApiFlow(t *testing.T) {
	c := newClient(t)

	c.do(http.MethodGet, "/api/campaigns", "", nil, http.StatusUnauthorized, nil)
	c.do(http.MethodGet, "/api/campaigns", "garbage", nil, http.StatusUnauthorized, nil)
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "root", Password: "wrong"}, http.StatusUnauthorized, nil)

	root := c.login("root", "rootpassword")

	var tenant tenantResponse
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusCreated, &tenant)
	if !tenant.Active {
		t.Fatal("tenant should default to active")
	}
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusBadRequest, nil)
	c.do(http.MethodPut, "/api/tenants/"+tenant.Id, root, map[string]any{"name": "Acme"}, http.StatusBadRequest, nil)

	// SuperAdmin must impersonate to work with campaigns.
	campaign := map[string]any{"name": "Launch", "expireInUtc": time.Now().Add(48 * time.Hour), "pixelColorA": 1}
	c.do(http.MethodPost, "/api/campaigns", root, campaign, http.StatusForbidden, nil)

	var imp tokenResponse
	c.do(http.MethodPost, "/api/auth/impersonate", root, impersonateRequest{TenantId: tenant.Id}, http.StatusOK, &imp)

	var me meResponse
	c.do(http.MethodGet, "/api/auth/me", imp.Token, nil, http.StatusOK, &me)
	if me.ImpersonatedTenantId != tenant.Id {
		t.Fatalf("impersonatedTenantId = %q", me.ImpersonatedTenantId)
	}
	if me.Tenant == nil || me.Tenant.Name != "Acme" || !me.Permissions.Campaigns.Write || !me.Permissions.Impersonate {
		t.Fatalf("me %+v", me)
	}

	var roles rolesResponse
	c.do(http.MethodGet, "/api/roles", imp.Token, nil, http.StatusOK, &roles)
	var assignable []string
	for _, r := range roles.Items {
		if r.Assignable {
			assignable = append(assignable, string(r.Name))
		}
	}
	if len(roles.Items) != 4 || strings.Join(assignable, ",") != "Admin,ContentManager,API" {
		t.Fatalf("roles %+v", roles)
	}
	c.do(http.MethodGet, "/api/roles", "", nil, http.StatusUnauthorized, nil)

	var admin userResponse
	c.do(http.MethodPost, "/api/users", imp.Token, map[string]any{
		"name": "acme-admin", "username": "acme-admin", "password": "adminpassword", "role": "Admin",
	}, http.StatusCreated, &admin)
	if admin.TenantId != tenant.Id {
		t.Fatalf("admin tenant = %q, want %q", admin.TenantId, tenant.Id)
	}

	adminToken := c.login("acme-admin", "adminpassword")
	c.do(http.MethodPost, "/api/tenants", adminToken, map[string]any{"name": "Other"}, http.StatusForbidden, nil)

	var created campaignResponse
	c.do(http.MethodPost, "/api/campaigns", adminToken, campaign, http.StatusCreated, &created)

	var page pageResponse[campaignResponse]
	c.do(http.MethodGet, "/api/campaigns?page=1&pageSize=10", imp.Token, nil, http.StatusOK, &page)
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Id != created.Id {
		t.Fatalf("unexpected page %+v", page)
	}
	c.do(http.MethodGet, "/api/campaigns?page=abc", adminToken, nil, http.StatusBadRequest, nil)

	// Validation errors come back per field.
	var verr errorResponse
	c.do(http.MethodPost, "/api/campaigns", adminToken, map[string]any{"name": "Bad", "pixelColorR": 300}, http.StatusBadRequest, &verr)
	if len(verr.Fields) == 0 {
		t.Fatal("expected field errors")
	}

	// API users can read but not write.
	c.do(http.MethodPost, "/api/users", adminToken, map[string]any{
		"name": "acme-api", "username": "acme-api", "password": "apipassword", "role": "API",
	}, http.StatusCreated, nil)
	apiToken := c.login("acme-api", "apipassword")
	c.do(http.MethodGet, "/api/campaigns/"+created.Id, apiToken, nil, http.StatusOK, nil)
	c.do(http.MethodPost, "/api/campaigns", apiToken, campaign, http.StatusForbidden, nil)

	// Deactivating a user invalidates its existing token immediately.
	var apiUser pageResponse[userResponse]
	c.do(http.MethodGet, "/api/users", adminToken, nil, http.StatusOK, &apiUser)
	for _, u := range apiUser.Items {
		if u.Username == "acme-api" {
			c.do(http.MethodPut, "/api/users/"+u.Id, adminToken, map[string]any{
				"name": u.Name, "username": u.Username, "active": false, "role": u.Role,
			}, http.StatusOK, nil)
		}
	}
	c.do(http.MethodGet, "/api/campaigns/"+created.Id, apiToken, nil, http.StatusUnauthorized, nil)
	c.do(http.MethodPost, "/api/auth/login", "", loginRequest{Username: "acme-api", Password: "apipassword"}, http.StatusUnauthorized, nil)
}
