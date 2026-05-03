package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func ParseCursor(cursor string) (int64, bool) {
	if cursor == "" {
		return 0, false
	}
	var ts int64
	_, err := parseCursorInt64(cursor, &ts)
	if err != nil {
		return 0, false
	}
	return ts, true
}

func parseCursorInt64(s string, v *int64) (bool, error) {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false, nil
		}
	}
	var val int64
	for i := 0; i < len(s); i++ {
		val = val*10 + int64(s[i]-'0')
	}
	*v = val
	return true, nil
}