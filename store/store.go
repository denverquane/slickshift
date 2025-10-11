package store

import (
	"database/sql"
	"net/http"
)

type UserCookies struct {
	UserID  string
	Cookies []*http.Cookie
}

type Redemption struct {
	Code     string
	Platform string
	Status   string
	TimeUnix int64
	Game     string
	Reward   sql.NullString
}

type Statistics struct {
	Users       map[string]int64 `json:"users"`
	Codes       map[string]int64 `json:"codes"`
	Redemptions map[string]int64 `json:"redemptions"`
}

const DiscordSource = "discord"

type Store interface {
	UserExists(userID string) bool
	AddUser(userID string) error
	GetUserPlatformAndDM(userID string) (string, bool, error)
	SetUserPlatform(userID, platform string) error
	SetUserDM(userID string, dm bool) error

	UserCookiesExists(userID string) bool
	EncryptAndSetUserCookies(userID string, cookie []*http.Cookie) error
	GetDecryptedUserCookies(userID string) ([]*http.Cookie, error)
	DeleteUserCookies(userID string) error
	GetAllDecryptedUserCookiesSorted(limit int64) ([]UserCookies, error)

	CodeExists(code string) bool
	AddCode(code, game string, userID *string, source *string) error
	SetCodeRewardAndSuccess(code, reward string, success bool) (bool, error)
	GetValidCodesNotRedeemedForUser(userID, platform string) ([]string, error)

	GetRecentRedemptionsForUser(userID, status string, quantity int) ([]Redemption, error)
	AddRedemption(userID, code, platform string, status string) error

	GetStatistics(userID string) (Statistics, error)

	Close() error
}
