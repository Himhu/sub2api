-- Add source field to user_subscriptions table
ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'paid';

-- Ensure constraints are enforced even if column already exists (drifted DB)
ALTER TABLE user_subscriptions ALTER COLUMN source SET DEFAULT 'paid';
ALTER TABLE user_subscriptions ALTER COLUMN source SET NOT NULL;

-- Backfill existing subscriptions as paid (for drifted installs with NULLs)
UPDATE user_subscriptions SET source = 'paid' WHERE source IS NULL OR source = '';

COMMENT ON COLUMN user_subscriptions.source IS 'Subscription source: paid, gift';
