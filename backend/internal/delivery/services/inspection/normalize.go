package inspection

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func Normalize(value string) string {
	return strings.ToLower(norm.NFKC.String(value))
}
