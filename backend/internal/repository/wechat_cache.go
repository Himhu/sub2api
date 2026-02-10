package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	wechatScenePrefix         = "wechat:scene:"
	wechatCooldownPrefix      = "wechat:cooldown:"
	wechatAccessTokenPrefix   = "wechat:access_token:"
	wechatPwdResetPrefix      = "wechat:password_reset:"
	wechatPwdResetCDPrefix    = "wechat:password_reset_cooldown:"
)

type weChatCache struct {
	rdb *redis.Client
}

func NewWeChatCache(rdb *redis.Client) service.WeChatCache {
	return &weChatCache{rdb: rdb}
}

// --- Access Token ---

func (c *weChatCache) GetAccessToken(ctx context.Context, appID string) (string, error) {
	val, err := c.rdb.Get(ctx, wechatAccessTokenPrefix+appID).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get wechat access token: %w", err)
	}
	return val, nil
}

func (c *weChatCache) SetAccessToken(ctx context.Context, appID, token string, ttl time.Duration) error {
	return c.rdb.Set(ctx, wechatAccessTokenPrefix+appID, token, ttl).Err()
}

// --- Scene Sessions ---

func (c *weChatCache) CreateScene(ctx context.Context, sceneID string, session *service.SceneSession, ttl time.Duration) error {
	return c.setJSON(ctx, wechatScenePrefix+sceneID, session, ttl)
}

func (c *weChatCache) GetScene(ctx context.Context, sceneID string) (*service.SceneSession, error) {
	var s service.SceneSession
	found, err := c.getJSON(ctx, wechatScenePrefix+sceneID, &s)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

func (c *weChatCache) UpdateScene(ctx context.Context, sceneID string, session *service.SceneSession, ttl time.Duration) error {
	return c.setJSON(ctx, wechatScenePrefix+sceneID, session, ttl)
}

func (c *weChatCache) DeleteScene(ctx context.Context, sceneID string) error {
	return c.rdb.Del(ctx, wechatScenePrefix+sceneID).Err()
}

// --- Send Cooldown ---

func (c *weChatCache) IsInCooldown(ctx context.Context, openid string) bool {
	exists, _ := c.rdb.Exists(ctx, wechatCooldownPrefix+openid).Result()
	return exists > 0
}

func (c *weChatCache) SetCooldown(ctx context.Context, openid string, ttl time.Duration) error {
	return c.rdb.Set(ctx, wechatCooldownPrefix+openid, "1", ttl).Err()
}

// --- Password Reset ---

func (c *weChatCache) SetPasswordResetCode(ctx context.Context, email string, session *service.PasswordResetSession, ttl time.Duration) error {
	return c.setJSON(ctx, wechatPwdResetPrefix+email, session, ttl)
}

func (c *weChatCache) GetPasswordResetCode(ctx context.Context, email string) (*service.PasswordResetSession, error) {
	var s service.PasswordResetSession
	found, err := c.getJSON(ctx, wechatPwdResetPrefix+email, &s)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

func (c *weChatCache) IncrPasswordResetAttempts(ctx context.Context, email string) (int, error) {
	session, err := c.GetPasswordResetCode(ctx, email)
	if err != nil {
		return 0, err
	}
	if session == nil {
		return 0, fmt.Errorf("no password reset session for %s", email)
	}
	session.Attempts++
	ttl, err := c.rdb.TTL(ctx, wechatPwdResetPrefix+email).Result()
	if err != nil || ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if err := c.setJSON(ctx, wechatPwdResetPrefix+email, session, ttl); err != nil {
		return 0, err
	}
	return session.Attempts, nil
}

func (c *weChatCache) DeletePasswordResetCode(ctx context.Context, email string) error {
	return c.rdb.Del(ctx, wechatPwdResetPrefix+email).Err()
}

// --- Password Reset Cooldown ---

func (c *weChatCache) IsPasswordResetInCooldown(ctx context.Context, email string) bool {
	exists, _ := c.rdb.Exists(ctx, wechatPwdResetCDPrefix+email).Result()
	return exists > 0
}

func (c *weChatCache) SetPasswordResetCooldown(ctx context.Context, email string, ttl time.Duration) error {
	return c.rdb.Set(ctx, wechatPwdResetCDPrefix+email, "1", ttl).Err()
}

// --- JSON helpers ---

func (c *weChatCache) setJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *weChatCache) getJSON(ctx context.Context, key string, dest any) (bool, error) {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get %s: %w", key, err)
	}
	return true, json.Unmarshal(data, dest)
}
