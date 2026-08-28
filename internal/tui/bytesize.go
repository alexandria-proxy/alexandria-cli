package tui

import (
	"strconv"
	"strings"
)

var sizeunits = []struct {
	name  string
	scale int64
}{
	{"tb", 1 << 40},
	{"gb", 1 << 30},
	{"mb", 1 << 20},
	{"kb", 1 << 10},
	{"b", 1},
}

func parsesize(raw string) (int64, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return 0, false
	}
	digits := s
	unit := ""
	for i, r := range s {
		if r >= '0' && r <= '9' || r == '.' {
			continue
		}
		digits, unit = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
		break
	}
	if digits == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(digits, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	if unit == "" {
		if n == 0 {
			return 0, true
		}
		unit = "mb"
	}
	for _, u := range sizeunits {
		if unit == u.name || unit == strings.TrimSuffix(u.name, "b") && u.name != "b" {
			return int64(n * float64(u.scale)), true
		}
	}
	return 0, false
}

func formatsize(n int64) string {
	if n <= 0 {
		return "0"
	}
	for _, u := range sizeunits {
		if n >= u.scale && n%u.scale == 0 {
			return strconv.FormatInt(n/u.scale, 10) + " " + u.name
		}
	}
	return strconv.FormatInt(n, 10) + " b"
}
