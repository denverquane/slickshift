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

-- rebuild shift_errors so id is an INTEGER PRIMARY KEY, which is what auto-assigns ids (INT PRIMARY KEY left them NULL)
CREATE TABLE shift_errors_new (
    id INTEGER PRIMARY KEY,
    user_id UNSIGNED BIG INT NOT NULL,
    code CHAR(29) NOT NULL,
    platform TEXT NOT NULL,
    error TEXT NOT NULL,
    created_unix UNSIGNED BIG INT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (code) REFERENCES shift_codes (code) ON DELETE CASCADE
);
INSERT INTO shift_errors_new (user_id, code, platform, error, created_unix)
    SELECT user_id, code, platform, error, created_unix FROM shift_errors ORDER BY created_unix;
DROP TABLE shift_errors;
ALTER TABLE shift_errors_new RENAME TO shift_errors;
