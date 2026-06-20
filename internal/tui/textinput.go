package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxinputrows = 3

type textinput struct {
	value     string
	cursorpos int
	scroll    int
}

func (t *textinput) focusend() { t.cursorpos = t.length() }

func (t textinput) length() int { return len([]rune(t.value)) }

func (t *textinput) handlekey(msg tea.KeyMsg, cw int) bool {
	handled := true
	switch msg.String() {
	case "left":
		if t.cursorpos > 0 {
			t.cursorpos--
		}
	case "right":
		if t.cursorpos < t.length() {
			t.cursorpos++
		}
	case "up":
		handled = t.moveup(cw)
	case "down":
		handled = t.movedown(cw)
	case "ctrl+left":
		t.movewordleft()
	case "ctrl+right":
		t.movewordright()
	case "ctrl+a":
		t.cursorpos = 0
	case "ctrl+e":
		t.cursorpos = t.length()
	case "ctrl+k":
		r := []rune(t.value)
		cp := clampint(t.cursorpos, 0, len(r))
		t.value = string(r[:cp])
	case "ctrl+w", "ctrl+h", "ctrl+backspace":
		t.deleteword()
	case "backspace":
		t.backspace()
	case " ":
		t.insert(" ")
	default:
		if msg.Type == tea.KeyRunes {
			t.insert(string(msg.Runes))
		} else {
			handled = false
		}
	}
	if handled {
		t.clampscroll(cw)
	}
	return handled
}

func (t *textinput) insert(s string) {
	r := []rune(t.value)
	cp := clampint(t.cursorpos, 0, len(r))
	ins := []rune(s)
	out := make([]rune, 0, len(r)+len(ins))
	out = append(out, r[:cp]...)
	out = append(out, ins...)
	out = append(out, r[cp:]...)
	t.value = string(out)
	t.cursorpos = cp + len(ins)
}

func (t *textinput) backspace() {
	r := []rune(t.value)
	cp := clampint(t.cursorpos, 0, len(r))
	if cp == 0 {
		return
	}
	t.value = string(append(r[:cp-1], r[cp:]...))
	t.cursorpos = cp - 1
}

func (t *textinput) deleteword() {
	r := []rune(t.value)
	cp := clampint(t.cursorpos, 0, len(r))
	i := cp
	for i > 0 && iswordsep(r[i-1]) {
		i--
	}
	for i > 0 && !iswordsep(r[i-1]) {
		i--
	}
	t.value = string(append(r[:i], r[cp:]...))
	t.cursorpos = i
}

func (t *textinput) moveup(cw int) bool {
	if cw < 1 {
		cw = 1
	}
	if t.cursorpos < cw {
		return false
	}
	t.cursorpos -= cw
	return true
}

func (t *textinput) movedown(cw int) bool {
	if cw < 1 {
		cw = 1
	}
	n := t.length()
	if t.cursorpos/cw >= n/cw {
		return false
	}
	t.cursorpos += cw
	if t.cursorpos > n {
		t.cursorpos = n
	}
	return true
}

func (t *textinput) movewordleft() {
	r := []rune(t.value)
	i := clampint(t.cursorpos, 0, len(r))
	for i > 0 && iswordsep(r[i-1]) {
		i--
	}
	for i > 0 && !iswordsep(r[i-1]) {
		i--
	}
	t.cursorpos = i
}

func (t *textinput) movewordright() {
	r := []rune(t.value)
	n := len(r)
	i := clampint(t.cursorpos, 0, n)
	for i < n && iswordsep(r[i]) {
		i++
	}
	for i < n && !iswordsep(r[i]) {
		i++
	}
	t.cursorpos = i
}

func (t *textinput) clampscroll(cw int) {
	if cw < 1 {
		cw = 1
	}
	n := t.length()
	t.cursorpos = clampint(t.cursorpos, 0, n)
	cursorrow := t.cursorpos / cw
	if cursorrow < t.scroll {
		t.scroll = cursorrow
	}
	if cursorrow > t.scroll+maxinputrows-1 {
		t.scroll = cursorrow - maxinputrows + 1
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t textinput) view(cw int, focused bool, fg lipgloss.Color) string {
	if cw < 1 {
		cw = 1
	}
	runes := []rune(t.value)
	n := len(runes)
	cp := clampint(t.cursorpos, 0, n)

	cells := make([]rune, n)
	copy(cells, runes)
	if focused && cp == n {
		cells = append(cells, ' ')
	}
	rows := wrapcells(cells, cw)
	totalrows := len(rows)

	cursorrow, cursorcol := cp/cw, cp%cw

	scroll := 0
	if focused {
		scroll = t.scroll
		if cursorrow < scroll {
			scroll = cursorrow
		}
		if cursorrow > scroll+maxinputrows-1 {
			scroll = cursorrow - maxinputrows + 1
		}
	}
	if scroll > max0(totalrows-maxinputrows) {
		scroll = max0(totalrows - maxinputrows)
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + maxinputrows
	if end > totalrows {
		end = totalrows
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	txt := lipgloss.NewStyle().Foreground(fg)
	blink := (time.Now().UnixMilli()/500)%2 == 0

	var lines []string
	for r := scroll; r < end; r++ {
		row := rows[r]
		var b strings.Builder
		for c := 0; c < len(row); c++ {
			switch {
			case r == scroll && scroll > 0 && c == 0:
				b.WriteString(dim.Render("…"))
			case r == end-1 && end < totalrows && c == len(row)-1:
				b.WriteString(dim.Render("…"))
			default:
				cell := string(row[c])
				if focused && r == cursorrow && c == cursorcol && blink {
					b.WriteString(lipgloss.NewStyle().Reverse(true).Render(cell))
				} else {
					b.WriteString(txt.Render(cell))
				}
			}
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func wrapcells(cells []rune, cw int) [][]rune {
	if cw < 1 {
		cw = 1
	}
	var rows [][]rune
	for i := 0; i < len(cells); i += cw {
		end := i + cw
		if end > len(cells) {
			end = len(cells)
		}
		rows = append(rows, cells[i:end])
	}
	if len(rows) == 0 {
		rows = append(rows, nil)
	}
	return rows
}

func iswordsep(r rune) bool {
	switch r {
	case ' ', '/', '{', '}', '[', ']':
		return true
	}
	return false
}

func clampint(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func cliprunes(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
