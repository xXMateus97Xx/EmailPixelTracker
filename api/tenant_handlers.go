package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

func (h *handler) listTenants(c *gin.Context) {
	page, ok := pageFromQuery(c)
	if !ok {
		return
	}
	result, err := h.svc.Tenants().List(c.Request.Context(), page)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPageResponse(result, newTenantResponse))
}

func (h *handler) getTenant(c *gin.Context) {
	tenant, err := h.svc.Tenants().GetById(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newTenantResponse(tenant))
}

func (h *handler) createTenant(c *gin.Context) {
	input, ok := tenantInput(c, true)
	if !ok {
		return
	}
	tenant, err := h.svc.Tenants().Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, newTenantResponse(tenant))
}

func (h *handler) updateTenant(c *gin.Context) {
	input, ok := tenantInput(c, false)
	if !ok {
		return
	}
	tenant, err := h.svc.Tenants().Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newTenantResponse(tenant))
}

func tenantInput(c *gin.Context, creating bool) (service.TenantInput, bool) {
	var req tenantRequest
	if !bindJSON(c, &req) {
		return service.TenantInput{}, false
	}
	active, ok := activeFlag(c, req.Active, creating)
	if !ok {
		return service.TenantInput{}, false
	}
	return service.TenantInput{Name: req.Name, Active: active}, true
}
