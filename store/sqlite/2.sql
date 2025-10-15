CREATE TABLE shift_errors (
    id INT PRIMARY KEY,
    user_id UNSIGNED BIG INT NOT NULL,
    code CHAR(29) NOT NULL,
    platform TEXT NOT NULL,
    error TEXT NOT NULL,
    created_unix UNSIGNED BIG INT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (code) REFERENCES shift_codes (code)
);