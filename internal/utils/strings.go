package utils

func ShortStr(s string) string {
	if len(s) > 8 {
		return s[:4] + "..." + s[len(s)-4:]
	}
	return s
}
