CREATE UNIQUE INDEX IF NOT EXISTS idx_wechat_bindings_user_id_unique
    ON wechat_bindings (user_id);

DROP INDEX IF EXISTS idx_wechat_bindings_user_id;
