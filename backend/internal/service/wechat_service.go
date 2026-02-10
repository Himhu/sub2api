package service

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/singleflight"
)

const (
	wechatTokenURL    = "https://api.weixin.qq.com/cgi-bin/token"
	wechatQRCodeURL   = "https://api.weixin.qq.com/cgi-bin/qrcode/create"
	wechatShowQRURL   = "https://mp.weixin.qq.com/cgi-bin/showqrcode"
	accessTokenMargin = 300 // seconds to subtract from expires_in
)

var (
	wechatHTTP = &http.Client{Timeout: 10 * time.Second}

	ErrWeChatDisabled      = infraerrors.ServiceUnavailable("WECHAT_DISABLED", "wechat is disabled")
	ErrWeChatNotConfigured = infraerrors.ServiceUnavailable("WECHAT_NOT_CONFIGURED", "wechat not configured")
)

// WeChatService is the WeChat API client.
type WeChatService struct {
	settingService *SettingService
	cache          WeChatCache
	sfGroup        singleflight.Group
}

func NewWeChatService(settingService *SettingService, cache WeChatCache) *WeChatService {
	return &WeChatService{settingService: settingService, cache: cache}
}

// RefreshAccessToken returns a cached access token or fetches a new one via singleflight.
func (s *WeChatService) RefreshAccessToken(ctx context.Context) (string, error) {
	cfg, err := s.requireConfig(ctx)
	if err != nil {
		return "", err
	}

	if token, _ := s.cache.GetAccessToken(ctx, cfg.AppID); token != "" {
		return token, nil
	}

	val, err, _ := s.sfGroup.Do("wechat_token:"+cfg.AppID, func() (any, error) {
		// double-check after acquiring singleflight
		if token, _ := s.cache.GetAccessToken(ctx, cfg.AppID); token != "" {
			return token, nil
		}
		token, expiresIn, err := s.fetchAccessToken(ctx, cfg)
		if err != nil {
			return "", err
		}
		ttl := expiresIn - accessTokenMargin
		if ttl < 60 {
			ttl = 60
		}
		_ = s.cache.SetAccessToken(ctx, cfg.AppID, token, time.Duration(ttl)*time.Second)
		return token, nil
	})
	if err != nil {
		return "", err
	}
	return val.(string), nil
}

// CreateTempQRCode creates a temporary QR code with a scene string.
func (s *WeChatService) CreateTempQRCode(ctx context.Context, sceneStr string, expireSec int) (ticket, qrcodeURL string, err error) {
	token, err := s.RefreshAccessToken(ctx)
	if err != nil {
		return "", "", err
	}

	body := wxQRCodeReq{ExpireSeconds: expireSec, ActionName: "QR_STR_SCENE"}
	body.ActionInfo.Scene.SceneStr = sceneStr

	var resp wxQRCodeResp
	if err := s.postJSON(ctx, wechatQRCodeURL, token, body, &resp); err != nil {
		return "", "", err
	}
	if resp.ErrCode != 0 {
		return "", "", wxError("WECHAT_QRCODE_ERROR", resp.wxErr)
	}
	if resp.Ticket == "" {
		return "", "", infraerrors.ServiceUnavailable("WECHAT_QRCODE_EMPTY", "empty ticket")
	}

	showURL := wechatShowQRURL + "?ticket=" + url.QueryEscape(resp.Ticket)
	return resp.Ticket, showURL, nil
}

// VerifySignature validates a WeChat callback signature.
func (s *WeChatService) VerifySignature(token, timestamp, nonce, signature string) bool {
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return strings.EqualFold(signature, fmt.Sprintf("%x", sum))
}

// ParseCallbackXML parses a WeChat callback XML payload.
func (s *WeChatService) ParseCallbackXML(body []byte) (*WeChatCallbackEvent, error) {
	var event WeChatCallbackEvent
	if err := xml.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("parse wechat callback xml: %w", err)
	}
	return &event, nil
}

// --- internal helpers ---

func (s *WeChatService) requireConfig(ctx context.Context) (WeChatConfig, error) {
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
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return WeChatConfig{}, ErrWeChatNotConfigured
	}
	return cfg, nil
}

func (s *WeChatService) fetchAccessToken(ctx context.Context, cfg WeChatConfig) (string, int, error) {
	u := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s",
		wechatTokenURL, url.QueryEscape(cfg.AppID), url.QueryEscape(cfg.AppSecret))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create token request: %w", err)
	}
	resp, err := wechatHTTP.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("wechat token http %d: %s", resp.StatusCode, raw)
	}

	var tokenResp struct {
		wxErr
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.ErrCode != 0 {
		return "", 0, wxError("WECHAT_TOKEN_ERROR", tokenResp.wxErr)
	}
	if tokenResp.AccessToken == "" {
		return "", 0, infraerrors.ServiceUnavailable("WECHAT_TOKEN_EMPTY", "empty access token")
	}
	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}

func (s *WeChatService) postJSON(ctx context.Context, baseURL, accessToken string, reqBody any, dest any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	u := baseURL + "?access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wechatHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("wechat request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[WeChat] HTTP %d: %s", resp.StatusCode, raw)
		return fmt.Errorf("wechat http %d", resp.StatusCode)
	}
	return json.Unmarshal(raw, dest)
}

func wxError(code string, e wxErr) error {
	return infraerrors.ServiceUnavailable(code, fmt.Sprintf("wechat: %d %s", e.ErrCode, e.ErrMsg))
}

// --- WeChat API types ---

type wxErr struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type wxQRCodeReq struct {
	ExpireSeconds int    `json:"expire_seconds"`
	ActionName    string `json:"action_name"`
	ActionInfo    struct {
		Scene struct {
			SceneStr string `json:"scene_str"`
		} `json:"scene"`
	} `json:"action_info"`
}

type wxQRCodeResp struct {
	wxErr
	Ticket string `json:"ticket"`
}
