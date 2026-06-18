package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxInputRows = 3

type textInput struct {
	value     string
	cursorPos int
	scroll    int
}

func (t *textInput) focusEnd() { t.cursorPos = t.length() }

func (t textInput) length() int { return len([]rune(t.value)) }

func (t *textInput) handleKey(msg tea.KeyMsg, cw int) bool {
	handled := true
	switch msg.String() {
	case "left":
		if t.cursorPos > 0 {
			t.cursorPos--
		}
	case "right":
		if t.cursorPos < t.length() {
			t.cursorPos++
		}
	case "up":
		handled = t.moveUp(cw)
	case "down":
		handled = t.moveDown(cw)
	case "ctrl+left":
		t.moveWordLeft()
	case "ctrl+right":
		t.moveWordRight()
	case "ctrl+a":
		t.cursorPos = 0
	case "ctrl+e":
		t.cursorPos = t.length()
	case "ctrl+k":
		r := []rune(t.value)
		cp := clampInt(t.cursorPos, 0, len(r))
		t.value = string(r[:cp])
	case "ctrl+backspace":
		t.deleteWord()
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
		t.clampScroll(cw)
	}
	return handled
}

func (t *textInput) insert(s string) {
	r := []rune(t.value)
	cp := clampInt(t.cursorPos, 0, len(r))
	ins := []rune(s)
	out := make([]rune, 0, len(r)+len(ins))
	out = append(out, r[:cp]...)
	out = append(out, ins...)
	out = append(out, r[cp:]...)
	t.value = string(out)
	t.cursorPos = cp + len(ins)
}

func (t *textInput) backspace() {
	r := []rune(t.value)
	cp := clampInt(t.cursorPos, 0, len(r))
	if cp == 0 {
		return
	}
	t.value = string(append(r[:cp-1], r[cp:]...))
	t.cursorPos = cp - 1
}

func (t *textInput) deleteWord() {
	r := []rune(t.value)
	cp := clampInt(t.cursorPos, 0, len(r))
	i := cp
	for i > 0 && isWordSep(r[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(r[i-1]) {
		i--
	}
	t.value = string(append(r[:i], r[cp:]...))
	t.cursorPos = i
}

func (t *textInput) moveUp(cw int) bool {
	if cw < 1 {
		cw = 1
	}
	if t.cursorPos < cw {
		return false
	}
	t.cursorPos -= cw
	return true
}

func (t *textInput) moveDown(cw int) bool {
	if cw < 1 {
		cw = 1
	}
	n := t.length()
	if t.cursorPos/cw >= n/cw {
		return false
	}
	t.cursorPos += cw
	if t.cursorPos > n {
		t.cursorPos = n
	}
	return true
}

func (t *textInput) moveWordLeft() {
	r := []rune(t.value)
	i := clampInt(t.cursorPos, 0, len(r))
	for i > 0 && isWordSep(r[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(r[i-1]) {
		i--
	}
	t.cursorPos = i
}

func (t *textInput) moveWordRight() {
	r := []rune(t.value)
	n := len(r)
	i := clampInt(t.cursorPos, 0, n)
	for i < n && isWordSep(r[i]) {
		i++
	}
	for i < n && !isWordSep(r[i]) {
		i++
	}
	t.cursorPos = i
}

func (t *textInput) clampScroll(cw int) {
	if cw < 1 {
		cw = 1
	}
	n := t.length()
	t.cursorPos = clampInt(t.cursorPos, 0, n)
	cursorRow := t.cursorPos / cw
	if cursorRow < t.scroll {
		t.scroll = cursorRow
	}
	if cursorRow > t.scroll+maxInputRows-1 {
		t.scroll = cursorRow - maxInputRows + 1
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t textInput) view(cw int, focused bool, fg lipgloss.Color) string {
	if cw < 1 {
		cw = 1
	}
	runes := []rune(t.value)
	n := len(runes)
	cp := clampInt(t.cursorPos, 0, n)

	cells := make([]rune, n)
	copy(cells, runes)
	if focused && cp == n {
		cells = append(cells, ' ')
	}
	rows := wrapCells(cells, cw)
	totalRows := len(rows)

	cursorRow, cursorCol := cp/cw, cp%cw

	scroll := 0
	if focused {
		scroll = t.scroll
		if cursorRow < scroll {
			scroll = cursorRow
		}
		if cursorRow > scroll+maxInputRows-1 {
			scroll = cursorRow - maxInputRows + 1
		}
	}
	if scroll > max0(totalRows-maxInputRows) {
		scroll = max0(totalRows - maxInputRows)
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + maxInputRows
	if end > totalRows {
		end = totalRows
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
			case r == end-1 && end < totalRows && c == len(row)-1:
				b.WriteString(dim.Render("…"))
			default:
				cell := string(row[c])
				if focused && r == cursorRow && c == cursorCol && blink {
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

func wrapCells(cells []rune, cw int) [][]rune {
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

func isWordSep(r rune) bool { return r == ' ' || r == '/' }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clipRunes(s string, w int) string {
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
