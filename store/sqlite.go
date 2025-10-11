package store

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/denverquane/slickshift/shift"

	_ "modernc.org/sqlite"
)

//go:embed sqlite/*.sql
var schemaFS embed.FS

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
	currentVersion, err := getVersion(db)
	if err != nil {
		return nil, err
	}
	slog.Info("initialized db", "version", currentVersion)

	err = fs.WalkDir(schemaFS, "sqlite", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			name := strings.TrimSuffix(d.Name(), ".sql")
			val, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				return err
			}
			if val > currentVersion {
				contents, err := schemaFS.ReadFile(path)
				if err != nil {
					return err
				}
				slog.Info("applying migration script", "version", val, "path", path)
				_, err = db.Exec(string(contents))
				if err != nil {
					return err
				}
				err = setVersion(db, val)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Sqlite{db: db, encryptor: encryptor}, nil
}

func getVersion(db *sql.DB) (int64, error) {
	var version int64
	err := db.QueryRow("PRAGMA user_version;").Scan(&version)
	if err != nil {
		return -1, err
	}
	return version, nil
}
func setVersion(db *sql.DB, version int64) error {
	_, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d;", version))
	return err
}

func (s *Sqlite) UserExists(userID string) bool {
	return s.exists("users", "id", userID)
}

func (s *Sqlite) AddUser(userID string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO users (id, updated_unix, created_unix) VALUES (?, ?, ?)", userID, t, t)
	return err
}

func (s *Sqlite) GetUserPlatformAndDM(userID string) (string, bool, error) {
	var dm sql.NullBool
	var platform sql.NullString
	err := s.db.QueryRow("SELECT platform, should_dm FROM users WHERE id = ?", userID).Scan(&platform, &dm)
	if err != nil {
		return "", false, err
	}
	if !platform.Valid {
		return "", dm.Valid && dm.Bool, nil
	}
	return platform.String, dm.Valid, nil
}

func (s *Sqlite) SetUserDM(userID string, dm bool) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("UPDATE users SET should_dm = ?, updated_unix = ? WHERE id = ?", dm, t, userID)
	return err
}

func (s *Sqlite) SetUserPlatform(userID, platform string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("UPDATE users SET platform = ?, updated_unix = ? WHERE id = ?", platform, t, userID)
	return err
}

func (s *Sqlite) UserCookiesExists(userID string) bool {
	return s.exists("user_cookies", "user_id", userID)
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

func (s *Sqlite) DeleteUserCookies(userID string) error {
	_, err := s.db.Exec("DELETE FROM user_cookies WHERE user_id=?", userID)
	return err
}

func (s *Sqlite) CodeExists(code string) bool {
	return s.exists("shift_codes", "code", code)
}

func (s *Sqlite) AddCode(code, game string, userID *string, source *string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT OR IGNORE INTO shift_codes (code, game, user_id, source, created_unix) VALUES (?, ?, ?, ?, ?)", code, game, userID, source, t)
	if err != nil {
		return err
	}
	return nil
}

func (s *Sqlite) SetCodeRewardAndSuccess(code, reward string, success bool) (bool, error) {
	t := time.Now().Unix()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false, err
	}

	if success {
		_, err = tx.Exec("UPDATE shift_codes SET success_unix = ? WHERE code = ?", t, code)
		if err != nil {
			tx.Rollback()
			return false, err
		}
	}
	res, err := tx.Exec("UPDATE shift_codes SET reward = ? WHERE code = ? AND reward IS NULL", reward, code)
	if err != nil {
		tx.Rollback()
		return false, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return false, err
	}

	return n == 1, tx.Commit()
}

func (s *Sqlite) GetValidCodesNotRedeemedForUser(userID, platform string) ([]string, error) {
	// grab codes that the user hasn't redeemed for the platform before,
	// AND, if the code hasn't been marked as expired/invalid before

	// TODO maybe have a minimum threshold on how many expiries have to be marked before we ignore?
	query := "SELECT sc.code FROM shift_codes sc WHERE " +
		"NOT EXISTS (SELECT 1 FROM redemptions r WHERE r.code = sc.code AND r.user_id = ? AND r.platform = ?) AND " +
		"NOT EXISTS (SELECT 1 FROM redemptions r WHERE r.code = sc.code AND (r.status = ? OR r.status = ?))" +
		"ORDER BY success_unix DESC" // sort preferentially for the most recently-successful codes
	rows, err := s.db.Query(query, userID, platform, shift.EXPIRED, shift.NOT_EXIST)
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

func (s *Sqlite) GetAllDecryptedUserCookiesSorted(limit int64) ([]UserCookies, error) {
	rows, err := s.db.Query("SELECT c.user_id, c.encrypted_cookie_json FROM user_cookies c JOIN users u ON c.user_id = u.id ORDER BY u.redemption_unix LIMIT ?", limit)
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
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO redemptions (code, user_id, platform, status, created_unix) VALUES (?, ?, ?, ?, ?)", code, userID, platform, status, t)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec("UPDATE users SET redemption_unix = ? WHERE id = ?", t, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Sqlite) GetStatistics(userID string) (Statistics, error) {
	var stats Statistics
	var totalUsers, steam, epic, xbox, psn int64
	var totalRedeem, expired, already, success int64
	var totalCodes, goldenKey, other, unknown int64
	err := s.db.QueryRow(`
    SELECT 
        (SELECT COUNT(*) FROM users),
        (SELECT COUNT(*) FROM users WHERE platform = ?),
        (SELECT COUNT(*) FROM users WHERE platform = ?),
        (SELECT COUNT(*) FROM users WHERE platform = ?),
        (SELECT COUNT(*) FROM users WHERE platform = ?),
        (SELECT COUNT(*) FROM shift_codes),
        (SELECT COUNT(*) FROM shift_codes WHERE reward = ?),
        (SELECT COUNT(*) FROM shift_codes WHERE reward IS NOT NULL AND reward != ?),
        (SELECT COUNT(*) FROM shift_codes WHERE reward IS NULL),
        (SELECT COUNT(*) FROM redemptions),
        (SELECT COUNT(*) FROM redemptions WHERE status = ?),
        (SELECT COUNT(*) FROM redemptions WHERE status = ?),                                      
        (SELECT COUNT(*) FROM redemptions WHERE status = ?)
`,
		shift.Steam, shift.Epic, shift.XboxLive, shift.PSN,

		shift.GoldenKey,
		shift.GoldenKey,

		shift.EXPIRED,
		shift.ALREADY_REDEEMED,
		shift.SUCCESS,
	).Scan(
		&totalUsers,
		&steam,
		&epic,
		&xbox,
		&psn,

		&totalCodes,
		&goldenKey,
		&other,
		&unknown,

		&totalRedeem,
		&expired,
		&already,
		&success,
	)
	if err != nil {
		return stats, err
	}
	stats.Users = map[string]int64{
		"total": totalUsers,
		"steam": steam,
		"epic":  epic,
		"xbox":  xbox,
		"psn":   psn,
	}
	stats.Codes = map[string]int64{
		"total":      totalCodes,
		"golden_key": goldenKey,
		"other":      other,
		"unknown":    unknown,
	}
	stats.Redemptions = map[string]int64{
		"total":            totalRedeem,
		"expired":          expired,
		"already_redeemed": already,
		"success":          success,
	}
	return stats, nil
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

func (s *Sqlite) Close() error {
	return s.db.Close()
}
