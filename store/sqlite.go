package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed sqlite.sql
var schema string

type Sqlite struct {
	db *sql.DB
}

func NewSqliteStore(filepath string) (Store, error) {
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

	return &Sqlite{db: db}, nil
}

func (s *Sqlite) UserExists(userID string) bool {
	return s.exists("users", "id", userID)
}

func (s *Sqlite) AddUser(userID string, platform Platform) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO users (id, platform, updated_unix) VALUES (?, ?, ?)", userID, platform, t)
	return err
}

func (s *Sqlite) SetUserPlatform(userID string, platform Platform) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO users (id, platform, updated_unix) VALUES (?, ?, ?) ON CONFLICT (id) DO UPDATE SET platform = excluded.platform, updated_unix = excluded.updated_unix", userID, platform, t)
	return err
}

func (s *Sqlite) EncryptAndSetUserCookie(userID string, cookie string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO user_cookies (user_id, cookie, updated_unix) VALUES (?, ?, ?) ON CONFLICT (user_id) DO UPDATE SET cookie = excluded.cookie, updated_unix = excluded.updated_unix", userID, cookie, t)
	return err
}

func (s *Sqlite) GetDecryptedUserCookie(userID string) (string, error) {
	var cookie string
	err := s.db.QueryRow("SELECT cookie FROM user_cookies WHERE user_id=?", userID).Scan(&cookie)
	if err != nil {
		return "", nil
	}
	return cookie, nil
}

func (s *Sqlite) AddCode(code string, userID *string, source *string) error {
	t := time.Now().Unix()
	_, err := s.db.Exec("INSERT INTO shift_codes (code, user_id, source, added_unix) VALUES (?, ?, ?, ?)", code, userID, source, t)
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
