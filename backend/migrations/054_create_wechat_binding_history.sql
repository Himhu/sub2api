CREATE TABLE IF NOT EXISTS wechat_binding_history (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    app_id      VARCHAR(64)  NOT NULL,
    openid      VARCHAR(64)  NOT NULL,
    unbound_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    reason      VARCHAR(64)  NOT NULL DEFAULT 'user_unbind',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_wechat_binding_history_app_openid
    ON wechat_binding_history (app_id, openid);

CREATE INDEX IF NOT EXISTS idx_wechat_binding_history_user_id
    ON wechat_binding_history (user_id);
