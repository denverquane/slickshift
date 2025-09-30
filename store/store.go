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

const DiscordSource = "discord"

type Store interface {
	UserExists(userID string) bool
	AddUser(userID string) error
	SetUserPlatform(userID, platform string) error
	GetUserPlatform(userID string) (string, error)
	SetUserDM(userID string, dm bool) error
	GetUserDM(userID string) (bool, error)

	EncryptAndSetUserCookies(userID string, cookie []*http.Cookie) error
	GetDecryptedUserCookies(userID string) ([]*http.Cookie, error)
	GetAllDecryptedUserCookies() ([]UserCookies, error)

	CodeExists(code string) bool
	AddCode(code, game string, userID *string, source *string) error
	SetCodeRewardIfNotSet(code, reward string) (bool, error)
	GetCodesNotRedeemedForUser(userID, platform string) ([]string, error)

	GetRecentRedemptionsForUser(userID, status string, quantity int) ([]Redemption, error)
	AddRedemption(userID, code, platform string, status string) error
}
