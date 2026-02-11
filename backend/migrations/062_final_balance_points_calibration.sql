-- 062: Final calibration of balance/points split
--
-- Formula: remaining_paid = max(redeemed - balance_consumed, 0)
--          correct_balance = min(remaining_paid, balance + points)
--          correct_points  = (balance + points) - correct_balance
--          adjustment      = correct_balance - current_balance
--
-- Where balance_consumed = SUM(actual_cost) from usage_logs WHERE billing_type=0
-- Positive adjustment = restore points → balance (previous over-transfer)
-- Negative adjustment = move balance → points (previous under-transfer)
--
-- Affected: ~17 users, net ~825 restored to balance

-- Audit table
CREATE TABLE IF NOT EXISTS user_final_calibration_audit (
    user_id          BIGINT PRIMARY KEY,
    redeemed_total   NUMERIC(20,8) NOT NULL DEFAULT 0,
    balance_consumed NUMERIC(20,8) NOT NULL DEFAULT 0,
    remaining_paid   NUMERIC(20,8) NOT NULL DEFAULT 0,
    correct_balance  NUMERIC(20,8) NOT NULL,
    correct_points   NUMERIC(20,8) NOT NULL,
    adjustment       NUMERIC(20,8) NOT NULL,
    balance_before   NUMERIC(20,8) NOT NULL,
    points_before    NUMERIC(20,8) NOT NULL,
    migrated_at      TIMESTAMPTZ
);

-- Capture users needing correction
INSERT INTO user_final_calibration_audit
    (user_id, redeemed_total, balance_consumed, remaining_paid,
     correct_balance, correct_points, adjustment,
     balance_before, points_before)
SELECT
    u.id,
    COALESCE(rc.redeemed, 0),
    COALESCE(bc.bal_consumed, 0),
    GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(bc.bal_consumed, 0), 0),
    LEAST(
        GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(bc.bal_consumed, 0), 0),
        u.balance + u.points
    ),
    (u.balance + u.points) - LEAST(
        GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(bc.bal_consumed, 0), 0),
        u.balance + u.points
    ),
    LEAST(
        GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(bc.bal_consumed, 0), 0),
        u.balance + u.points
    ) - u.balance,
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
    SELECT user_id, SUM(actual_cost) AS bal_consumed
    FROM usage_logs
    WHERE billing_type = 0 AND user_id IS NOT NULL
    GROUP BY user_id
) bc ON bc.user_id = u.id
WHERE u.deleted_at IS NULL
  AND (u.balance > 0 OR u.points > 0)
  AND ROUND((
      LEAST(
          GREATEST(COALESCE(rc.redeemed, 0) - COALESCE(bc.bal_consumed, 0), 0),
          u.balance + u.points
      ) - u.balance
  )::numeric, 2) != 0
  AND NOT EXISTS (
      SELECT 1 FROM user_final_calibration_audit a WHERE a.user_id = u.id
  );

-- Apply adjustment (with state guard)
WITH calibrated AS (
    UPDATE users u
    SET balance = u.balance + a.adjustment,
        points  = u.points  - a.adjustment
    FROM user_final_calibration_audit a
    WHERE a.user_id = u.id
      AND a.migrated_at IS NULL
      AND u.balance = a.balance_before
      AND u.points  = a.points_before
    RETURNING a.user_id
)
UPDATE user_final_calibration_audit a
SET migrated_at = now()
FROM calibrated c
WHERE a.user_id = c.user_id;
