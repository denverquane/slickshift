package shift

type ResponseType int

const (
	Success ResponseType = iota
	AlreadyRedeemed
	Invalid
	Expired
	Unrecognized
)

const (
	ALREADY_REDEEMED = "This SHiFT code has already been redeemed"
	NOT_EXIST        = "This SHiFT code does not exist"
	SUCCESS          = "Your code was successfully redeemed"
)

func DetermineResponseType(input string) ResponseType {
	switch input {
	case ALREADY_REDEEMED:
		return AlreadyRedeemed
	case SUCCESS:
		return Success
	case NOT_EXIST:
		return Invalid
	default:
		return Unrecognized
	}
}
