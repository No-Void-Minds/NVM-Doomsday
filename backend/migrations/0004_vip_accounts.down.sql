DROP INDEX IF EXISTS registrations_vip_account_active_key;
ALTER TABLE registrations DROP CONSTRAINT IF EXISTS registrations_vip_account_only_vip;
ALTER TABLE registrations DROP COLUMN IF EXISTS vip_account_id;
DROP INDEX IF EXISTS vip_accounts_account_key_key;
DROP TABLE IF EXISTS vip_accounts;
