package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	wechatSceneTTL          = 5 * time.Minute
	wechatQRCodeExpireSec   = 300
	wechatCooldownDuration  = 60 * time.Second
	wechatPwdResetTTL       = 15 * time.Minute
	wechatMaxVerifyAttempts  = 5
	wechatShortCodeLength   = 4
	wechatShortCodeCharset  = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

var (
	ErrWeChatSignatureInvalid          = infraerrors.Forbidden("WECHAT_SIGNATURE_INVALID", "invalid wechat signature")
	ErrWeChatSessionExpired            = infraerrors.BadRequest("WECHAT_SESSION_EXPIRED", "verification session expired")
	ErrWeChatCodeNotReady              = infraerrors.BadRequest("WECHAT_CODE_NOT_READY", "verification code not ready")
	ErrWeChatInvalidCode               = infraerrors.BadRequest("WECHAT_CODE_INVALID", "invalid verification code")
	ErrWeChatTooManyAttempts           = infraerrors.Forbidden("WECHAT_TOO_MANY_ATTEMPTS", "too many attempts, please try again later")
	ErrWeChatPwdResetCooldown          = infraerrors.Forbidden("WECHAT_PWD_RESET_COOLDOWN", "password reset recently requested, please wait")
	ErrWeChatPwdResetExpired           = infraerrors.BadRequest("WECHAT_PWD_RESET_EXPIRED", "password reset code expired")
	ErrWeChatPwdResetInvalid           = infraerrors.BadRequest("WECHAT_PWD_RESET_INVALID", "invalid password reset code")
	ErrWeChatPwdResetTooManyAttempts   = infraerrors.Forbidden("WECHAT_PWD_RESET_TOO_MANY_ATTEMPTS", "too many password reset attempts")
	ErrWeChatBindingExists             = infraerrors.BadRequest("WECHAT_BINDING_EXISTS", "wechat already bound for this account")
)

// WeChatVerificationService handles QR scan and OAuth verification flows.
type WeChatVerificationService struct {
	wechatService      *WeChatService
	cache              WeChatCache
	bindingRepo        WeChatBindingRepository
	bindingHistoryRepo WeChatBindingHistoryRepository
	settingService     *SettingService
	userRepo           UserRepository
}

func NewWeChatVerificationService(
	wechatService *WeChatService,
	cache WeChatCache,
	bindingRepo WeChatBindingRepository,
	bindingHistoryRepo WeChatBindingHistoryRepository,
	settingService *SettingService,
	userRepo UserRepository,
) *WeChatVerificationService {
	return &WeChatVerificationService{
		wechatService:      wechatService,
		cache:              cache,
		bindingRepo:        bindingRepo,
		bindingHistoryRepo: bindingHistoryRepo,
		settingService:     settingService,
		userRepo:           userRepo,
	}
}

// CreateQRCodeScene creates a pending scene session and returns a QR code URL.
func (s *WeChatVerificationService) CreateQRCodeScene(ctx context.Context) (sceneID, qrcodeURL string, expireSec int, err error) {
	sceneID, err = generateRandomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate scene id: %w", err)
	}

	session := &SceneSession{Status: "pending", CreatedAt: time.Now().Unix()}
	if err := s.cache.CreateScene(ctx, sceneID, session, wechatSceneTTL); err != nil {
		return "", "", 0, fmt.Errorf("store scene session: %w", err)
	}

	_, qrcodeURL, err = s.wechatService.CreateTempQRCode(ctx, sceneID, wechatQRCodeExpireSec)
	if err != nil {
		return "", "", 0, err
	}
	return sceneID, qrcodeURL, wechatQRCodeExpireSec, nil
}

// CreateShortCodeScene creates a pending scene session keyed by a 4-char short code.
func (s *WeChatVerificationService) CreateShortCodeScene(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateShortCode()
		if err != nil {
			return "", err
		}
		if existing, _ := s.cache.GetScene(ctx, code); existing != nil {
			continue
		}
		session := &SceneSession{Status: "pending", CreatedAt: time.Now().Unix()}
		if err := s.cache.CreateScene(ctx, code, session, wechatSceneTTL); err != nil {
			return "", fmt.Errorf("store scene session: %w", err)
		}
		return code, nil
	}
	return "", fmt.Errorf("generate short code: exhausted attempts")
}

// createUserScopedScene creates a short code scene bound to a specific user.
func (s *WeChatVerificationService) createUserScopedScene(ctx context.Context, userID int64) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateShortCode()
		if err != nil {
			return "", err
		}
		if existing, _ := s.cache.GetScene(ctx, code); existing != nil {
			continue
		}
		session := &SceneSession{
			Status:    "pending",
			CreatedAt: time.Now().Unix(),
			UserID:    userID,
		}
		if err := s.cache.CreateScene(ctx, code, session, wechatSceneTTL); err != nil {
			return "", fmt.Errorf("store scene session: %w", err)
		}
		return code, nil
	}
	return "", fmt.Errorf("generate short code: exhausted attempts")
}

// PollScanStatus returns the current scan status for a scene.
func (s *WeChatVerificationService) PollScanStatus(ctx context.Context, sceneID string) (string, error) {
	if sceneID == "" {
		return "expired", nil
	}
	session, err := s.cache.GetScene(ctx, sceneID)
	if err != nil {
		return "", fmt.Errorf("get scene session: %w", err)
	}
	if session == nil {
		return "expired", nil
	}
	return session.Status, nil
}

// HandleCallback verifies and processes a WeChat callback event.
func (s *WeChatVerificationService) HandleCallback(ctx context.Context, xmlBody []byte, sig, ts, nonce string) (string, error) {
	cfg, err := s.getConfig(ctx)
	if err != nil {
		return "", err
	}
	if cfg.Token == "" {
		return "", ErrWeChatNotConfigured
	}
	if !s.wechatService.VerifySignature(cfg.Token, ts, nonce, sig) {
		return "", ErrWeChatSignatureInvalid
	}

	event, err := s.wechatService.ParseCallbackXML(xmlBody)
	if err != nil {
		return "", infraerrors.BadRequest("WECHAT_XML_INVALID", "invalid callback xml")
	}

	var replyXML string
	switch strings.ToLower(event.MsgType) {
	case "event":
		switch strings.ToLower(event.Event) {
		case "subscribe", "scan":
			sceneID := parseSceneID(event.EventKey)
			replyXML, err = s.handleSubscribeOrScan(ctx, event.FromUserName, event.ToUserName, sceneID)
			if err != nil {
				log.Printf("[WeChatVerification] handleSubscribeOrScan error: %v", err)
			}
		case "unsubscribe":
			if err := s.handleUnsubscribe(ctx, event.FromUserName); err != nil {
				log.Printf("[WeChatVerification] handleUnsubscribe error: %v", err)
			}
		}
	case "text":
		replyXML, err = s.handleTextMessage(ctx, event.FromUserName, event.ToUserName, event.Content)
		if err != nil {
			log.Printf("[WeChatVerification] handleTextMessage error: %v", err)
		}
	}

	if replyXML != "" {
		return replyXML, nil
	}
	return "success", nil
}

func (s *WeChatVerificationService) handleSubscribeOrScan(ctx context.Context, openid, toUserName, sceneID string) (string, error) {
	if openid == "" || sceneID == "" {
		return "", nil
	}

	// 重新关注时恢复 subscribed 状态
	if cfg, err := s.getConfig(ctx); err == nil && cfg.AppID != "" {
		_ = s.bindingRepo.UpdateSubscribed(ctx, cfg.AppID, openid, true)
	}

	session, err := s.cache.GetScene(ctx, sceneID)
	if err != nil {
		return "", fmt.Errorf("get scene: %w", err)
	}
	if session == nil {
		return "", nil
	}

	// 冷却期内：仅更新 openid，不重新生成验证码
	if s.cache.IsInCooldown(ctx, openid) {
		if session.OpenID == "" {
			session.OpenID = openid
			_ = s.cache.UpdateScene(ctx, sceneID, session, wechatSceneTTL)
		}
		return s.buildTextReply(openid, toUserName, "验证码发送过于频繁，请稍后再试")
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	session.Status = "code_sent"
	session.OpenID = openid
	session.VerifyCode = code
	if err := s.cache.UpdateScene(ctx, sceneID, session, wechatSceneTTL); err != nil {
		return "", fmt.Errorf("update scene: %w", err)
	}

	_ = s.cache.SetCooldown(ctx, openid, wechatCooldownDuration)
	return s.buildTextReply(openid, toUserName, fmt.Sprintf("您的验证码：%s，5分钟内有效。", code))
}

func (s *WeChatVerificationService) handleUnsubscribe(ctx context.Context, openid string) error {
	if openid == "" {
		return nil
	}
	cfg, err := s.getConfig(ctx)
	if err != nil {
		return err
	}
	return s.bindingRepo.UpdateSubscribed(ctx, cfg.AppID, openid, false)
}

// ValidateAndConsumeCode validates a verification code and returns the openid.
func (s *WeChatVerificationService) ValidateAndConsumeCode(ctx context.Context, sceneID, inputCode string) (string, error) {
	if sceneID == "" || inputCode == "" {
		return "", infraerrors.BadRequest("WECHAT_CODE_REQUIRED", "scene and code are required")
	}
	return s.validateSceneCode(ctx, sceneID, inputCode)
}

// GetBindingStatus returns whether a user has a WeChat binding.
func (s *WeChatVerificationService) GetBindingStatus(ctx context.Context, userID int64) (bool, string, error) {
	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, "", fmt.Errorf("get wechat binding: %w", err)
	}
	if binding == nil || binding.OpenID == "" {
		return false, "", nil
	}
	return true, maskOpenID(binding.OpenID), nil
}

// InitiateBinding starts a binding flow for an existing user.
func (s *WeChatVerificationService) InitiateBinding(ctx context.Context, userID int64, password string) (string, string, error) {
	cfg, err := s.getConfig(ctx)
	if err != nil {
		return "", "", err
	}
	if cfg.AppID == "" {
		return "", "", ErrWeChatNotConfigured
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("get user: %w", err)
	}
	if !user.CheckPassword(password) {
		return "", "", ErrPasswordIncorrect
	}

	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("get wechat binding: %w", err)
	}
	if binding != nil {
		return "", "", ErrWeChatBindingExists
	}

	shortCode, err := s.createUserScopedScene(ctx, userID)
	if err != nil {
		return "", "", err
	}
	return shortCode, shortCode, nil
}

// CompleteBinding validates a code and binds the openid to the user.
func (s *WeChatVerificationService) CompleteBinding(ctx context.Context, userID int64, sceneID, code string) error {
	cfg, err := s.getConfig(ctx)
	if err != nil {
		return err
	}

	// Verify scene belongs to this user before consuming
	session, err := s.cache.GetScene(ctx, sceneID)
	if err != nil {
		return fmt.Errorf("get scene: %w", err)
	}
	if session == nil {
		return ErrWeChatSessionExpired
	}
	if session.UserID != 0 && session.UserID != userID {
		return infraerrors.Forbidden("WECHAT_SCENE_MISMATCH", "scene does not belong to this user")
	}

	openid, err := s.ValidateAndConsumeCode(ctx, sceneID, code)
	if err != nil {
		return err
	}

	if existing, err := s.bindingRepo.GetByUserID(ctx, userID); err != nil {
		return fmt.Errorf("check user binding: %w", err)
	} else if existing != nil {
		return ErrWeChatBindingExists
	}
	if existing, err := s.bindingRepo.GetByOpenID(ctx, cfg.AppID, openid); err != nil {
		return fmt.Errorf("check openid binding: %w", err)
	} else if existing != nil {
		return ErrWeChatAlreadyBound
	}
	// Tombstone check: block OpenID previously bound by a different user
	if history, err := s.bindingHistoryRepo.GetByOpenID(ctx, cfg.AppID, openid); err != nil {
		return fmt.Errorf("get binding history: %w", err)
	} else if history != nil && history.UserID != userID {
		return ErrWeChatAlreadyBound
	}

	if err := s.bindingRepo.Create(ctx, userID, cfg.AppID, openid); err != nil {
		// Handle unique constraint race condition
		errMsg := err.Error()
		if strings.Contains(errMsg, "unique") || strings.Contains(errMsg, "duplicate") {
			return ErrWeChatAlreadyBound
		}
		return fmt.Errorf("create wechat binding: %w", err)
	}
	return nil
}

// Unbind removes a user's WeChat binding after password verification.
// A tombstone record is written to prevent the same OpenID from registering new accounts.
func (s *WeChatVerificationService) Unbind(ctx context.Context, userID int64, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get wechat binding: %w", err)
	}
	if binding == nil {
		return ErrWeChatBindingNotFound
	}

	if password == "" {
		return ErrPasswordRequired
	}
	if !user.CheckPassword(password) {
		return ErrPasswordIncorrect
	}

	// Write tombstone before deleting — fail-close if history write fails
	if err := s.bindingHistoryRepo.Create(ctx, userID, binding.AppID, binding.OpenID, "user_unbind"); err != nil {
		return fmt.Errorf("create binding history: %w", err)
	}

	return s.bindingRepo.Delete(ctx, userID)
}

func (s *WeChatVerificationService) validateSceneCode(ctx context.Context, sceneID, inputCode string) (string, error) {
	session, err := s.cache.GetScene(ctx, sceneID)
	if err != nil {
		return "", fmt.Errorf("get scene: %w", err)
	}
	if session == nil {
		return "", ErrWeChatSessionExpired
	}
	if session.Status != "code_sent" {
		return "", ErrWeChatCodeNotReady
	}
	if session.Attempts >= wechatMaxVerifyAttempts {
		return "", ErrWeChatTooManyAttempts
	}
	if session.VerifyCode == inputCode {
		_ = s.cache.DeleteScene(ctx, sceneID)
		return session.OpenID, nil
	}
	session.Attempts++
	_ = s.cache.UpdateScene(ctx, sceneID, session, wechatSceneTTL)
	if session.Attempts >= wechatMaxVerifyAttempts {
		return "", ErrWeChatTooManyAttempts
	}
	return "", ErrWeChatInvalidCode
}

// RequestPasswordReset generates a password reset code (kept for API-triggered resets).
func (s *WeChatVerificationService) RequestPasswordReset(ctx context.Context, openid, email string) error {
	if openid == "" || email == "" {
		return infraerrors.BadRequest("WECHAT_PWD_RESET_INVALID", "openid or email is empty")
	}
	if _, err := s.getConfig(ctx); err != nil {
		return err
	}
	_, err := s.preparePasswordResetCode(ctx, email)
	return err
}

// ValidatePasswordResetCode validates a password reset code and consumes it on success.
func (s *WeChatVerificationService) ValidatePasswordResetCode(ctx context.Context, email, code string) error {
	if email == "" || code == "" {
		return infraerrors.BadRequest("WECHAT_PWD_RESET_INVALID", "email or code is empty")
	}

	session, err := s.cache.GetPasswordResetCode(ctx, email)
	if err != nil {
		return fmt.Errorf("get reset session: %w", err)
	}
	if session == nil {
		return ErrWeChatPwdResetExpired
	}
	if session.Attempts >= wechatMaxVerifyAttempts {
		return ErrWeChatPwdResetTooManyAttempts
	}
	if session.Code == code {
		_ = s.cache.DeletePasswordResetCode(ctx, email)
		return nil
	}

	attempts, err := s.cache.IncrPasswordResetAttempts(ctx, email)
	if err != nil {
		return fmt.Errorf("incr reset attempts: %w", err)
	}
	if attempts >= wechatMaxVerifyAttempts {
		return ErrWeChatPwdResetTooManyAttempts
	}
	return ErrWeChatPwdResetInvalid
}

// handleTextMessage processes text messages: short code verification or password reset keywords.
func (s *WeChatVerificationService) handleTextMessage(ctx context.Context, openid, toUserName, content string) (string, error) {
	if openid == "" || toUserName == "" {
		return "", nil
	}

	// Try short code registration flow first
	trimmed := strings.ToUpper(strings.TrimSpace(content))
	if len(trimmed) == wechatShortCodeLength {
		if reply, err := s.tryShortCodeVerification(ctx, openid, toUserName, trimmed); reply != "" || err != nil {
			return reply, err
		}
	}

	if !isPasswordResetKeyword(content) {
		return "", nil
	}

	cfg, err := s.getConfig(ctx)
	if err != nil {
		return "", err
	}

	binding, err := s.bindingRepo.GetByOpenID(ctx, cfg.AppID, openid)
	if err != nil {
		return "", fmt.Errorf("get wechat binding: %w", err)
	}
	if binding == nil {
		return s.buildTextReply(openid, toUserName, "请先在站点完成注册并绑定微信后再重置密码")
	}

	user, err := s.userRepo.GetByID(ctx, binding.UserID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	if user == nil || user.Email == "" {
		return s.buildTextReply(openid, toUserName, "未找到绑定账号，请联系管理员")
	}
	if !user.IsActive() {
		return s.buildTextReply(openid, toUserName, "账号已被禁用，无法重置密码")
	}

	password, err := s.resetPasswordDirectly(ctx, user)
	if err != nil {
		if errors.Is(err, ErrWeChatPwdResetCooldown) {
			return s.buildTextReply(openid, toUserName, "密码重置过于频繁，请稍后再试")
		}
		return "", err
	}

	return s.buildTextReply(openid, toUserName, fmt.Sprintf("您的密码已重置为：%s\n请登录后在个人资料中尽快修改密码。", password))
}

// tryShortCodeVerification checks if the message is a valid short code and processes registration verification.
// Returns ("", nil) if the message is not a short code match, allowing fallthrough to other handlers.
func (s *WeChatVerificationService) tryShortCodeVerification(ctx context.Context, openid, toUserName, shortCode string) (string, error) {
	session, err := s.cache.GetScene(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("get scene: %w", err)
	}
	if session == nil {
		return "", nil // not a short code, fall through
	}

	cfg, err := s.getConfig(ctx)
	if err != nil {
		return "", err
	}

	// Anti-batch: one WeChat account can only bind one site account
	binding, err := s.bindingRepo.GetByOpenID(ctx, cfg.AppID, openid)
	if err != nil {
		return "", fmt.Errorf("get wechat binding: %w", err)
	}
	if binding != nil {
		return s.buildTextReply(openid, toUserName, "该微信已绑定其他账号，无法继续注册")
	}
	// Tombstone check: block OpenID that was previously bound (unless re-binding to same user)
	if history, err := s.bindingHistoryRepo.GetByOpenID(ctx, cfg.AppID, openid); err != nil {
		return "", fmt.Errorf("get binding history: %w", err)
	} else if history != nil && (session.UserID == 0 || history.UserID != session.UserID) {
		return s.buildTextReply(openid, toUserName, "该微信已绑定过账号，无法用于新注册")
	}

	if s.cache.IsInCooldown(ctx, openid) {
		if session.OpenID == "" {
			session.OpenID = openid
			_ = s.cache.UpdateScene(ctx, shortCode, session, wechatSceneTTL)
		}
		return s.buildTextReply(openid, toUserName, "验证码发送过于频繁，请稍后再试")
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	session.Status = "code_sent"
	session.OpenID = openid
	session.VerifyCode = code
	if err := s.cache.UpdateScene(ctx, shortCode, session, wechatSceneTTL); err != nil {
		return "", fmt.Errorf("update scene: %w", err)
	}

	_ = s.cache.SetCooldown(ctx, openid, wechatCooldownDuration)
	return s.buildTextReply(openid, toUserName, fmt.Sprintf("您的验证码：%s，5分钟内有效。", code))
}

// --- internal helpers ---

func (s *WeChatVerificationService) getConfig(ctx context.Context) (WeChatConfig, error) {
	if s.settingService == nil {
		return WeChatConfig{}, ErrWeChatNotConfigured
	}
	if !s.settingService.IsWeChatEnabled(ctx) {
		return WeChatConfig{}, ErrWeChatDisabled
	}
	cfg, err := s.settingService.GetWeChatConfig(ctx)
	if err != nil {
		return WeChatConfig{}, fmt.Errorf("get wechat config: %w", err)
	}
	return cfg, nil
}

func (s *WeChatVerificationService) preparePasswordResetCode(ctx context.Context, email string) (string, error) {
	if s.cache.IsPasswordResetInCooldown(ctx, email) {
		return "", ErrWeChatPwdResetCooldown
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return "", fmt.Errorf("generate reset code: %w", err)
	}

	session := &PasswordResetSession{Code: code, CreatedAt: time.Now().Unix()}
	if err := s.cache.SetPasswordResetCode(ctx, email, session, wechatPwdResetTTL); err != nil {
		return "", fmt.Errorf("store reset session: %w", err)
	}

	_ = s.cache.SetPasswordResetCooldown(ctx, email, wechatCooldownDuration)
	return code, nil
}

func (s *WeChatVerificationService) resetPasswordDirectly(ctx context.Context, user *User) (string, error) {
	if s.cache.IsPasswordResetInCooldown(ctx, user.Email) {
		return "", ErrWeChatPwdResetCooldown
	}

	password, err := generateRandomPassword()
	if err != nil {
		return "", fmt.Errorf("generate reset password: %w", err)
	}
	if err := user.SetPassword(password); err != nil {
		return "", fmt.Errorf("set password: %w", err)
	}
	user.TokenVersion++
	if err := s.userRepo.Update(ctx, user); err != nil {
		return "", fmt.Errorf("update user: %w", err)
	}

	_ = s.cache.SetPasswordResetCooldown(ctx, user.Email, wechatCooldownDuration)
	return password, nil
}

func (s *WeChatVerificationService) buildTextReply(toUser, fromUser, content string) (string, error) {
	reply := WeChatTextReply{
		ToUserName:   toUser,
		FromUserName: fromUser,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      content,
	}
	out, err := xml.Marshal(reply)
	if err != nil {
		return "", fmt.Errorf("marshal wechat reply: %w", err)
	}
	return string(out), nil
}

func isPasswordResetKeyword(content string) bool {
	s := strings.TrimSpace(strings.ToLower(content))
	switch s {
	case "验证码", "重置密码", "修改密码", "reset", "reset password":
		return true
	default:
		return false
	}
}

func parseSceneID(eventKey string) string {
	if strings.HasPrefix(eventKey, "qrscene_") {
		return strings.TrimPrefix(eventKey, "qrscene_")
	}
	return eventKey
}

func maskOpenID(openid string) string {
	if len(openid) <= 4 {
		return openid[:1] + "***"
	}
	if len(openid) <= 8 {
		return openid[:2] + "***" + openid[len(openid)-1:]
	}
	return openid[:3] + "***" + openid[len(openid)-3:]
}


func generateShortCode() (string, error) {
	buf := make([]byte, wechatShortCodeLength)
	max := big.NewInt(int64(len(wechatShortCodeCharset)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}
		buf[i] = wechatShortCodeCharset[n.Int64()]
	}
	return string(buf), nil
}

func generateRandomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func generateSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate random code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateRandomPassword() (string, error) {
	const length = 12
	buf := make([]byte, length)
	max := big.NewInt(int64(len(wechatShortCodeCharset)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate random password: %w", err)
		}
		buf[i] = wechatShortCodeCharset[n.Int64()]
	}
	return string(buf), nil
}
