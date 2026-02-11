-- 061: Fix 060 over-transfer (subscription billing was wrongly included)
--
-- 060 summed ALL usage_logs.actual_cost as "consumed from balance",
-- but only billing_type=0 (Balance) actually deducts from balance.
-- Subscription billing (type=1) was included, overstating consumption
-- by ~10,700, causing ~3,131 excess transfer from balance to points.
--
-- This migration reverses the excess: points → balance.

-- Audit table
CREATE TABLE IF NOT EXISTS user_060_correction_audit (
    user_id            BIGINT PRIMARY KEY,
    wrong_transfer     NUMERIC(20,8) NOT NULL,
    correct_transfer   NUMERIC(20,8) NOT NULL,
    reversal_amount    NUMERIC(20,8) NOT NULL,
    balance_before     NUMERIC(20,8) NOT NULL,
    points_before      NUMERIC(20,8) NOT NULL,
    migrated_at        TIMESTAMPTZ
);

-- Capture users needing correction
INSERT INTO user_060_correction_audit
    (user_id, wrong_transfer, correct_transfer, reversal_amount,
     balance_before, points_before)
SELECT
    a.user_id,
    a.transfer_amount,
    GREATEST(
        a.balance_before - LEAST(
            GREATEST(a.redeemed_total - COALESCE(bl.balance_consumed, 0), 0),
            a.balance_before
        ),
        0
    ),
    a.transfer_amount - GREATEST(
        a.balance_before - LEAST(
            GREATEST(a.redeemed_total - COALESCE(bl.balance_consumed, 0), 0),
            a.balance_before
        ),
        0
    ),
    u.balance,
    u.points
FROM user_consumption_aware_balance_audit a
JOIN users u ON u.id = a.user_id
LEFT JOIN (
    SELECT user_id, SUM(actual_cost) AS balance_consumed
    FROM usage_logs
    WHERE billing_type = 0 AND user_id IS NOT NULL
    GROUP BY user_id
) bl ON bl.user_id = a.user_id
WHERE a.migrated_at IS NOT NULL
  AND a.transfer_amount > GREATEST(
        a.balance_before - LEAST(
            GREATEST(a.redeemed_total - COALESCE(bl.balance_consumed, 0), 0),
            a.balance_before
        ),
        0
    )
  AND NOT EXISTS (
      SELECT 1 FROM user_060_correction_audit c WHERE c.user_id = a.user_id
  );

-- Reverse: points → balance (with state guard)
WITH corrected AS (
    UPDATE users u
    SET balance = u.balance + c.reversal_amount,
        points  = u.points  - c.reversal_amount
    FROM user_060_correction_audit c
    WHERE c.user_id = u.id
      AND c.migrated_at IS NULL
      AND u.balance = c.balance_before
      AND u.points  = c.points_before
    RETURNING c.user_id
)
UPDATE user_060_correction_audit c
SET migrated_at = now()
FROM corrected m
WHERE c.user_id = m.user_id;
