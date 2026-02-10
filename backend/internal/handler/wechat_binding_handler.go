package handler

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// WeChatBindingHandler handles WeChat binding requests for logged-in users.
type WeChatBindingHandler struct {
	verifService *service.WeChatVerificationService
}

// NewWeChatBindingHandler creates a new WeChatBindingHandler.
func NewWeChatBindingHandler(verifService *service.WeChatVerificationService) *WeChatBindingHandler {
	return &WeChatBindingHandler{verifService: verifService}
}

// GetStatus returns the WeChat binding status for the current user.
// GET /api/v1/user/wechat/status
func (h *WeChatBindingHandler) GetStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	bound, maskedOpenID, err := h.verifService.GetBindingStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	resp := gin.H{"bound": bound}
	if bound {
		resp["openid_masked"] = maskedOpenID
	}
	response.Success(c, resp)
}

// WeChatBindRequest represents the request to initiate WeChat binding.
type WeChatBindRequest struct {
	Password string `json:"password"`
}

// InitiateBind starts the WeChat binding process by generating a short code.
// POST /api/v1/user/wechat/bind
func (h *WeChatBindingHandler) InitiateBind(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req WeChatBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}

	sceneID, shortCode, err := h.verifService.InitiateBinding(c.Request.Context(), subject.UserID, req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"scene_id":   sceneID,
		"short_code": shortCode,
	})
}

// WeChatBindConfirmRequest represents the request to confirm WeChat binding.
type WeChatBindConfirmRequest struct {
	SceneID string `json:"scene_id" binding:"required"`
	Code    string `json:"code" binding:"required,len=6"`
}

// ConfirmBind completes the WeChat binding with a verification code.
// POST /api/v1/user/wechat/confirm
func (h *WeChatBindingHandler) ConfirmBind(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req WeChatBindConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.verifService.CompleteBinding(c.Request.Context(), subject.UserID, req.SceneID, req.Code); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"success": true})
}

// WeChatUnbindRequest represents the request to unbind WeChat.
type WeChatUnbindRequest struct {
	Password string `json:"password"`
}

// Unbind removes the WeChat binding for the current user.
// POST /api/v1/user/wechat/unbind
func (h *WeChatBindingHandler) Unbind(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req WeChatUnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}

	if err := h.verifService.Unbind(c.Request.Context(), subject.UserID, req.Password); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"success": true})
}
