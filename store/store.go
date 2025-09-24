package store

type Store interface {
	UserExists(userID string) bool
	AddUser(userID string, platform string) error
	SetUserPlatform(userID string, platform string) error

	SetUserCookie(userID string, cookie string) error
	GetUserCookie(userID string) (string, error)

	AddCode(code string, userID *string, source *string) error
}
