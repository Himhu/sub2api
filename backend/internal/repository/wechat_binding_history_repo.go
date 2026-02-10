package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/wechatbindinghistory"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type weChatBindingHistoryRepository struct {
	client *dbent.Client
}

func NewWeChatBindingHistoryRepository(client *dbent.Client) service.WeChatBindingHistoryRepository {
	return &weChatBindingHistoryRepository{client: client}
}

func (r *weChatBindingHistoryRepository) Create(ctx context.Context, userID int64, appID, openid, reason string) error {
	if reason == "" {
		reason = "user_unbind"
	}
	// Upsert: if tombstone already exists for this (app_id, openid), update it
	err := r.client.WeChatBindingHistory.Create().
		SetUserID(userID).
		SetAppID(appID).
		SetOpenid(openid).
		SetReason(reason).
		SetUnboundAt(time.Now()).
		OnConflictColumns(wechatbindinghistory.FieldAppID, wechatbindinghistory.FieldOpenid).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert wechat binding history: %w", err)
	}
	return nil
}

func (r *weChatBindingHistoryRepository) GetByOpenID(ctx context.Context, appID, openid string) (*service.WeChatBindingHistory, error) {
	row, err := r.client.WeChatBindingHistory.Query().
		Where(
			wechatbindinghistory.AppID(appID),
			wechatbindinghistory.Openid(openid),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get wechat binding history by openid: %w", err)
	}
	return &service.WeChatBindingHistory{
		ID:        row.ID,
		UserID:    row.UserID,
		AppID:     row.AppID,
		OpenID:    row.Openid,
		UnboundAt: row.UnboundAt,
		Reason:    row.Reason,
	}, nil
}
