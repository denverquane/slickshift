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
	Code     string         `json:"code"`
	Platform string         `json:"platform"`
	Status   string         `json:"status"`
	TimeUnix int64          `json:"time_unix"`
	Game     string         `json:"game"`
	Reward   sql.NullString `json:"reward"`
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
	GetValidCodesNotRedeemedForUser(userID, platform string, limit int) ([]string, error)

	GetRecentRedemptionsForUser(userID, status string, quantity int) ([]Redemption, error)
	RedemptionSummaryForUser(userID string) (map[string]int64, error)
	AddRedemption(userID, code, platform string, status string) error

	AddShiftError(userID, code, platform, error string) error
	GetShiftErrors(userID string) ([]string, error)
	ClearShiftErrors(userID string) error

	GetStatistics(userID string) (Statistics, error)

	Close() error
}
