-- Add points field to users table for gift credits (invites/promotions)
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS points DECIMAL(20,8) NOT NULL DEFAULT 0;

-- Ensure constraints are enforced even if column already exists (drifted DB)
ALTER TABLE users ALTER COLUMN points SET DEFAULT 0;
ALTER TABLE users ALTER COLUMN points SET NOT NULL;

-- Partial index for users with positive points (active users only)
CREATE INDEX IF NOT EXISTS idx_users_points
  ON users (points)
  WHERE points > 0 AND deleted_at IS NULL;

COMMENT ON COLUMN users.points IS 'Gift points from invites/promotions (separate from paid balance)';
