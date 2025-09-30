package shift

import "regexp"

var CodeLength = 29

var CodeRegex = regexp.MustCompile("^(?:[A-Z0-9]{5}-){4}[A-Z0-9]{5}$")
