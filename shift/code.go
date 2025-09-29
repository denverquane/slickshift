package shift

import "regexp"

var CodeRegex = regexp.MustCompile("^(?:[A-Z0-9]{5}-){4}[A-Z0-9]{5}$")
