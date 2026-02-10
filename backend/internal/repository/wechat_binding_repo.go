package repository

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/wechatbinding"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type weChatBindingRepository struct {
	client *dbent.Client
}

func NewWeChatBindingRepository(client *dbent.Client) service.WeChatBindingRepository {
	return &weChatBindingRepository{client: client}
}

func (r *weChatBindingRepository) Create(ctx context.Context, userID int64, appID, openid string) error {
	_, err := r.client.WeChatBinding.Create().
		SetUserID(userID).
		SetAppID(appID).
		SetOpenid(openid).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create wechat binding: %w", err)
	}
	return nil
}

func (r *weChatBindingRepository) GetByOpenID(ctx context.Context, appID, openid string) (*service.WeChatBinding, error) {
	row, err := r.client.WeChatBinding.Query().
		Where(
			wechatbinding.AppID(appID),
			wechatbinding.Openid(openid),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get wechat binding by openid: %w", err)
	}
	return toServiceBinding(row), nil
}

func (r *weChatBindingRepository) GetByUserID(ctx context.Context, userID int64) (*service.WeChatBinding, error) {
	row, err := r.client.WeChatBinding.Query().
		Where(wechatbinding.UserID(userID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get wechat binding by user: %w", err)
	}
	return toServiceBinding(row), nil
}

func (r *weChatBindingRepository) UpdateSubscribed(ctx context.Context, appID, openid string, subscribed bool) error {
	_, err := r.client.WeChatBinding.Update().
		Where(
			wechatbinding.AppID(appID),
			wechatbinding.Openid(openid),
		).
		SetSubscribed(subscribed).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update wechat binding subscribed: %w", err)
	}
	return nil
}

func (r *weChatBindingRepository) Delete(ctx context.Context, userID int64) error {
	_, err := r.client.WeChatBinding.Delete().
		Where(wechatbinding.UserID(userID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete wechat binding: %w", err)
	}
	return nil
}

func toServiceBinding(row *dbent.WeChatBinding) *service.WeChatBinding {
	return &service.WeChatBinding{
		ID:         row.ID,
		UserID:     row.UserID,
		AppID:      row.AppID,
		OpenID:     row.Openid,
		UnionID:    row.Unionid,
		Subscribed: row.Subscribed,
	}
}
