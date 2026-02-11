-- 058: Migrate invite bonus balance to points
--
-- Scope: Users who invited others and still have balance > 0.
--        Each invite = 30 points. Transfer min(invite_count * 30, balance)
--        from balance to points.
--
-- Affected: ~25 users, ~2,505.17 total

-- Audit table
CREATE TABLE IF NOT EXISTS user_invite_balance_to_points_audit (
    user_id         BIGINT PRIMARY KEY,
    invited_count   INT NOT NULL,
    transfer_amount NUMERIC(20,8) NOT NULL,
    balance_before  NUMERIC(20,8) NOT NULL,
    points_before   NUMERIC(20,8) NOT NULL DEFAULT 0,
    migrated_at     TIMESTAMPTZ
);

-- Capture eligible users
INSERT INTO user_invite_balance_to_points_audit
    (user_id, invited_count, transfer_amount, balance_before, points_before)
SELECT
    u.id,
    inv.cnt,
    LEAST(inv.cnt * 30, u.balance),
    u.balance,
    u.points
FROM users u
JOIN (
    SELECT invited_by_user_id, COUNT(*) AS cnt
    FROM users
    WHERE invited_by_user_id IS NOT NULL AND deleted_at IS NULL
    GROUP BY invited_by_user_id
) inv ON inv.invited_by_user_id = u.id
WHERE u.deleted_at IS NULL
  AND u.balance > 0
  AND NOT EXISTS (
      SELECT 1 FROM user_invite_balance_to_points_audit a WHERE a.user_id = u.id
  );

-- Transfer: balance → points (with state guard)
WITH migrated AS (
    UPDATE users u
    SET points  = u.points + a.transfer_amount,
        balance = u.balance - a.transfer_amount
    FROM user_invite_balance_to_points_audit a
    WHERE a.user_id = u.id
      AND a.migrated_at IS NULL
      AND u.balance  = a.balance_before
      AND u.points   = a.points_before
    RETURNING a.user_id
)
UPDATE user_invite_balance_to_points_audit a
SET migrated_at = now()
FROM migrated m
WHERE a.user_id = m.user_id;
