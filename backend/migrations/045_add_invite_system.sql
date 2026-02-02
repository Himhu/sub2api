-- 邀请注册系统（简化版：代理即用户）

-- 1. 用户表新增代理相关字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_agent BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS parent_agent_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_is_agent ON users(is_agent) WHERE is_agent = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_parent_agent ON users(parent_agent_id);

COMMENT ON COLUMN users.is_agent IS '是否是代理';
COMMENT ON COLUMN users.parent_agent_id IS '上级代理用户ID，NULL表示顶级代理';

-- 2. 用户表新增邀请相关字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code VARCHAR(32) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS belong_agent_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_invite_code ON users(invite_code);
CREATE INDEX IF NOT EXISTS idx_users_invited_by ON users(invited_by_user_id);
CREATE INDEX IF NOT EXISTS idx_users_belong_agent ON users(belong_agent_id);

COMMENT ON COLUMN users.invite_code IS '用户专属邀请码';
COMMENT ON COLUMN users.invited_by_user_id IS '邀请人用户ID';
COMMENT ON COLUMN users.belong_agent_id IS '所属代理用户ID（用于显示联系方式）';
