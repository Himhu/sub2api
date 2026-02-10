package service

import (
	"context"
	"encoding/xml"
	"time"
)

// SceneSession represents a QR code scan session stored in Redis.
type SceneSession struct {
	Status     string `json:"status"`      // "pending" | "code_sent" | "consumed" | "bound"
	OpenID     string `json:"openid"`      // set after scan
	VerifyCode string `json:"verify_code"` // 6-digit code
	Attempts   int    `json:"attempts"`    // verification attempts
	CreatedAt  int64  `json:"created_at"`
	UserID     int64  `json:"user_id,omitempty"` // set for user-binding scenes
}

// PasswordResetSession represents a password reset session stored in Redis.
type PasswordResetSession struct {
	Code      string `json:"code"`
	Attempts  int    `json:"attempts"`
	CreatedAt int64  `json:"created_at"`
}

// WeChatTextReply represents a passive text reply to WeChat.
type WeChatTextReply struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

// WeChatCallbackEvent represents a parsed WeChat callback XML event.
type WeChatCallbackEvent struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"` // openid
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"` // text message content
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	Ticket       string   `xml:"Ticket"`
}

// WeChatBindingRepository defines DB operations for wechat_bindings.
type WeChatBindingRepository interface {
	Create(ctx context.Context, userID int64, appID, openid string) error
	GetByOpenID(ctx context.Context, appID, openid string) (*WeChatBinding, error)
	GetByUserID(ctx context.Context, userID int64) (*WeChatBinding, error)
	UpdateSubscribed(ctx context.Context, appID, openid string, subscribed bool) error
	Delete(ctx context.Context, userID int64) error
}

// WeChatCache defines Redis operations for WeChat verification flows.
type WeChatCache interface {
	// Access token
	GetAccessToken(ctx context.Context, appID string) (string, error)
	SetAccessToken(ctx context.Context, appID, token string, ttl time.Duration) error

	// QR code scene sessions
	CreateScene(ctx context.Context, sceneID string, session *SceneSession, ttl time.Duration) error
	GetScene(ctx context.Context, sceneID string) (*SceneSession, error)
	UpdateScene(ctx context.Context, sceneID string, session *SceneSession, ttl time.Duration) error
	DeleteScene(ctx context.Context, sceneID string) error

	// Send cooldown (per openid)
	IsInCooldown(ctx context.Context, openid string) bool
	SetCooldown(ctx context.Context, openid string, ttl time.Duration) error

	// Password reset code
	SetPasswordResetCode(ctx context.Context, email string, session *PasswordResetSession, ttl time.Duration) error
	GetPasswordResetCode(ctx context.Context, email string) (*PasswordResetSession, error)
	IncrPasswordResetAttempts(ctx context.Context, email string) (int, error)
	DeletePasswordResetCode(ctx context.Context, email string) error

	// Password reset cooldown
	IsPasswordResetInCooldown(ctx context.Context, email string) bool
	SetPasswordResetCooldown(ctx context.Context, email string, ttl time.Duration) error
}

// WeChatBindingHistoryRepository defines DB operations for wechat binding history (tombstones).
type WeChatBindingHistoryRepository interface {
	Create(ctx context.Context, userID int64, appID, openid, reason string) error
	GetByOpenID(ctx context.Context, appID, openid string) (*WeChatBindingHistory, error)
}

// WeChatBinding is the service-layer representation of a wechat binding record.
type WeChatBinding struct {
	ID         int64
	UserID     int64
	AppID      string
	OpenID     string
	UnionID    *string
	Subscribed bool
}

// WeChatBindingHistory is the service-layer representation of a binding history record.
type WeChatBindingHistory struct {
	ID        int64
	UserID    int64
	AppID     string
	OpenID    string
	UnboundAt time.Time
	Reason    string
}
