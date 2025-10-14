package shift

type ResponseType int

const (
	Success ResponseType = iota
	AlreadyRedeemed
	Invalid
	Expired
	Link2KAccount
	Unrecognized
)

const (
	ALREADY_REDEEMED = "This SHiFT code has already been redeemed"
	NOT_EXIST        = "This SHiFT code does not exist"
	EXPIRED          = "This SHiFT code has expired"
	SUCCESS          = "Your code was successfully redeemed"
	LINK2K           = "To redeem this SHiFT code, please link your 2K account."
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
	default:
		return Unrecognized
	}
}
