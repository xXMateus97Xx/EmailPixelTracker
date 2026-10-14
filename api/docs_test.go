package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository/memrepo"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(openAPISpec)
	if err != nil {
		t.Fatalf("loading openapi.yaml: %v", err)
	}
	return doc
}

func TestOpenAPISpecIsValid(t *testing.T) {
	if err := loadSpec(t).Validate(context.Background()); err != nil {
		t.Fatalf("openapi.yaml is invalid: %v", err)
	}
}

var ginParam = regexp.MustCompile(`:(\w+)`)

// Every route served by the API must be documented, and every documented
// operation must exist.
func TestOpenAPICoversAllRoutes(t *testing.T) {
	doc := loadSpec(t)
	router := newClient(t).router

	served := map[string]bool{}
	for _, r := range router.Routes() {
		if strings.HasPrefix(r.Path, "/docs") {
			continue
		}
		path := ginParam.ReplaceAllString(r.Path, "{$1}")
		served[r.Method+" "+path] = true

		item := doc.Paths.Value(path)
		if item == nil || item.GetOperation(r.Method) == nil {
			t.Errorf("route %s %s is not documented in openapi.yaml", r.Method, path)
		}
	}

	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			if !served[method+" "+path] {
				t.Errorf("openapi.yaml documents %s %s, which the API does not serve", method, path)
			}
		}
	}
}

func newDocsRouter(t *testing.T, cfg RouterConfig) (*gin.Engine, error) {
	t.Helper()
	svc, err := service.New(memrepo.New(), service.Config{BcryptCost: bcrypt.MinCost})
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := NewTokenManager("0123456789abcdef0123456789abcdef", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(svc, tokens, cfg)
}

func get(router *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestDocsPages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		docsPath string
		// page and spec are the expected URLs; redirect is requested without
		// the trailing slash and must redirect to page ("" when served at root).
		page, spec, redirect string
	}{
		{"default", "/docs", "/docs/", "/docs/openapi.yaml", "/docs"},
		{"without slashes", "docs", "/docs/", "/docs/openapi.yaml", "/docs"},
		{"nested", "/api-docs/v1/", "/api-docs/v1/", "/api-docs/v1/openapi.yaml", "/api-docs/v1"},
		{"empty is root", "", "/", "/openapi.yaml", ""},
		{"blank is root", "  /  ", "/", "/openapi.yaml", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := newDocsRouter(t, RouterConfig{DocsEnabled: true, DocsPath: tt.docsPath})
			if err != nil {
				t.Fatal(err)
			}

			for url, contentType := range map[string]string{tt.page: "text/html", tt.spec: "application/yaml"} {
				rec := get(router, url)
				if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), contentType) || rec.Body.Len() == 0 {
					t.Errorf("%s: status %d, content type %q, %d bytes", url, rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
				}
			}
			if tt.redirect != "" {
				rec := get(router, tt.redirect)
				if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != tt.page {
					t.Errorf("%s: status %d, location %q; want redirect to %s", tt.redirect, rec.Code, rec.Header().Get("Location"), tt.page)
				}
			}

			// The API keeps working alongside the docs.
			if rec := get(router, "/health/live"); rec.Code != http.StatusOK {
				t.Errorf("/health/live: status %d", rec.Code)
			}
		})
	}
}

// Docs under the prefix of API routes do not shadow them.
func TestDocsNextToApiRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, docsPath := range []string{"api", "pixel"} {
		router, err := newDocsRouter(t, RouterConfig{DocsEnabled: true, DocsPath: docsPath})
		if err != nil {
			t.Fatalf("%s: %v", docsPath, err)
		}
		if rec := get(router, "/"+docsPath+"/"); rec.Code != http.StatusOK {
			t.Errorf("%s: docs page status %d", docsPath, rec.Code)
		}
		if rec := get(router, "/api/campaigns"); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: /api/campaigns status %d, want 401", docsPath, rec.Code)
		}
		if rec := get(router, "/pixel/unknown"); rec.Code != http.StatusNotFound || rec.Body.Len() != 0 {
			t.Errorf("%s: /pixel/unknown status %d, body %q", docsPath, rec.Code, rec.Body.String())
		}
	}
}

func TestDocsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, err := newDocsRouter(t, RouterConfig{DocsEnabled: false, DocsPath: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/docs", "/docs/", "/docs/openapi.yaml", "/", "/openapi.yaml"} {
		if rec := get(router, url); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", url, rec.Code)
		}
	}
}

func TestDocsInvalidPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/docs/:id", "/docs/*all", "/my docs", "/docs?x=1"} {
		if _, err := newDocsRouter(t, RouterConfig{DocsEnabled: true, DocsPath: path}); err == nil {
			t.Errorf("DocsPath %q: expected an error", path)
		}
	}
	// An invalid path is ignored while the docs are disabled.
	if _, err := newDocsRouter(t, RouterConfig{DocsEnabled: false, DocsPath: "/docs/:id"}); err != nil {
		t.Errorf("disabled docs with invalid path: %v", err)
	}
}
