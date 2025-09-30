package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed sqlite.sql
var schema string

type Sqlite struct {
	db        *sql.DB
	encryptor *Encryptor
}

func NewSqliteStore(filepath string, encryptor *Encryptor) (Store, error) {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return &Sqlite{db: db, encryptor: encryptor}, nil
}

func (s *Sqlite) UserExists(userID string) bool {
	return s.exists("users", "id", userID)
}

func (s *Sqlite) AddUser(userID string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO users (id, updated_unix, created_unix) VALUES (?, ?, ?)", userID, t, t)
	return err
}

func (s *Sqlite) SetUserDM(userID string, dm bool) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("UPDATE users SET should_dm = ?, updated_unix = ? WHERE id = ?", dm, t, userID)
	return err
}

func (s *Sqlite) GetUserDM(userID string) (bool, error) {
	var value sql.NullBool
	err := s.db.QueryRow("SELECT should_dm FROM users WHERE id = ?", userID).Scan(&value)
	if err != nil {
		return false, err
	}
	return value.Valid && value.Bool, nil
}

func (s *Sqlite) SetUserPlatform(userID, platform string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("UPDATE users SET platform = ?, updated_unix = ? WHERE id = ?", platform, t, userID)
	return err
}

func (s *Sqlite) GetUserPlatform(userID string) (string, error) {
	var platform sql.NullString
	err := s.db.QueryRow("SELECT platform FROM users WHERE id = ?", userID).Scan(&platform)
	if err != nil {
		return "", err
	}
	if platform.Valid {
		return platform.String, nil
	}
	return "", nil
}

func (s *Sqlite) EncryptAndSetUserCookies(userID string, cookies []*http.Cookie) error {
	cookieJson, err := json.Marshal(cookies)
	if err != nil {
		return err
	}
	encrypted, err := s.encryptor.Encrypt(string(cookieJson))
	if err != nil {
		return err
	}
	t := time.Now().Unix()
	_, err = s.db.Exec("INSERT INTO user_cookies (user_id, encrypted_cookie_json, updated_unix) VALUES (?, ?, ?) ON CONFLICT (user_id) DO UPDATE SET encrypted_cookie_json = excluded.encrypted_cookie_json, updated_unix = excluded.updated_unix", userID, encrypted, t)
	return err
}

func (s *Sqlite) GetDecryptedUserCookies(userID string) ([]*http.Cookie, error) {
	var cipherText string
	err := s.db.QueryRow("SELECT encrypted_cookie_json FROM user_cookies WHERE user_id=?", userID).Scan(&cipherText)
	if err != nil {
		return nil, err
	}
	cookieJson, err := s.encryptor.Decrypt(cipherText)
	if err != nil {
		return nil, err
	}
	var cookies []*http.Cookie
	err = json.Unmarshal([]byte(cookieJson), &cookies)
	if err != nil {
		return nil, err
	}
	return cookies, nil
}

func (s *Sqlite) CodeExists(code string) bool {
	return s.exists("shift_codes", "code", code)
}

func (s *Sqlite) AddCode(code, game string, userID *string, source *string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT OR IGNORE INTO shift_codes (code, game, user_id, source, created_unix) VALUES (?, ?, ?, ?, ?)", code, game, userID, source, t)
	return err
}

func (s *Sqlite) SetCodeRewardIfNotSet(code, reward string) (bool, error) {
	res, err := s.db.Exec("UPDATE shift_codes SET reward = ? WHERE code = ? AND reward IS NULL", reward, code)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (s *Sqlite) GetCodesNotRedeemedForUser(userID, platform string) ([]string, error) {
	rows, err := s.db.Query("SELECT sc.code FROM shift_codes sc WHERE NOT EXISTS (SELECT 1 FROM redemptions r WHERE r.code = sc.code AND r.user_id = ? AND r.platform = ?)", userID, platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code string
		err = rows.Scan(&code)
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func (s *Sqlite) GetAllDecryptedUserCookies() ([]UserCookies, error) {
	rows, err := s.db.Query("SELECT user_id, encrypted_cookie_json FROM user_cookies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var userCookies []UserCookies
	for rows.Next() {
		var userID string
		var cipherText string
		err = rows.Scan(&userID, &cipherText)
		if err != nil {
			return nil, err
		}
		cookieJson, err := s.encryptor.Decrypt(cipherText)
		if err != nil {
			return nil, err
		}
		var cookies []*http.Cookie
		err = json.Unmarshal([]byte(cookieJson), &cookies)
		if err != nil {
			return nil, err
		}
		userCookies = append(userCookies, UserCookies{
			UserID:  userID,
			Cookies: cookies,
		})
	}
	return userCookies, nil
}

func (s *Sqlite) GetRecentRedemptionsForUser(userID string, status string, quantity int) ([]Redemption, error) {
	if quantity <= 0 {
		return nil, nil
	}
	var insert string
	args := []any{
		userID,
	}
	if status != "" {
		insert = "AND r.status = ? "
		args = append(args, status)
	}
	query := fmt.Sprintf("SELECT r.code, r.platform, r.status, r.created_unix, s.game, s.reward FROM redemptions r JOIN shift_codes s ON r.code = s.code WHERE r.user_id = ? %sORDER BY r.created_unix DESC LIMIT %d", insert, quantity)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRedemptions(rows)
}

func scanRedemptions(rows *sql.Rows) ([]Redemption, error) {
	var redemptions []Redemption
	for rows.Next() {
		var redemption Redemption
		err := rows.Scan(&redemption.Code, &redemption.Platform, &redemption.Status, &redemption.TimeUnix, &redemption.Game, &redemption.Reward)
		if err != nil {
			return nil, err
		}
		redemptions = append(redemptions, redemption)
	}
	return redemptions, nil
}

func (s *Sqlite) AddRedemption(userID, code, platform string, status string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO redemptions (code, user_id, platform, status, created_unix) VALUES (?, ?, ?, ?, ?)", code, userID, platform, status, t)
	return err
}

func (s *Sqlite) exists(table, field, value string) bool {
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE %s = ?)", table, field)
	var exists bool
	err := s.db.QueryRow(query, value).Scan(&exists)
	if err != nil {
		log.Println(err)
		return false
	}
	return exists
}
