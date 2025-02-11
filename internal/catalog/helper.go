package catalog

import "strings"

func ToLabel(name string) string {
	// apiKey -> Api Key
	var result string
	for i, char := range name {
		if i > 0 && char >= 'A' && char <= 'Z' {
			result += " "
		}
		result += string(char)
		if len(result) == 1 {
			result = strings.ToUpper(result)
		}
	}
	return result
}
