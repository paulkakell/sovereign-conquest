package schema

// Additive and idempotent: old accounts remain active, and version-zero tokens
// keep working until the account's credentials, permissions or status change.
const userManagementDDL = `
ALTER TABLE users ADD COLUMN IF NOT EXISTS session_version bigint NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS account_revision bigint NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS account_status text NOT NULL DEFAULT 'active'
	CHECK (account_status IN ('active', 'suspended', 'banned'));
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended_until timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderation_reason text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderated_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_users_username_id ON users(username, id);
CREATE INDEX IF NOT EXISTS idx_users_account_status ON users(account_status);
`
