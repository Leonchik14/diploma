DROP INDEX IF EXISTS idx_password_reset_codes_user_id;
DROP INDEX IF EXISTS idx_password_reset_codes_email_expires;
DROP INDEX IF EXISTS idx_password_reset_codes_email_created;
DROP TABLE IF EXISTS password_reset_codes;
