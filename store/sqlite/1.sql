CREATE TABLE users (
    id UNSIGNED BIG INT PRIMARY KEY,
    platform TEXT,
    should_dm BOOLEAN,
    redemption_unix UNSIGNED BIG INT, -- last time ANY code was redeemed (success or not)
    updated_unix UNSIGNED BIG INT NOT NULL,
    created_unix UNSIGNED BIG INT NOT NULL
);

CREATE TABLE user_cookies (
    user_id UNSIGNED BIG INT PRIMARY KEY,
    encrypted_cookie_json TEXT NOT NULL,
    updated_unix UNSIGNED BIG INT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE shift_codes (
    code CHAR(29) NOT NULL PRIMARY KEY,
    game TEXT NOT NULL,
    reward TEXT,
    source TEXT, -- where the code was sourced, if not by a user (twitter, instagram, etc)
    user_id BIG INT, -- who added/registered the code
    success_unix UNSIGNED BIG INT, -- last time the code was redeemed successfully
    created_unix UNSIGNED BIG INT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE TABLE redemptions (
    code CHAR(29) NOT NULL,
    user_id BIG INT NOT NULL,
    platform TEXT NOT NULL,
    status TEXT NOT NULL,

    created_unix UNSIGNED BIG INT NOT NULL,

    FOREIGN KEY (code) REFERENCES shift_codes (code) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (code, user_id, platform)
);