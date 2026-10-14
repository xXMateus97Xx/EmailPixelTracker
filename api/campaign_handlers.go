package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

func (h *handler) listCampaigns(c *gin.Context) {
	page, ok := pageFromQuery(c)
	if !ok {
		return
	}
	result, err := h.svc.Campaigns().List(c.Request.Context(), page)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPageResponse(result, newCampaignResponse))
}

func (h *handler) getCampaign(c *gin.Context) {
	campaign, err := h.svc.Campaigns().GetById(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newCampaignResponse(campaign))
}

func (h *handler) createCampaign(c *gin.Context) {
	input, ok := campaignInput(c, true)
	if !ok {
		return
	}
	campaign, err := h.svc.Campaigns().Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, newCampaignResponse(campaign))
}

func (h *handler) updateCampaign(c *gin.Context) {
	input, ok := campaignInput(c, false)
	if !ok {
		return
	}
	campaign, err := h.svc.Campaigns().Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newCampaignResponse(campaign))
}

func campaignInput(c *gin.Context, creating bool) (service.CampaignInput, bool) {
	var req campaignRequest
	if !bindJSON(c, &req) {
		return service.CampaignInput{}, false
	}
	active, ok := activeFlag(c, req.Active, creating)
	if !ok {
		return service.CampaignInput{}, false
	}
	return service.CampaignInput{
		Name:        req.Name,
		Active:      active,
		ExpireInUtc: req.ExpireInUtc,
		PixelColorR: req.PixelColorR,
		PixelColorG: req.PixelColorG,
		PixelColorB: req.PixelColorB,
		PixelColorA: req.PixelColorA,
	}, true
}
