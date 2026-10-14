package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
)

// authenticate requires a valid bearer token and attaches its principal to the
// request context, where the services read it from.
func authenticate(tokens *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
			return
		}

		p, err := tokens.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: err.Error()})
			return
		}

		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), p))
		c.Next()
	}
}
