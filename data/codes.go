package data

import (
	_ "embed"
	"strings"

	"github.com/denverquane/slickshift/shift"
)

// NOTE many of these codes are expired. The bot DB is oriented around community-generated
// feedback on codes being expired or not, and thus, maintaining a strict list of only valid codes
// is not a priority

//go:embed bl4_codes.txt
var bl4CodesText string

// return a map to guarantee no duplicates
func DefaultBL4Codes() map[string]struct{} {
	codes := map[string]struct{}{}
	for _, code := range strings.Split(bl4CodesText, "\n") {
		code = strings.TrimSpace(code)
		if shift.CodeRegex.MatchString(code) {
			codes[code] = struct{}{}
		}
	}
	return codes
}
