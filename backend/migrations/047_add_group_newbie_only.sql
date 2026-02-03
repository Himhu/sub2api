-- Add newbie-only flag to groups
-- This flag indicates that the group is only visible to users who have not used any redeem codes

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS is_newbie_only BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_groups_is_newbie_only
  ON groups (is_newbie_only);

-- Add comment
COMMENT ON COLUMN groups.is_newbie_only IS 'If true, only users without redeem history can see/use this group';
