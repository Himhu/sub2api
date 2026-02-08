-- Rename is_newbie_only to is_points_only
-- Points-only groups are exclusively for points-based billing users

ALTER TABLE groups RENAME COLUMN is_newbie_only TO is_points_only;

ALTER INDEX IF EXISTS idx_groups_is_newbie_only RENAME TO idx_groups_is_points_only;

COMMENT ON COLUMN groups.is_points_only IS 'If true, this group is points-only: requires points > 0, enforces points billing';
