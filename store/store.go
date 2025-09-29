package store

import "net/http"

type UserCookies struct {
	UserID  string
	Cookies []*http.Cookie
}

type Redemption struct {
	Code     string
	Platform string
	Status   string
	TimeUnix int64
}

type Store interface {
	UserExists(userID string) bool
	AddUser(userID, platform string, dm bool) error
	SetUserPlatform(userID, platform string) error
	GetUserPlatform(userID string) (string, error)
	SetUserDM(userID string, dm bool) error
	GetUserDM(userID string) (bool, error)

	EncryptAndSetUserCookies(userID string, cookie []*http.Cookie) error
	GetDecryptedUserCookies(userID string) ([]*http.Cookie, error)

	AddCode(code, game string, userID *string, source *string) error
	SetCodeRewardIfNotSet(code, reward string) (bool, error)
	GetAllUserCookies() ([]UserCookies, error)
	GetCodesNotRedeemedForUser(userID, platform string) ([]string, error)

	RecentSuccessfulRedemptions(quantity int) ([]Redemption, error)
	RecentRedemptionsForUser(userID string, quantity int) ([]Redemption, error)
	AddRedemption(userID, code, platform string, status string) error
}
