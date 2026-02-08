-- Add source field to redeem_codes table
ALTER TABLE redeem_codes
  ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'paid';

-- Ensure constraints are enforced even if column already exists (drifted DB)
ALTER TABLE redeem_codes ALTER COLUMN source SET DEFAULT 'paid';
ALTER TABLE redeem_codes ALTER COLUMN source SET NOT NULL;

COMMENT ON COLUMN redeem_codes.source IS 'Redeem code source: paid, gift';
