package smithery

import (
	"strings"
)

func Title(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
