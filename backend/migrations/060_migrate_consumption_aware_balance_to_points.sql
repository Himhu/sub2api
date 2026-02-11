-- 060: Migrate gifted balance to points (consumption-aware)
--
-- Scope: Users with balance > 0 whose consumption exceeds their
--        redeem-code total (partially or fully). The paid portion
--        has been consumed, so remaining balance is gifted.
--
-- Formula: remaining_paid = min(max(redeemed - consumed, 0), balance)
--          transfer       = balance - remaining_paid
--
-- Affected: ~61 users, ~3,387.25 total

-- Audit table
CREATE TABLE IF NOT EXISTS user_consumption_aware_balance_audit (
    user_id          BIGINT PRIMARY KEY,
    redeemed_total   NUMERIC(20,8) NOT NULL DEFAULT 0,
    total_consumed   NUMERIC(20,8) NOT NULL DEFAULT 0,
    remaining_paid   NUMERIC(20,8) NOT NULL DEFAULT 0,
    transfer_amount  NUMERIC(20,8) NOT NULL,
    balance_before   NUMERIC(20,8) NOT NULL,
    points_before    NUMERIC(20,8) NOT NULL DEFAULT 0,
    migrated_at      TIMESTAMPTZ
);

-- Capture eligible users
INSERT INTO user_consumption_aware_balance_audit
    (user_id, redeemed_total, total_consumed, remaining_paid,
     transfer_amount, balance_before, points_before)
SELECT
    u.id,
    COALESCE(rc.redeemed, 0),
    COALESCE(ul.consumed, 0),
    LEAST(GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(ul.consumed, 0), 0), u.balance),
    u.balance - LEAST(GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(ul.consumed, 0), 0), u.balance),
    u.balance,
    u.points
FROM users u
LEFT JOIN (
    SELECT used_by, SUM(value) AS redeemed
    FROM redeem_codes
    WHERE used_by IS NOT NULL
    GROUP BY used_by
) rc ON rc.used_by = u.id
LEFT JOIN (
    SELECT user_id, SUM(actual_cost) AS consumed
    FROM usage_logs
    WHERE user_id IS NOT NULL
    GROUP BY user_id
) ul ON ul.user_id = u.id
WHERE u.deleted_at IS NULL
  AND u.balance > 0
  AND u.balance - LEAST(GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(ul.consumed, 0), 0), u.balance) > 0
  AND NOT EXISTS (
      SELECT 1 FROM user_consumption_aware_balance_audit a WHERE a.user_id = u.id
  );

-- Transfer: balance → points (with state guard)
WITH migrated AS (
    UPDATE users u
    SET points  = u.points  + a.transfer_amount,
        balance = u.balance - a.transfer_amount
    FROM user_consumption_aware_balance_audit a
    WHERE a.user_id = u.id
      AND a.migrated_at IS NULL
      AND u.balance = a.balance_before
      AND u.points  = a.points_before
    RETURNING a.user_id
)
UPDATE user_consumption_aware_balance_audit a
SET migrated_at = now()
FROM migrated m
WHERE a.user_id = m.user_id;
