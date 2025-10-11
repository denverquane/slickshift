package shift

const GoldenKey = "Golden Key for Borderlands 4"

type Reward struct {
	Title       string
	Date        string // UTC time? - 8:00+PST redemption was after midnight because it displayed the next day
	Description string
}
