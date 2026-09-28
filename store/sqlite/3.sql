ALTER TABLE users ADD COLUMN alert TEXT; -- last problem the user was notified about (session expired, etc), so they're only notified once
ALTER TABLE users ADD COLUMN alert_unix UNSIGNED BIG INT;

ALTER TABLE user_cookies ADD COLUMN verified_unix UNSIGNED BIG INT; -- last time the session was confirmed to be logged in to SHiFT

-- only final outcomes should be recorded as redemptions. Remove anything else (like needing to link a 2K account first),
-- so those codes get retried. Worst case, retrying a code that was actually redeemed returns "already redeemed"
DELETE FROM redemptions WHERE status NOT IN (
    'Your code was successfully redeemed',
    'This SHiFT code has already been redeemed',
    'This SHiFT code has expired',
    'This SHiFT code does not exist'
);
