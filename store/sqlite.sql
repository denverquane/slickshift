CREATE TABLE IF NOT EXISTS users (
    id UNSIGNED BIG INT PRIMARY KEY,
    platform TEXT NOT NULL,
    updated_unix BIG INT
);

CREATE TABLE IF NOT EXISTS user_cookies (
    user_id UNSIGNED BIG INT PRIMARY KEY,
    cookie TEXT NOT NULL,
    updated_unix BIG INT,

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS shift_codes (
    code CHAR[29] PRIMARY KEY,
    reward TEXT,
    source TEXT, -- where the code was sourced, if not by a user (twitter, instagram, etc)
    user_id BIG INT, -- who added/registered the code
    added_unix BIG INT NOT NULL,
    redeemed_unix BIG INT, -- last time the code was successfully redeemed

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS redemptions (
    code CHAR(29) NOT NULL,
    user_id BIG INT NOT NULL,
    platform TEXT NOT NULL,
    success INT NOT NULL,

    time_unix BIG INT NOT NULL,

    FOREIGN KEY (code) REFERENCES shift_codes (code) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (code, user_id, platform)
);