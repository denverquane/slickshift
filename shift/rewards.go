package shift

type Reward struct {
	Title       string
	Date        string // UTC time? - 8:00+PST redemption was after midnight because it displayed the next day
	Description string
}
