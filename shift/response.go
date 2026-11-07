package shift

type ResponseType int

const (
	Success ResponseType = iota
	AlreadyRedeemed
	Invalid
	Expired
	Link2KAccount
	WrongGame
	Unrecognized
)

const (
	ALREADY_REDEEMED = "This SHiFT code has already been redeemed"
	NOT_EXIST        = "This SHiFT code does not exist"
	EXPIRED          = "This SHiFT code has expired"
	SUCCESS          = "Your code was successfully redeemed"
	LINK2K           = "To redeem this SHiFT code, please link your 2K account."
	// WRONG_GAME isn't from SHiFT. It's the status recorded when SHiFT says the code is for a different game, so the
	// code isn't redeemed (or tried again for anyone)
	WRONG_GAME = "This SHiFT code isn't for Borderlands 4"
)

func DetermineResponseType(input string) ResponseType {
	switch input {
	case SUCCESS:
		return Success
	case ALREADY_REDEEMED:
		return AlreadyRedeemed
	case EXPIRED:
		return Expired
	case NOT_EXIST:
		return Invalid
	case LINK2K:
		return Link2KAccount
	case WRONG_GAME:
		return WrongGame
	default:
		return Unrecognized
	}
}
