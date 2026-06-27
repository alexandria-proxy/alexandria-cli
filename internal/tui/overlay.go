package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func placeoverlay(x, y int, fg, bg string) string {
	fglines := strings.Split(fg, "\n")
	bglines := strings.Split(bg, "\n")
	fgh := len(fglines)

	fgw := 0
	for _, l := range fglines {
		if w := ansi.StringWidth(l); w > fgw {
			fgw = w
		}
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	for len(bglines) < y+fgh {
		bglines = append(bglines, "")
	}

	var b strings.Builder
	for i, bgline := range bglines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < y || i >= y+fgh {
			b.WriteString(bgline)
			continue
		}
		fgline := fglines[i-y]
		if w := ansi.StringWidth(fgline); w < fgw {
			fgline += strings.Repeat(" ", fgw-w)
		}
		left := ansi.Truncate(bgline, x, "")
		if lw := ansi.StringWidth(left); lw < x {
			left += strings.Repeat(" ", x-lw)
		}
		right := ansi.TruncateLeft(bgline, x+fgw, "")
		b.WriteString(left)
		b.WriteString("\x1b[0m")
		b.WriteString(fgline)
		b.WriteString("\x1b[0m")
		b.WriteString(right)
	}
	return b.String()
}
