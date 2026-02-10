-- 056: Add subscription_id to api_keys for cross-platform subscription sharing
ALTER TABLE api_keys ADD COLUMN subscription_id BIGINT NULL;
CREATE INDEX idx_api_keys_subscription_id ON api_keys(subscription_id);
ALTER TABLE api_keys ADD CONSTRAINT fk_api_keys_subscription
  FOREIGN KEY (subscription_id) REFERENCES user_subscriptions(id) ON DELETE SET NULL;
