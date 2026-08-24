package rellog

import "strings"

func isCanonicalScriptPath(value string) bool {
	if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "/") || strings.Contains(value, `\`) {
		return false
	}
	if len(value) >= 3 && isASCIIAlpha(value[0]) && value[1] == ':' && value[2] == '/' {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func isASCIIAlpha(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
