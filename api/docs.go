package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// The OpenAPI document and the Swagger UI page are embedded in the binary.
var (
	//go:embed docs/openapi.yaml
	openAPISpec []byte
	//go:embed docs/index.html
	docsPage []byte
)

// Plain path segments only: ':' and '*' would become gin parameters.
var docsPathPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+(/[A-Za-z0-9._~-]+)*$`)

// docsBasePath normalizes the configured path to "/" (application root) or
// "/<path>/". The trailing slash lets the page load the specification with a
// relative URL; gin redirects "/<path>" to "/<path>/".
func docsBasePath(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return "/", nil
	}
	if !docsPathPattern.MatchString(path) {
		return "", fmt.Errorf("invalid docs path %q: use letters, digits, '.', '_', '~', '-' and '/'", path)
	}
	return "/" + path + "/", nil
}

// registerDocs serves the Swagger UI at the base path and the specification
// at <base path>openapi.yaml.
func registerDocs(router *gin.Engine, path string) (err error) {
	base, err := docsBasePath(path)
	if err != nil {
		return err
	}

	// gin panics when a route conflicts with one already registered.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("docs path %q conflicts with another route: %v", base, r)
		}
	}()

	router.GET(base, func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", docsPage)
	})
	router.GET(base+"openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", openAPISpec)
	})
	return nil
}
