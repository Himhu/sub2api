CREATE TABLE IF NOT EXISTS wechat_bindings (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id      VARCHAR(64)  NOT NULL,
    openid      VARCHAR(64)  NOT NULL,
    unionid     VARCHAR(64),
    subscribed  BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_wechat_bindings_app_openid
    ON wechat_bindings (app_id, openid);

CREATE INDEX IF NOT EXISTS idx_wechat_bindings_user_id
    ON wechat_bindings (user_id);
