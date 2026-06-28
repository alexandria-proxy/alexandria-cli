package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	toastlife = 4 * time.Second
	toastfade = 600 * time.Millisecond
	toastmax  = 3
)

type toastkind int

const (
	toasterr toastkind = iota
	toastok
	toastinfo
)

type toast struct {
	text string
	kind toastkind
	born time.Time
}

func toastcolor(k toastkind) string {
	switch k {
	case toastok:
		return "#A6E0AC"
	case toastinfo:
		return "#B9C2C9"
	default:
		return "#E0A6AC"
	}
}

func toasticon(k toastkind) string {
	switch k {
	case toastok:
		return "✓"
	case toastinfo:
		return "›"
	default:
		return "✕"
	}
}

func (t toast) opacity() float64 {
	e := time.Since(t.born)
	if e >= toastlife {
		return 0
	}
	if rem := toastlife - e; rem < toastfade {
		return float64(rem) / float64(toastfade)
	}
	return 1
}

func (t toast) render(maxw int) string {
	col := lipgloss.Color(dimhex(toastcolor(t.kind), t.opacity()))
	innerw := maxw - 4
	if innerw < 1 {
		innerw = 1
	}
	body := lipgloss.NewStyle().Foreground(col).Width(innerw).Render(toasticon(t.kind) + "  " + oneline(t.text))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(col).
		Padding(0, 1).
		Render(body)
}

func (m Menu) hastoasts() bool {
	for _, t := range m.toasts {
		if t.opacity() > 0 {
			return true
		}
	}
	return false
}

func (m Menu) rendertoasts(maxw int) string {
	var boxes []string
	for _, t := range m.toasts {
		if t.opacity() <= 0 {
			continue
		}
		boxes = append(boxes, t.render(maxw))
	}
	if len(boxes) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, boxes...)
}

func (m *Menu) pushtoast(kind toastkind, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	m.toasts = append(m.toasts, toast{text: text, kind: kind, born: time.Now()})
	if len(m.toasts) > toastmax {
		m.toasts = m.toasts[len(m.toasts)-toastmax:]
	}
}

func (m *Menu) prunetoasts() {
	var out []toast
	for _, t := range m.toasts {
		if time.Since(t.born) < toastlife {
			out = append(out, t)
		}
	}
	m.toasts = out
}

func (m Menu) withtoasts(view string) string {
	if m.width == 0 || !m.hastoasts() {
		return view
	}
	tw := 48
	if tw > m.width-4 {
		tw = m.width - 4
	}
	if tw < 10 {
		return view
	}
	stack := m.rendertoasts(tw)
	if stack == "" {
		return view
	}
	return placeoverlay(2, 1, stack, view)
}

func dimhex(hex string, t float64) string {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "#" + hex
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return "#" + hex
	}
	r := int(float64((n>>16)&0xff) * t)
	g := int(float64((n>>8)&0xff) * t)
	b := int(float64(n&0xff) * t)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}
