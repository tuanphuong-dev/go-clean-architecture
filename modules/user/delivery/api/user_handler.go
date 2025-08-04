package api

import (
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/middleware"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	usecase     domain.UserUsecase
	middlewares middleware.Middlewares
}

func NewUserHandler(usecase domain.UserUsecase, middlewares middleware.Middlewares) *UserHandler {
	return &UserHandler{
		usecase:     usecase,
		middlewares: middlewares,
	}
}

func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	user := rg.Group("/users")

	// Apply authentication and rate limiting for user operations
	user.Use(h.middlewares.Authenticator())
	user.Use(h.middlewares.APIRateLimits())

	user.POST("", h.Create)
	user.GET("/:id", h.GetByID)
	user.PUT("/:id", h.Update)
	user.PUT("/:id/password", h.ChangePassword)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req domain.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	user, err := h.usecase.Create(c.Request.Context(), &req)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseCreated(c, user, "User created successfully")
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.usecase.FindByID(c.Request.Context(), id, nil)
	if err != nil || user == nil {
		common.ResponseNotFound(c, "user not found")
		return
	}
	common.ResponseOK(c, user, "User found")
}

func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req domain.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	if err := h.usecase.Update(c.Request.Context(), id, &req); err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseNoContent(c, "User updated successfully")
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	id := c.Param("id")
	var req domain.UserChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequest(c, err.Error())
		return
	}
	req.UserID = id
	if err := h.usecase.ChangePassword(c.Request.Context(), &req); err != nil {
		common.ResponseError(c, err)
		return
	}
	common.ResponseNoContent(c, "Password changed successfully")
}
