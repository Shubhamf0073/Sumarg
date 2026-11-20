-- Parent OTP (M1.3): indexes for the per-mobile / per-school limits and the 24 h worker purge. The tables are
-- empty before go-live, so plain CREATE INDEX inside the migration transaction is fine.

-- +goose Up
CREATE INDEX otp_recent ON otp_challenges (tenant_id, mobile, created_at);
CREATE INDEX otp_expiry ON otp_challenges (created_at);
CREATE INDEX notifications_otp ON notifications (tenant_id, created_at) WHERE event = 'otp';

-- +goose Down
DROP INDEX notifications_otp;
DROP INDEX otp_expiry;
DROP INDEX otp_recent;
