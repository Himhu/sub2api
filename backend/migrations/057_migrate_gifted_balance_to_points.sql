-- 057: Migrate gifted balance to points for users who never paid
--
-- Scope: Users with balance > 0 who have never used a redeem code
--        and have no active subscription. These balances are admin-gifted
--        credits that belong in the points field.
--
-- Affected: ~609 users, total ~22,793.00
-- Includes agent users (10 users, ~90.55)

-- Audit table (no FK cascade — preserve audit even if user is deleted)
CREATE TABLE IF NOT EXISTS user_balance_to_points_audit (
    user_id        BIGINT PRIMARY KEY,
    balance_before NUMERIC(20,8) NOT NULL,
    points_before  NUMERIC(20,8) NOT NULL DEFAULT 0,
    migrated_at    TIMESTAMPTZ
);

-- Capture eligible users (skip if already captured)
INSERT INTO user_balance_to_points_audit (user_id, balance_before, points_before)
SELECT u.id, u.balance, u.points
FROM users u
WHERE u.balance > 0
  AND NOT EXISTS (
      SELECT 1 FROM redeem_codes rc WHERE rc.used_by = u.id
  )
  AND NOT EXISTS (
      SELECT 1 FROM user_subscriptions us
      WHERE us.user_id = u.id AND us.deleted_at IS NULL
  )
  AND NOT EXISTS (
      SELECT 1 FROM user_balance_to_points_audit a WHERE a.user_id = u.id
  );

-- Move balance → points, with state guard to prevent double-add
WITH migrated AS (
    UPDATE users u
    SET points  = u.points + a.balance_before,
        balance = 0
    FROM user_balance_to_points_audit a
    WHERE a.user_id = u.id
      AND a.migrated_at IS NULL
      AND u.balance = a.balance_before
      AND u.points  = a.points_before
    RETURNING a.user_id
)
UPDATE user_balance_to_points_audit a
SET migrated_at = now()
FROM migrated m
WHERE a.user_id = m.user_id;
