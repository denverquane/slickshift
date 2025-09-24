package store

type Platform string

const (
	STEAM       Platform = "steam"
	EPIC        Platform = "epic"
	XBOX        Platform = "xbox"
	PLAYSTATION Platform = "playstation"
)

type Store interface {
	UserExists(userID string) bool
	AddUser(userID string, platform Platform) error
	SetUserPlatform(userID string, platform Platform) error

	EncryptAndSetUserCookie(userID string, cookie string) error
	GetDecryptedUserCookie(userID string) (string, error)

	AddCode(code string, userID *string, source *string) error
}
