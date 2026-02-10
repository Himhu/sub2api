package handler

import (
	"io"
	"log"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// WeChatHandler handles WeChat verification endpoints.
type WeChatHandler struct {
	verifService *service.WeChatVerificationService
	wechatSvc    *service.WeChatService
	settingSvc   *service.SettingService
}

func NewWeChatHandler(
	verifService *service.WeChatVerificationService,
	wechatSvc *service.WeChatService,
	settingSvc *service.SettingService,
) *WeChatHandler {
	return &WeChatHandler{
		verifService: verifService,
		wechatSvc:    wechatSvc,
		settingSvc:   settingSvc,
	}
}

// PostQRCode creates a QR code scene for desktop scanning.
// POST /api/v1/auth/wechat/qrcode
func (h *WeChatHandler) PostQRCode(c *gin.Context) {
	sceneID, qrcodeURL, expireSec, err := h.verifService.CreateQRCodeScene(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"scene_id":       sceneID,
		"qrcode_url":     qrcodeURL,
		"expire_seconds": expireSec,
	})
}

// PostShortCode creates a short code scene for text-based verification.
// POST /api/v1/auth/wechat/shortcode
func (h *WeChatHandler) PostShortCode(c *gin.Context) {
	shortCode, err := h.verifService.CreateShortCodeScene(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"scene_id":   shortCode,
		"short_code": shortCode,
	})
}

// GetScanStatus polls the scan status for a scene.
// GET /api/v1/auth/wechat/scan-status?scene_id=xxx
func (h *WeChatHandler) GetScanStatus(c *gin.Context) {
	sceneID := c.Query("scene_id")
	if sceneID == "" {
		response.BadRequest(c, "scene_id is required")
		return
	}
	status, err := h.verifService.PollScanStatus(c.Request.Context(), sceneID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": status})
}

// GetCallback handles WeChat server URL verification.
// GET /api/v1/auth/wechat/callback?signature=xxx&timestamp=xxx&nonce=xxx&echostr=xxx
func (h *WeChatHandler) GetCallback(c *gin.Context) {
	sig := c.Query("signature")
	ts := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	cfg, err := h.settingSvc.GetWeChatConfig(c.Request.Context())
	if err != nil || cfg.Token == "" {
		c.String(http.StatusForbidden, "not configured")
		return
	}

	if !h.wechatSvc.VerifySignature(cfg.Token, ts, nonce, sig) {
		c.String(http.StatusForbidden, "invalid signature")
		return
	}
	c.String(http.StatusOK, echostr)
}

// PostCallback handles WeChat event push notifications.
// POST /api/v1/auth/wechat/callback
func (h *WeChatHandler) PostCallback(c *gin.Context) {
	sig := c.Query("signature")
	ts := c.Query("timestamp")
	nonce := c.Query("nonce")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	reply, err := h.verifService.HandleCallback(c.Request.Context(), body, sig, ts, nonce)
	if err != nil {
		log.Printf("[WeChatCallback] HandleCallback error: %v", err)
		c.String(http.StatusOK, "success")
		return
	}
	c.String(http.StatusOK, reply)
}
