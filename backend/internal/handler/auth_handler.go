package handler

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	cfg         *config.Config
	authService *service.AuthService
	userService *service.UserService
	settingSvc  *service.SettingService
	totpService *service.TotpService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(cfg *config.Config, authService *service.AuthService, userService *service.UserService, settingService *service.SettingService, totpService *service.TotpService) *AuthHandler {
	return &AuthHandler{
		cfg:         cfg,
		authService: authService,
		userService: userService,
		settingSvc:  settingService,
		totpService: totpService,
	}
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	VerifyCode     string `json:"verify_code"`
	TurnstileToken string `json:"turnstile_token"`
	PromoCode      string `json:"promo_code"` // 注册优惠码
}

// SendVerifyCodeRequest 发送验证码请求
type SendVerifyCodeRequest struct {
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token"`
}

// SendVerifyCodeResponse 发送验证码响应
type SendVerifyCodeResponse struct {
	Message   string `json:"message"`
	Countdown int    `json:"countdown"` // 倒计时秒数
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token"`
}

// AuthResponse 认证响应格式（匹配前端期望）
type AuthResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	User        *dto.User `json:"user"`
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证（当提供了邮箱验证码时跳过，因为发送验证码时已验证过）
	if req.VerifyCode == "" {
		if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	// 邀请注册启用时，邀请码必填
	if h.settingSvc != nil && h.settingSvc.IsInviteRegistrationEnabled(c.Request.Context()) {
		if req.PromoCode == "" {
			response.BadRequest(c, "邀请码不能为空")
			return
		}
	}

	token, user, err := h.authService.RegisterWithVerification(c.Request.Context(), req.Email, req.Password, req.VerifyCode, req.PromoCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        dto.UserFromService(user),
	})
}

// SendVerifyCode 发送邮箱验证码
// POST /api/v1/auth/send-verify-code
func (h *AuthHandler) SendVerifyCode(c *gin.Context) {
	var req SendVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	result, err := h.authService.SendVerifyCodeAsync(c.Request.Context(), req.Email)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, SendVerifyCodeResponse{
		Message:   "Verification code sent successfully",
		Countdown: result.Countdown,
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Check if TOTP 2FA is enabled for this user
	if h.totpService != nil && h.settingSvc != nil && h.settingSvc.IsTotpEnabled(c.Request.Context()) && user.TotpEnabled {
		// Create a temporary login session for 2FA
		tempToken, err := h.totpService.CreateLoginSession(c.Request.Context(), user.ID, user.Email)
		if err != nil {
			response.InternalError(c, "Failed to create 2FA session")
			return
		}

		response.Success(c, TotpLoginResponse{
			Requires2FA:     true,
			TempToken:       tempToken,
			UserEmailMasked: service.MaskEmail(user.Email),
		})
		return
	}

	response.Success(c, AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        dto.UserFromService(user),
	})
}

// TotpLoginResponse represents the response when 2FA is required
type TotpLoginResponse struct {
	Requires2FA     bool   `json:"requires_2fa"`
	TempToken       string `json:"temp_token,omitempty"`
	UserEmailMasked string `json:"user_email_masked,omitempty"`
}

// Login2FARequest represents the 2FA login request
type Login2FARequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	TotpCode  string `json:"totp_code" binding:"required,len=6"`
}

// Login2FA completes the login with 2FA verification
// POST /api/v1/auth/login/2fa
func (h *AuthHandler) Login2FA(c *gin.Context) {
	var req Login2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	slog.Debug("login_2fa_request",
		"temp_token_len", len(req.TempToken),
		"totp_code_len", len(req.TotpCode))

	// Get the login session
	session, err := h.totpService.GetLoginSession(c.Request.Context(), req.TempToken)
	if err != nil || session == nil {
		tokenPrefix := ""
		if len(req.TempToken) >= 8 {
			tokenPrefix = req.TempToken[:8]
		}
		slog.Debug("login_2fa_session_invalid",
			"temp_token_prefix", tokenPrefix,
			"error", err)
		response.BadRequest(c, "Invalid or expired 2FA session")
		return
	}

	slog.Debug("login_2fa_session_found",
		"user_id", session.UserID,
		"email", session.Email)

	// Verify the TOTP code
	if err := h.totpService.VerifyCode(c.Request.Context(), session.UserID, req.TotpCode); err != nil {
		slog.Debug("login_2fa_verify_failed",
			"user_id", session.UserID,
			"error", err)
		response.ErrorFrom(c, err)
		return
	}

	// Delete the login session
	_ = h.totpService.DeleteLoginSession(c.Request.Context(), req.TempToken)

	// Get the user
	user, err := h.userService.GetByID(c.Request.Context(), session.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Generate the JWT token
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.Success(c, AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        dto.UserFromService(user),
	})
}

// GetCurrentUser handles getting current authenticated user
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// 使用 GetByIDWithSubscriptions 加载完整用户数据（含订阅信息）
	// 仅此端点需要订阅数据，其他认证场景使用 GetByID 以提升性能
	user, err := h.userService.GetByIDWithSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	type UserResponse struct {
		*dto.User
		RunMode string `json:"run_mode"`
	}

	runMode := config.RunModeStandard
	if h.cfg != nil {
		runMode = h.cfg.RunMode
	}

	response.Success(c, UserResponse{User: dto.UserFromService(user), RunMode: runMode})
}

// ValidateInviteCodeRequest 验证邀请码请求
type ValidateInviteCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ValidateInviteCodeResponse 验证邀请码响应
type ValidateInviteCodeResponse struct {
	Valid       bool    `json:"valid"`
	BonusAmount float64 `json:"bonus_amount,omitempty"`
	ErrorCode   string  `json:"error_code,omitempty"`
}

// ValidateInviteCode 验证邀请码（公开接口，注册前调用）
// POST /api/v1/auth/validate-invite-code
func (h *AuthHandler) ValidateInviteCode(c *gin.Context) {
	// 检查邀请注册功能是否启用
	if h.settingSvc != nil && !h.settingSvc.IsInviteRegistrationEnabled(c.Request.Context()) {
		response.Success(c, ValidateInviteCodeResponse{
			Valid:     false,
			ErrorCode: "INVITE_CODE_DISABLED",
		})
		return
	}

	var req ValidateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 通过 userService 验证邀请码（大小写不敏感）
	inviteCode := strings.ToUpper(strings.TrimSpace(req.Code))

	// 邀请码格式校验：必须是 16 位十六进制字符
	if !service.IsValidInviteCodeFormat(inviteCode) {
		response.Success(c, ValidateInviteCodeResponse{
			Valid:     false,
			ErrorCode: "INVITE_CODE_INVALID_FORMAT",
		})
		return
	}

	inviter, err := h.userService.GetUserByInviteCode(c.Request.Context(), inviteCode)
	if err != nil {
		// 区分 "not found" 和其他错误
		if errors.Is(err, service.ErrUserNotFound) {
			response.Success(c, ValidateInviteCodeResponse{
				Valid:     false,
				ErrorCode: "INVITE_CODE_NOT_FOUND",
			})
			return
		}
		// 其他错误（如数据库故障）返回服务错误
		response.ErrorFrom(c, err)
		return
	}
	if inviter == nil {
		response.Success(c, ValidateInviteCodeResponse{
			Valid:     false,
			ErrorCode: "INVITE_CODE_NOT_FOUND",
		})
		return
	}

	// 检查邀请人是否激活
	if !inviter.IsActive() {
		response.Success(c, ValidateInviteCodeResponse{
			Valid:     false,
			ErrorCode: "INVITER_NOT_ACTIVE",
		})
		return
	}

	// 获取被邀请人奖励金额
	var bonusAmount float64
	if h.settingSvc != nil {
		bonusAmount = h.settingSvc.GetInviteeBonus(c.Request.Context())
	}

	response.Success(c, ValidateInviteCodeResponse{
		Valid:       true,
		BonusAmount: bonusAmount,
	})
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token"`
}

// ForgotPasswordResponse 忘记密码响应
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// ForgotPassword 请求密码重置
// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Build frontend base URL from request
	scheme := "https"
	if c.Request.TLS == nil {
		// Check X-Forwarded-Proto header (common in reverse proxy setups)
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	frontendBaseURL := scheme + "://" + c.Request.Host

	// Request password reset (async)
	// Note: This returns success even if email doesn't exist (to prevent enumeration)
	if err := h.authService.RequestPasswordResetAsync(c.Request.Context(), req.Email, frontendBaseURL); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, ForgotPasswordResponse{
		Message: "If your email is registered, you will receive a password reset link shortly.",
	})
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordResponse 重置密码响应
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPassword 重置密码
// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Reset password
	if err := h.authService.ResetPassword(c.Request.Context(), req.Email, req.Token, req.NewPassword); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, ResetPasswordResponse{
		Message: "Your password has been reset successfully. You can now log in with your new password.",
	})
}
