package data

import (
	_ "embed"
	"strings"

	"github.com/denverquane/slickshift/shift"
)

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
