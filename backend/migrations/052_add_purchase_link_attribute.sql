-- Migration: Add purchase_link user attribute definition

-- +goose Up
-- +goose StatementBegin

INSERT INTO user_attribute_definitions (key, name, description, type, options, required, validation, placeholder, display_order, enabled, created_at, updated_at)
SELECT
    'purchase_link',
    '购买链接',
    '购买链接',
    'url',
    '[]'::jsonb,
    false,
    '{}'::jsonb,
    '请输入购买链接',
    COALESCE((SELECT MAX(display_order) + 1 FROM user_attribute_definitions WHERE deleted_at IS NULL), 0),
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM user_attribute_definitions WHERE key = 'purchase_link' AND deleted_at IS NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM user_attribute_values
WHERE attribute_id IN (
    SELECT id FROM user_attribute_definitions WHERE key = 'purchase_link' AND deleted_at IS NULL
);

UPDATE user_attribute_definitions
SET deleted_at = NOW()
WHERE key = 'purchase_link' AND deleted_at IS NULL;

-- +goose StatementEnd
