package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials     = infraerrors.Unauthorized("INVALID_CREDENTIALS", "invalid email or password")
	ErrUserNotActive          = infraerrors.Forbidden("USER_NOT_ACTIVE", "user is not active")
	ErrEmailExists            = infraerrors.Conflict("EMAIL_EXISTS", "email already exists")
	ErrInviteCodeExists       = infraerrors.Conflict("INVITE_CODE_EXISTS", "invite code already exists")
	ErrEmailReserved          = infraerrors.BadRequest("EMAIL_RESERVED", "email is reserved")
	ErrInvalidToken           = infraerrors.Unauthorized("INVALID_TOKEN", "invalid token")
	ErrTokenExpired           = infraerrors.Unauthorized("TOKEN_EXPIRED", "token has expired")
	ErrAccessTokenExpired     = infraerrors.Unauthorized("ACCESS_TOKEN_EXPIRED", "access token has expired")
	ErrTokenTooLarge          = infraerrors.BadRequest("TOKEN_TOO_LARGE", "token too large")
	ErrTokenRevoked           = infraerrors.Unauthorized("TOKEN_REVOKED", "token has been revoked")
	ErrRefreshTokenInvalid    = infraerrors.Unauthorized("REFRESH_TOKEN_INVALID", "invalid refresh token")
	ErrRefreshTokenExpired    = infraerrors.Unauthorized("REFRESH_TOKEN_EXPIRED", "refresh token has expired")
	ErrRefreshTokenReused     = infraerrors.Unauthorized("REFRESH_TOKEN_REUSED", "refresh token has been reused")
	ErrInviteCodeRequired     = infraerrors.BadRequest("INVITE_CODE_REQUIRED", "invite code is required")
	ErrInviteCodeInvalid      = infraerrors.BadRequest("INVITE_CODE_INVALID", "invite code is invalid")
	ErrInviteCodeFormat       = infraerrors.BadRequest("INVITE_CODE_INVALID_FORMAT", "invite code must be 16 hex characters")
	ErrRegDisabled            = infraerrors.Forbidden("REGISTRATION_DISABLED", "registration is currently disabled")
	ErrServiceUnavailable     = infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "service temporarily unavailable")
	ErrWeChatAlreadyBound     = infraerrors.Conflict("WECHAT_ALREADY_BOUND", "this wechat account is already bound to another user")
	ErrWeChatBindingNotFound  = infraerrors.BadRequest("WECHAT_BINDING_NOT_FOUND", "no wechat binding found for this account")
	ErrPasswordResetFailed    = infraerrors.BadRequest("PASSWORD_RESET_FAILED", "password reset failed")
)

// maxTokenLength 限制 token 大小，避免超长 header 触发解析时的异常内存分配。
const maxTokenLength = 8192

// inviteCodeLen 邀请码长度（16位十六进制字符）
const inviteCodeLen = 16

// IsValidInviteCodeFormat 验证邀请码格式（16位十六进制字符，大写）
func IsValidInviteCodeFormat(code string) bool {
	if len(code) != inviteCodeLen {
		return false
	}
	for _, c := range code {
		if !((c >= 'A' && c <= 'F') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// refreshTokenPrefix is the prefix for refresh tokens to distinguish them from access tokens.
const refreshTokenPrefix = "rt_"

// JWTClaims JWT载荷数据
type JWTClaims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"` // Used to invalidate tokens on password change
	jwt.RegisteredClaims
}

// AuthService 认证服务
type AuthService struct {
	userRepo               UserRepository
	redeemRepo             RedeemCodeRepository
	refreshTokenCache      RefreshTokenCache
	cfg                    *config.Config
	settingService         *SettingService
	turnstileService       *TurnstileService
	wechatVerifService     *WeChatVerificationService
	wechatBindingRepo      WeChatBindingRepository
	wechatBindingHistoryRepo WeChatBindingHistoryRepository
}

// NewAuthService 创建认证服务实例
func NewAuthService(
	userRepo UserRepository,
	redeemRepo RedeemCodeRepository,
	refreshTokenCache RefreshTokenCache,
	cfg *config.Config,
	settingService *SettingService,
	turnstileService *TurnstileService,
	wechatVerifService *WeChatVerificationService,
	wechatBindingRepo WeChatBindingRepository,
	wechatBindingHistoryRepo WeChatBindingHistoryRepository,
) *AuthService {
	return &AuthService{
		userRepo:                 userRepo,
		redeemRepo:               redeemRepo,
		refreshTokenCache:        refreshTokenCache,
		cfg:                      cfg,
		settingService:           settingService,
		turnstileService:         turnstileService,
		wechatVerifService:       wechatVerifService,
		wechatBindingRepo:        wechatBindingRepo,
		wechatBindingHistoryRepo: wechatBindingHistoryRepo,
	}
}

// Register 用户注册，返回token和用户
func (s *AuthService) Register(ctx context.Context, email, password string) (string, *User, error) {
	return s.RegisterWithVerification(ctx, email, password, "", "")
}

// RegisterWithVerification 用户注册（支持推荐码），返回token和用户
func (s *AuthService) RegisterWithVerification(ctx context.Context, email, password, verifyCode, promoCode string) (string, *User, error) {
	// 检查是否开放注册（默认关闭：settingService 未配置时不允许注册）
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return "", nil, ErrRegDisabled
	}

	// 防止用户注册 LinuxDo OAuth 合成邮箱，避免第三方登录与本地账号发生碰撞。
	if isReservedEmail(email) {
		return "", nil, ErrEmailReserved
	}

	// 邀请注册启用时，校验推荐码
	var inviter *User
	if s.settingService != nil && s.settingService.IsInviteRegistrationEnabled(ctx) {
		inviteCode := strings.ToUpper(strings.TrimSpace(promoCode))
		if inviteCode == "" {
			return "", nil, ErrInviteCodeRequired
		}
		// 格式验证：6-32位字母数字
		if !IsValidInviteCodeFormat(inviteCode) {
			return "", nil, ErrInviteCodeFormat
		}
		var inviteErr error
		inviter, inviteErr = s.userRepo.GetByInviteCode(ctx, inviteCode)
		if inviteErr != nil {
			if errors.Is(inviteErr, ErrUserNotFound) {
				return "", nil, ErrInviteCodeInvalid
			}
			log.Printf("[Auth] Database error checking invite code: %v", inviteErr)
			return "", nil, ErrServiceUnavailable
		}
		if inviter == nil || !inviter.IsActive() {
			return "", nil, ErrInviteCodeInvalid
		}
	}

	// 检查邮箱是否已存在
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		log.Printf("[Auth] Database error checking email exists: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	if existsEmail {
		return "", nil, ErrEmailExists
	}

	// 密码哈希
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	// 获取默认配置
	defaultBalance := s.cfg.Default.UserBalance
	defaultConcurrency := s.cfg.Default.UserConcurrency
	if s.settingService != nil {
		defaultBalance = s.settingService.GetDefaultBalance(ctx)
		defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
	}

	// 为新用户生成唯一邀请码
	inviteCode, err := s.generateUniqueInviteCode(ctx)
	if err != nil {
		log.Printf("[Auth] Failed to generate invite code: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	// 创建用户
	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         RoleUser,
		Balance:      defaultBalance,
		Concurrency:  defaultConcurrency,
		Status:       StatusActive,
		InviteCode:   &inviteCode,
	}
	// 设置邀请人和所属代理
	if inviter != nil {
		inviterID := inviter.ID
		user.InvitedByUserID = &inviterID
		// 设置所属代理ID
		if inviter.IsAgent {
			// 邀请人是代理 → 直接设置为邀请人
			user.BelongAgentID = &inviterID
		} else if inviter.BelongAgentID != nil {
			// 邀请人不是代理 → 继承邀请人的所属代理
			user.BelongAgentID = inviter.BelongAgentID
		}
	}

	if err := s.createUserWithInviteCodeRetry(ctx, user); err != nil {
		if errors.Is(err, ErrEmailExists) {
			return "", nil, ErrEmailExists
		}
		log.Printf("[Auth] Database error creating user: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	// 邀请奖励发放逻辑
	if inviter != nil && s.settingService != nil {
		// 邀请人奖励：只有非代理用户邀请时才发放（代理是推广渠道，不获得邀请奖励）
		if !inviter.IsAgent {
			inviterBonus := s.settingService.GetInviterBonus(ctx)
			if inviterBonus > 0 {
				if err := s.userRepo.AddPoints(ctx, inviter.ID, inviterBonus); err != nil {
					log.Printf("[Auth] Failed to add inviter bonus for user %d: %v", inviter.ID, err)
				}
			}
		}
		// 被邀请人奖励：始终发放
		inviteeBonus := s.settingService.GetInviteeBonus(ctx)
		if inviteeBonus > 0 {
			if err := s.userRepo.AddPoints(ctx, user.ID, inviteeBonus); err != nil {
				log.Printf("[Auth] Failed to add invitee bonus for user %d: %v", user.ID, err)
			} else {
				// 更新本地用户对象的积分，以便返回正确的值
				user.Points += inviteeBonus
			}
		}
	}

	// 生成token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}

// RegisterWithWeChatVerification 用户注册（微信验证码），返回token和用户
func (s *AuthService) RegisterWithWeChatVerification(ctx context.Context, email, password, sceneID, verifyCode string, promoCode string) (string, *User, error) {
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return "", nil, ErrRegDisabled
	}
	if isReservedEmail(email) {
		return "", nil, ErrEmailReserved
	}
	if s.wechatVerifService == nil || s.wechatBindingRepo == nil || s.wechatBindingHistoryRepo == nil {
		log.Println("[Auth] WeChat verification not configured")
		return "", nil, ErrServiceUnavailable
	}

	// 校验微信验证码 → 获取 openid
	openid, err := s.wechatVerifService.ValidateAndConsumeCode(ctx, sceneID, verifyCode)
	if err != nil {
		return "", nil, err
	}

	// 获取 appID
	cfg, err := s.settingService.GetWeChatConfig(ctx)
	if err != nil {
		log.Printf("[Auth] Failed to get wechat config: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	if cfg.AppID == "" {
		return "", nil, ErrServiceUnavailable
	}

	// 检查 openid 是否已绑定
	binding, err := s.wechatBindingRepo.GetByOpenID(ctx, cfg.AppID, openid)
	if err != nil {
		log.Printf("[Auth] Database error checking wechat binding: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	if binding != nil {
		return "", nil, ErrWeChatAlreadyBound
	}
	// Tombstone check: block OpenID that was previously bound to any account
	if history, err := s.wechatBindingHistoryRepo.GetByOpenID(ctx, cfg.AppID, openid); err != nil {
		log.Printf("[Auth] Database error checking wechat binding history: %v", err)
		return "", nil, ErrServiceUnavailable
	} else if history != nil {
		return "", nil, ErrWeChatAlreadyBound
	}

	// 检查推荐码（invite registration 系统）
	var inviter *User
	if s.settingService.IsInviteRegistrationEnabled(ctx) {
		inviteCode := strings.ToUpper(strings.TrimSpace(promoCode))
		if inviteCode == "" {
			return "", nil, ErrInviteCodeRequired
		}
		if !IsValidInviteCodeFormat(inviteCode) {
			return "", nil, ErrInviteCodeFormat
		}
		inviter, err = s.userRepo.GetByInviteCode(ctx, inviteCode)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return "", nil, ErrInviteCodeInvalid
			}
			return "", nil, ErrServiceUnavailable
		}
		if inviter == nil || !inviter.IsActive() {
			return "", nil, ErrInviteCodeInvalid
		}
	}

	// 检查邮箱
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrServiceUnavailable
	}
	if existsEmail {
		return "", nil, ErrEmailExists
	}

	// 创建用户
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	defaultBalance := s.settingService.GetDefaultBalance(ctx)
	defaultConcurrency := s.settingService.GetDefaultConcurrency(ctx)

	newInviteCode, err := s.generateUniqueInviteCode(ctx)
	if err != nil {
		return "", nil, ErrServiceUnavailable
	}

	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         RoleUser,
		Balance:      defaultBalance,
		Concurrency:  defaultConcurrency,
		Status:       StatusActive,
		InviteCode:   &newInviteCode,
	}
	if inviter != nil {
		inviterID := inviter.ID
		user.InvitedByUserID = &inviterID
		if inviter.IsAgent {
			user.BelongAgentID = &inviterID
		} else if inviter.BelongAgentID != nil {
			user.BelongAgentID = inviter.BelongAgentID
		}
	}

	if err := s.createUserWithInviteCodeRetry(ctx, user); err != nil {
		if errors.Is(err, ErrEmailExists) {
			return "", nil, ErrEmailExists
		}
		return "", nil, ErrServiceUnavailable
	}

	// 创建微信绑定
	if err := s.wechatBindingRepo.Create(ctx, user.ID, cfg.AppID, openid); err != nil {
		log.Printf("[Auth] Failed to create wechat binding for user %d: %v", user.ID, err)
		// 回滚：删除已创建的用户，避免无绑定的孤立账号
		if delErr := s.userRepo.Delete(ctx, user.ID); delErr != nil {
			log.Printf("[Auth] Failed to rollback user %d after binding failure: %v", user.ID, delErr)
		}
		return "", nil, ErrServiceUnavailable
	}

	// 邀请奖励
	if inviter != nil {
		if !inviter.IsAgent {
			if bonus := s.settingService.GetInviterBonus(ctx); bonus > 0 {
				_ = s.userRepo.AddPoints(ctx, inviter.ID, bonus)
			}
		}
		if bonus := s.settingService.GetInviteeBonus(ctx); bonus > 0 {
			if err := s.userRepo.AddPoints(ctx, user.ID, bonus); err == nil {
				user.Points += bonus
			}
		}
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	return token, user, nil
}

// VerifyTurnstile 验证Turnstile token
func (s *AuthService) VerifyTurnstile(ctx context.Context, token string, remoteIP string) error {
	required := s.cfg != nil && s.cfg.Server.Mode == "release" && s.cfg.Turnstile.Required

	if required {
		if s.settingService == nil {
			log.Println("[Auth] Turnstile required but settings service is not configured")
			return ErrTurnstileNotConfigured
		}
		enabled := s.settingService.IsTurnstileEnabled(ctx)
		secretConfigured := s.settingService.GetTurnstileSecretKey(ctx) != ""
		if !enabled || !secretConfigured {
			log.Printf("[Auth] Turnstile required but not configured (enabled=%v, secret_configured=%v)", enabled, secretConfigured)
			return ErrTurnstileNotConfigured
		}
	}

	if s.turnstileService == nil {
		if required {
			log.Println("[Auth] Turnstile required but service not configured")
			return ErrTurnstileNotConfigured
		}
		return nil // 服务未配置则跳过验证
	}

	if !required && s.settingService != nil && s.settingService.IsTurnstileEnabled(ctx) && s.settingService.GetTurnstileSecretKey(ctx) == "" {
		log.Println("[Auth] Turnstile enabled but secret key not configured")
	}

	return s.turnstileService.VerifyToken(ctx, token, remoteIP)
}

// IsTurnstileEnabled 检查是否启用Turnstile验证
func (s *AuthService) IsTurnstileEnabled(ctx context.Context) bool {
	if s.turnstileService == nil {
		return false
	}
	return s.turnstileService.IsEnabled(ctx)
}

// IsRegistrationEnabled 检查是否开放注册
func (s *AuthService) IsRegistrationEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false // 安全默认：settingService 未配置时关闭注册
	}
	return s.settingService.IsRegistrationEnabled(ctx)
}

// Login 用户登录，返回JWT token
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *User, error) {
	// 查找用户
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		// 记录数据库错误但不暴露给用户
		log.Printf("[Auth] Database error during login: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	// 验证密码
	if !s.CheckPassword(password, user.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 生成JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}

// LoginOrRegisterOAuth 用于第三方 OAuth/SSO 登录：
// - 如果邮箱已存在：直接登录（不需要本地密码）
// - 如果邮箱不存在：创建新用户并登录
//
// 注意：该函数用于 LinuxDo OAuth 登录场景（不同于上游账号的 OAuth，例如 Claude/OpenAI/Gemini）。
// 为了满足现有数据库约束（需要密码哈希），新用户会生成随机密码并进行哈希保存。
func (s *AuthService) LoginOrRegisterOAuth(ctx context.Context, email, username string) (string, *User, error) {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// OAuth 首次登录视为注册（fail-close：settingService 未配置时不允许注册）
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return "", nil, ErrRegDisabled
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				log.Printf("[Auth] Failed to generate random password for oauth signup: %v", err)
				return "", nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return "", nil, fmt.Errorf("hash password: %w", err)
			}

			// 新用户默认值。
			defaultBalance := s.cfg.Default.UserBalance
			defaultConcurrency := s.cfg.Default.UserConcurrency
			if s.settingService != nil {
				defaultBalance = s.settingService.GetDefaultBalance(ctx)
				defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
			}

			// 为新用户生成唯一邀请码
			oauthInviteCode, err := s.generateUniqueInviteCode(ctx)
			if err != nil {
				log.Printf("[Auth] Failed to generate invite code for oauth user: %v", err)
				return "", nil, ErrServiceUnavailable
			}

			newUser := &User{
				Email:        email,
				Username:     username,
				PasswordHash: hashedPassword,
				Role:         RoleUser,
				Balance:      defaultBalance,
				Concurrency:  defaultConcurrency,
				Status:       StatusActive,
				InviteCode:   &oauthInviteCode,
			}

			if err := s.createUserWithInviteCodeRetry(ctx, newUser); err != nil {
				if errors.Is(err, ErrEmailExists) {
					// 并发场景：GetByEmail 与 Create 之间用户被创建。
					user, err = s.userRepo.GetByEmail(ctx, email)
					if err != nil {
						log.Printf("[Auth] Database error getting user after conflict: %v", err)
						return "", nil, ErrServiceUnavailable
					}
				} else {
					log.Printf("[Auth] Database error creating oauth user: %v", err)
					return "", nil, ErrServiceUnavailable
				}
			} else {
				user = newUser
			}
		} else {
			log.Printf("[Auth] Database error during oauth login: %v", err)
			return "", nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 尽力补全：当用户名为空时，使用第三方返回的用户名回填。
	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			log.Printf("[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	return token, user, nil
}

// LoginOrRegisterOAuthWithTokenPair 用于第三方 OAuth/SSO 登录，返回完整的 TokenPair
// 与 LoginOrRegisterOAuth 功能相同，但返回 TokenPair 而非单个 token
func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, email, username string) (*TokenPair, *User, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, nil, errors.New("refresh token cache not configured")
	}

	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// OAuth 首次登录视为注册
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return nil, nil, ErrRegDisabled
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				log.Printf("[Auth] Failed to generate random password for oauth signup: %v", err)
				return nil, nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return nil, nil, fmt.Errorf("hash password: %w", err)
			}

			defaultBalance := s.cfg.Default.UserBalance
			defaultConcurrency := s.cfg.Default.UserConcurrency
			if s.settingService != nil {
				defaultBalance = s.settingService.GetDefaultBalance(ctx)
				defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
			}

			newUser := &User{
				Email:        email,
				Username:     username,
				PasswordHash: hashedPassword,
				Role:         RoleUser,
				Balance:      defaultBalance,
				Concurrency:  defaultConcurrency,
				Status:       StatusActive,
			}

			if err := s.userRepo.Create(ctx, newUser); err != nil {
				if errors.Is(err, ErrEmailExists) {
					user, err = s.userRepo.GetByEmail(ctx, email)
					if err != nil {
						log.Printf("[Auth] Database error getting user after conflict: %v", err)
						return nil, nil, ErrServiceUnavailable
					}
				} else {
					log.Printf("[Auth] Database error creating oauth user: %v", err)
					return nil, nil, ErrServiceUnavailable
				}
			} else {
				user = newUser
			}
		} else {
			log.Printf("[Auth] Database error during oauth login: %v", err)
			return nil, nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return nil, nil, ErrUserNotActive
	}

	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			log.Printf("[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	tokenPair, err := s.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return nil, nil, fmt.Errorf("generate token pair: %w", err)
	}
	return tokenPair, user, nil
}

// ValidateToken 验证JWT token并返回用户声明
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// 先做长度校验，尽早拒绝异常超长 token，降低 DoS 风险。
	if len(tokenString) > maxTokenLength {
		return nil, ErrTokenTooLarge
	}

	// 使用解析器并限制可接受的签名算法，防止算法混淆。
	parser := jwt.NewParser(jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Name,
		jwt.SigningMethodHS384.Name,
		jwt.SigningMethodHS512.Name,
	}))

	// 保留默认 claims 校验（exp/nbf），避免放行过期或未生效的 token。
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// token 过期但仍返回 claims（用于 RefreshToken 等场景）
			// jwt-go 在解析时即使遇到过期错误，token.Claims 仍会被填充
			if claims, ok := token.Claims.(*JWTClaims); ok {
				return claims, ErrTokenExpired
			}
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func randomHexString(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 16
	}
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isReservedEmail(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return strings.HasSuffix(normalized, LinuxDoConnectSyntheticEmailDomain)
}

// GenerateToken 生成JWT access token
// 使用新的access_token_expire_minutes配置项（如果配置了），否则回退到expire_hour
func (s *AuthService) GenerateToken(user *User) (string, error) {
	now := time.Now()
	var expiresAt time.Time
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		expiresAt = now.Add(time.Duration(s.cfg.JWT.AccessTokenExpireMinutes) * time.Minute)
	} else {
		// 向后兼容：使用旧的expire_hour配置
		expiresAt = now.Add(time.Duration(s.cfg.JWT.ExpireHour) * time.Hour)
	}

	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// GetAccessTokenExpiresIn 返回Access Token的有效期（秒）
// 用于前端设置刷新定时器
func (s *AuthService) GetAccessTokenExpiresIn() int {
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		return s.cfg.JWT.AccessTokenExpireMinutes * 60
	}
	return s.cfg.JWT.ExpireHour * 3600
}

// HashPassword 使用bcrypt加密密码
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword 验证密码是否匹配
func (s *AuthService) CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, oldTokenString string) (string, error) {
	// 验证旧token（即使过期也允许，用于刷新）
	claims, err := s.ValidateToken(oldTokenString)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		return "", err
	}

	// 获取最新的用户信息
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidToken
		}
		log.Printf("[Auth] Database error refreshing token: %v", err)
		return "", ErrServiceUnavailable
	}

	// 检查用户状态
	if !user.IsActive() {
		return "", ErrUserNotActive
	}

	// Security: Check TokenVersion to prevent refreshing revoked tokens
	// This ensures tokens issued before a password change cannot be refreshed
	if claims.TokenVersion != user.TokenVersion {
		return "", ErrTokenRevoked
	}

	// 生成新token
	return s.GenerateToken(user)
}

// createUserWithInviteCodeRetry 创建用户，遇到 invite_code 碰撞时自动重试。
func (s *AuthService) createUserWithInviteCodeRetry(ctx context.Context, user *User) error {
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		if err := s.userRepo.Create(ctx, user); err != nil {
			if !errors.Is(err, ErrInviteCodeExists) {
				return err
			}
			if i == maxRetries-1 {
				return err
			}
			code, genErr := s.generateUniqueInviteCode(ctx)
			if genErr != nil {
				return genErr
			}
			user.InviteCode = &code
			continue
		}
		return nil
	}
	return ErrInviteCodeExists
}

// generateUniqueInviteCode 生成唯一邀请码（带冲突重试）
func (s *AuthService) generateUniqueInviteCode(ctx context.Context) (string, error) {
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code, err := generateInviteCode()
		if err != nil {
			return "", err
		}
		_, err = s.userRepo.GetByInviteCode(ctx, code)
		if errors.Is(err, ErrUserNotFound) {
			return code, nil
		}
		if err != nil {
			return "", fmt.Errorf("check invite code existence: %w", err)
		}
	}
	return "", fmt.Errorf("failed to generate unique invite code after %d attempts", maxRetries)
}

// ==================== Refresh Token Methods ====================

// TokenPair 包含Access Token和Refresh Token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token有效期（秒）
}

// GenerateTokenPair 生成Access Token和Refresh Token对
// familyID: 可选的Token家族ID，用于Token轮转时保持家族关系
func (s *AuthService) GenerateTokenPair(ctx context.Context, user *User, familyID string) (*TokenPair, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, errors.New("refresh token cache not configured")
	}

	// 生成Access Token
	accessToken, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 生成Refresh Token
	refreshToken, err := s.generateRefreshToken(ctx, user, familyID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.GetAccessTokenExpiresIn(),
	}, nil
}

// generateRefreshToken 生成并存储Refresh Token
func (s *AuthService) generateRefreshToken(ctx context.Context, user *User, familyID string) (string, error) {
	// 生成随机Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawToken := refreshTokenPrefix + hex.EncodeToString(tokenBytes)

	// 计算Token哈希（存储哈希而非原始Token）
	tokenHash := hashToken(rawToken)

	// 如果没有提供familyID，生成新的
	if familyID == "" {
		familyBytes := make([]byte, 16)
		if _, err := rand.Read(familyBytes); err != nil {
			return "", fmt.Errorf("generate family id: %w", err)
		}
		familyID = hex.EncodeToString(familyBytes)
	}

	now := time.Now()
	ttl := time.Duration(s.cfg.JWT.RefreshTokenExpireDays) * 24 * time.Hour

	data := &RefreshTokenData{
		UserID:       user.ID,
		TokenVersion: user.TokenVersion,
		FamilyID:     familyID,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
	}

	// 存储Token数据
	if err := s.refreshTokenCache.StoreRefreshToken(ctx, tokenHash, data, ttl); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	// 添加到用户Token集合
	if err := s.refreshTokenCache.AddToUserTokenSet(ctx, user.ID, tokenHash, ttl); err != nil {
		log.Printf("[Auth] Failed to add token to user set: %v", err)
		// 不影响主流程
	}

	// 添加到家族Token集合
	if err := s.refreshTokenCache.AddToFamilyTokenSet(ctx, familyID, tokenHash, ttl); err != nil {
		log.Printf("[Auth] Failed to add token to family set: %v", err)
		// 不影响主流程
	}

	return rawToken, nil
}

// RefreshTokenPair 使用Refresh Token刷新Token对
// 实现Token轮转：每次刷新都会生成新的Refresh Token，旧Token立即失效
func (s *AuthService) RefreshTokenPair(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, ErrRefreshTokenInvalid
	}

	// 验证Token格式
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return nil, ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)

	// 获取Token数据
	data, err := s.refreshTokenCache.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			// Token不存在，可能是已被使用（Token轮转）或已过期
			log.Printf("[Auth] Refresh token not found, possible reuse attack")
			return nil, ErrRefreshTokenInvalid
		}
		log.Printf("[Auth] Error getting refresh token: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 检查Token是否过期
	if time.Now().After(data.ExpiresAt) {
		// 删除过期Token
		_ = s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
		return nil, ErrRefreshTokenExpired
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, data.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// 用户已删除，撤销整个Token家族
			_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
			return nil, ErrRefreshTokenInvalid
		}
		log.Printf("[Auth] Database error getting user for token refresh: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 检查用户状态
	if !user.IsActive() {
		// 用户被禁用，撤销整个Token家族
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrUserNotActive
	}

	// 检查TokenVersion（密码更改后所有Token失效）
	if data.TokenVersion != user.TokenVersion {
		// TokenVersion不匹配，撤销整个Token家族
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrTokenRevoked
	}

	// Token轮转：立即使旧Token失效
	if err := s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash); err != nil {
		log.Printf("[Auth] Failed to delete old refresh token: %v", err)
		// 继续处理，不影响主流程
	}

	// 生成新的Token对，保持同一个家族ID
	return s.GenerateTokenPair(ctx, user, data.FamilyID)
}

// RevokeRefreshToken 撤销单个Refresh Token
func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)
	return s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
}

// RevokeAllUserSessions 撤销用户的所有会话（所有Refresh Token）
// 用于密码更改或用户主动登出所有设备
func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	return s.refreshTokenCache.DeleteUserRefreshTokens(ctx, userID)
}

// hashToken 计算Token的SHA256哈希
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
