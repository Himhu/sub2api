-- 059: Migrate remaining gifted balance to points
--
-- Scope: Users whose balance exceeds their total redeem-code value.
--        The excess (balance - redeemed_total) is admin-gifted credit
--        that belongs in points.
--
-- Affected: ~40 users, ~1,081.19 total

-- Audit table
CREATE TABLE IF NOT EXISTS user_remaining_gifted_balance_audit (
    user_id          BIGINT PRIMARY KEY,
    redeemed_total   NUMERIC(20,8) NOT NULL DEFAULT 0,
    transfer_amount  NUMERIC(20,8) NOT NULL,
    balance_before   NUMERIC(20,8) NOT NULL,
    points_before    NUMERIC(20,8) NOT NULL DEFAULT 0,
    migrated_at      TIMESTAMPTZ
);

-- Capture eligible users
INSERT INTO user_remaining_gifted_balance_audit
    (user_id, redeemed_total, transfer_amount, balance_before, points_before)
SELECT
    u.id,
    COALESCE(rc.redeemed, 0),
    u.balance - COALESCE(rc.redeemed, 0),
    u.balance,
    u.points
FROM users u
LEFT JOIN (
    SELECT used_by, SUM(value) AS redeemed
    FROM redeem_codes
    WHERE used_by IS NOT NULL
    GROUP BY used_by
) rc ON rc.used_by = u.id
WHERE u.deleted_at IS NULL
  AND u.balance > 0
  AND u.balance > COALESCE(rc.redeemed, 0)
  AND NOT EXISTS (
      SELECT 1 FROM user_remaining_gifted_balance_audit a WHERE a.user_id = u.id
  );

-- Transfer: balance → points (with state guard)
WITH migrated AS (
    UPDATE users u
    SET points  = u.points  + a.transfer_amount,
        balance = u.balance - a.transfer_amount
    FROM user_remaining_gifted_balance_audit a
    WHERE a.user_id = u.id
      AND a.migrated_at IS NULL
      AND u.balance = a.balance_before
      AND u.points  = a.points_before
    RETURNING a.user_id
)
UPDATE user_remaining_gifted_balance_audit a
SET migrated_at = now()
FROM migrated m
WHERE a.user_id = m.user_id;
