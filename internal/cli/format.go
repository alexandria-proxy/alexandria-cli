package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
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
	switch s {
	case "":
		return 0, false
	case "unlimited", "off", "none":
		return 0, true
	}
	digits, unit := s, ""
	for i, r := range s {
		if r >= '0' && r <= '9' || r == '.' {
			continue
		}
		digits, unit = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
		break
	}
	n, err := strconv.ParseFloat(digits, 64)
	if digits == "" || err != nil || n < 0 {
		return 0, false
	}
	if unit == "" {
		if n == 0 {
			return 0, true
		}
		unit = "mb"
	}
	unit = strings.TrimSuffix(strings.Replace(unit, "ib", "b", 1), "b")
	for _, u := range sizeunits {
		if unit == strings.TrimSuffix(u.name, "b") {
			return int64(n * float64(u.scale)), true
		}
	}
	return 0, false
}

func formatsize(n int64) string {
	if n <= 0 {
		return "unlimited"
	}
	for _, u := range sizeunits {
		if n >= u.scale && n%u.scale == 0 {
			return strconv.FormatInt(n/u.scale, 10) + u.name
		}
	}
	return strconv.FormatInt(n, 10) + "b"
}

func trimdec(f float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0")
}

func humanbytes(n int64) string {
	switch {
	case n >= 1<<40:
		return trimdec(float64(n)/(1<<40)) + "tb"
	case n >= 1<<30:
		return trimdec(float64(n)/(1<<30)) + "gb"
	case n >= 1<<20:
		return trimdec(float64(n)/(1<<20)) + "mb"
	}
	return strconv.FormatInt(n/(1<<10), 10) + "kb"
}

func usage(s sub.Subscription) string {
	if s.Usedbytes <= 0 && s.Totalbytes <= 0 {
		return ""
	}
	total := "∞"
	if s.Totalbytes > 0 {
		total = humanbytes(s.Totalbytes)
	}
	return humanbytes(s.Usedbytes) + " (of " + total + ")"
}

func day(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02")
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

func hours(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	h := int(d.Hours())
	if h >= 24 && h%24 == 0 {
		return strconv.Itoa(h/24) + "d"
	}
	return strconv.Itoa(h) + "h"
}

func uptime(since int64) string {
	if since <= 0 {
		return ""
	}
	d := time.Since(time.Unix(since, 0))
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s/60%60, s%60)
}

func isjson(raw string) bool {
	s := strings.TrimSpace(raw)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func protolabel(s sub.Server) string {
	p := strings.ToLower(strings.TrimSpace(s.Protocol))
	p = strings.Join(strings.Fields(strings.ReplaceAll(p, " / ", "/")), "")
	if isjson(s.Raw) && !strings.HasSuffix(p, "json") {
		p += ";json"
	}
	return p
}

func pingtext(ms int) string {
	switch {
	case ms < 0:
		return "dead"
	case ms == 0:
		return "—"
	}
	return strconv.Itoa(ms) + "ms"
}

func pingcolor(ms int) string {
	switch {
	case ms < 0:
		return red
	case ms == 0:
		return dim
	case ms <= 100:
		return green
	case ms <= 200:
		return yellow
	}
	return red
}

func pingrank(ms int) int {
	if ms <= 0 {
		return 1 << 30
	}
	return ms
}
