package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService *service.UserService
	attrService *service.UserAttributeService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService, attrService *service.UserAttributeService) *UserHandler {
	return &UserHandler{
		userService: userService,
		attrService: attrService,
	}
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateProfileRequest represents the update profile request payload
type UpdateProfileRequest struct {
	Username *string `json:"username"`
}

// GetProfile handles getting user profile
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userData, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(userData))
}

// ChangePassword handles changing user password
// POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.ChangePasswordRequest{
		CurrentPassword: req.OldPassword,
		NewPassword:     req.NewPassword,
	}
	err := h.userService.ChangePassword(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// UpdateProfile handles updating user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.UpdateProfileRequest{
		Username: req.Username,
	}
	updatedUser, err := h.userService.UpdateProfile(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(updatedUser))
}

// UpdateUserAttributesRequest represents the update user attributes request payload
type UpdateUserAttributesRequest struct {
	Attributes map[int64]string `json:"attributes" binding:"required"`
}

// GetAttributeDefinitions returns all enabled attribute definitions for users
// GET /api/v1/users/me/attributes/definitions
func (h *UserHandler) GetAttributeDefinitions(c *gin.Context) {
	defs, err := h.attrService.ListDefinitions(c.Request.Context(), true) // only enabled
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Convert to response format
	type AttrDefResponse struct {
		ID          int64                         `json:"id"`
		Key         string                        `json:"key"`
		Name        string                        `json:"name"`
		Description string                        `json:"description"`
		Type        string                        `json:"type"`
		Options     []service.UserAttributeOption `json:"options"`
		Required    bool                          `json:"required"`
		Placeholder string                        `json:"placeholder"`
	}

	out := make([]AttrDefResponse, 0, len(defs))
	for _, def := range defs {
		out = append(out, AttrDefResponse{
			ID:          def.ID,
			Key:         def.Key,
			Name:        def.Name,
			Description: def.Description,
			Type:        string(def.Type),
			Options:     def.Options,
			Required:    def.Required,
			Placeholder: def.Placeholder,
		})
	}

	response.Success(c, out)
}

// GetMyAttributes returns the current user's attribute values
// GET /api/v1/users/me/attributes
func (h *UserHandler) GetMyAttributes(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	values, err := h.attrService.GetUserAttributes(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Convert to map format: attributeID -> value
	out := make(map[int64]string)
	for _, v := range values {
		out[v.AttributeID] = v.Value
	}

	response.Success(c, out)
}

// UpdateMyAttributes updates the current user's attribute values
// PUT /api/v1/users/me/attributes
func (h *UserHandler) UpdateMyAttributes(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateUserAttributesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Convert to service input format
	inputs := make([]service.UpdateUserAttributeInput, 0, len(req.Attributes))
	for attrID, value := range req.Attributes {
		inputs = append(inputs, service.UpdateUserAttributeInput{
			AttributeID: attrID,
			Value:       value,
		})
	}

	if err := h.attrService.UpdateUserAttributes(c.Request.Context(), subject.UserID, inputs); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Attributes updated successfully"})
}
