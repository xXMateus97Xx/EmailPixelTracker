package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

func (h *handler) listUsers(c *gin.Context) {
	page, ok := pageFromQuery(c)
	if !ok {
		return
	}
	result, err := h.svc.Users().List(c.Request.Context(), page)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPageResponse(result, newUserResponse))
}

func (h *handler) getUser(c *gin.Context) {
	user, err := h.svc.Users().GetById(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserResponse(user))
}

func (h *handler) createUser(c *gin.Context) {
	input, ok := userInput(c, true)
	if !ok {
		return
	}
	user, err := h.svc.Users().Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, newUserResponse(user))
}

func (h *handler) updateUser(c *gin.Context) {
	input, ok := userInput(c, false)
	if !ok {
		return
	}
	user, err := h.svc.Users().Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserResponse(user))
}

func userInput(c *gin.Context, creating bool) (service.UserInput, bool) {
	var req userRequest
	if !bindJSON(c, &req) {
		return service.UserInput{}, false
	}
	active, ok := activeFlag(c, req.Active, creating)
	if !ok {
		return service.UserInput{}, false
	}
	return service.UserInput{
		Name:     req.Name,
		Username: req.Username,
		Password: req.Password,
		Active:   active,
		Role:     req.Role,
	}, true
}
