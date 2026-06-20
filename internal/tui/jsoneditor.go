package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type jsonEditor struct {
	lines  []string
	row    int
	col    int
	scroll int
	hoff   int
}

func newJSONEditor(s string) jsonEditor {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return jsonEditor{lines: lines}
}

func (e jsonEditor) value() string { return strings.Join(e.lines, "\n") }

func (e *jsonEditor) line() []rune { return []rune(e.lines[e.row]) }

func (e *jsonEditor) setLine(r []rune) { e.lines[e.row] = string(r) }

func (e *jsonEditor) handleKey(msg tea.KeyMsg, w, h int) {
	switch msg.String() {
	case "left":
		if e.col > 0 {
			e.col--
		} else if e.row > 0 {
			e.row--
			e.col = len(e.line())
		}
	case "right":
		if e.col < len(e.line()) {
			e.col++
		} else if e.row < len(e.lines)-1 {
			e.row++
			e.col = 0
		}
	case "up":
		if e.row > 0 {
			e.row--
			e.col = clampInt(e.col, 0, len(e.line()))
		}
	case "down":
		if e.row < len(e.lines)-1 {
			e.row++
			e.col = clampInt(e.col, 0, len(e.line()))
		}
	case "home", "ctrl+a":
		e.col = 0
	case "end", "ctrl+e":
		e.col = len(e.line())
	case "ctrl+left", "alt+b":
		e.wordLeft()
	case "ctrl+right", "alt+f":
		e.wordRight()
	case "enter":
		e.newline()
	case "backspace":
		e.backspace()
	case "ctrl+w", "ctrl+h", "ctrl+backspace", "alt+backspace":
		e.deleteWord()
	case "alt+d":
		e.deleteWordForward()
	case "tab":
		e.insert("  ")
	case " ":
		e.insert(" ")
	default:
		if msg.Type == tea.KeyRunes {
			e.insert(string(msg.Runes))
		}
	}
	e.clamp(w, h)
}

func (e *jsonEditor) insert(s string) {
	r := e.line()
	cp := clampInt(e.col, 0, len(r))
	ins := []rune(s)
	out := make([]rune, 0, len(r)+len(ins))
	out = append(out, r[:cp]...)
	out = append(out, ins...)
	out = append(out, r[cp:]...)
	e.setLine(out)
	e.col = cp + len(ins)
}

func (e *jsonEditor) newline() {
	r := e.line()
	cp := clampInt(e.col, 0, len(r))
	head, tail := string(r[:cp]), string(r[cp:])
	e.lines[e.row] = head
	rest := append([]string{tail}, e.lines[e.row+1:]...)
	e.lines = append(e.lines[:e.row+1], rest...)
	e.row++
	e.col = 0
}

func (e *jsonEditor) backspace() {
	r := e.line()
	cp := clampInt(e.col, 0, len(r))
	if cp > 0 {
		e.setLine(append(r[:cp-1], r[cp:]...))
		e.col = cp - 1
		return
	}
	if e.row == 0 {
		return
	}
	prev := []rune(e.lines[e.row-1])
	e.col = len(prev)
	e.lines[e.row-1] = string(prev) + string(r)
	e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	e.row--
}

func (e *jsonEditor) deleteWord() {
	r := e.line()
	cp := clampInt(e.col, 0, len(r))
	if cp == 0 {
		e.backspace()
		return
	}
	i := cp
	for i > 0 && isWordSep(r[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(r[i-1]) {
		i--
	}
	e.setLine(append(r[:i], r[cp:]...))
	e.col = i
}

func (e *jsonEditor) deleteWordForward() {
	r := e.line()
	cp := clampInt(e.col, 0, len(r))
	if cp >= len(r) {
		return
	}
	i := cp
	for i < len(r) && isWordSep(r[i]) {
		i++
	}
	for i < len(r) && !isWordSep(r[i]) {
		i++
	}
	e.setLine(append(r[:cp], r[i:]...))
}

func (e *jsonEditor) wordLeft() {
	r := e.line()
	i := clampInt(e.col, 0, len(r))
	for i > 0 && isWordSep(r[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(r[i-1]) {
		i--
	}
	e.col = i
}

func (e *jsonEditor) wordRight() {
	r := e.line()
	n := len(r)
	i := clampInt(e.col, 0, n)
	for i < n && isWordSep(r[i]) {
		i++
	}
	for i < n && !isWordSep(r[i]) {
		i++
	}
	e.col = i
}

func (e *jsonEditor) clamp(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	e.row = clampInt(e.row, 0, len(e.lines)-1)
	e.col = clampInt(e.col, 0, len(e.line()))
	if e.row < e.scroll {
		e.scroll = e.row
	}
	if e.row > e.scroll+h-1 {
		e.scroll = e.row - h + 1
	}
	if e.col < e.hoff {
		e.hoff = e.col
	}
	if e.col > e.hoff+w-1 {
		e.hoff = e.col - w + 1
	}
	if e.scroll < 0 {
		e.scroll = 0
	}
	if e.hoff < 0 {
		e.hoff = 0
	}
}

var (
	jsonText   = lipgloss.Color("#ABB2BF")
	jsonKey    = lipgloss.Color("#61AFEF")
	jsonString = lipgloss.Color("#98C379")
	jsonNumber = lipgloss.Color("#D19A66")
	jsonBool   = lipgloss.Color("#C678DD")
	jsonPunct  = lipgloss.Color("#7E868D")
)

func highlightJSON(r []rune) []lipgloss.Color {
	cols := make([]lipgloss.Color, len(r))
	for i := range cols {
		cols[i] = jsonText
	}
	i := 0
	for i < len(r) {
		switch c := r[i]; {
		case c == '"':
			j := i + 1
			for j < len(r) {
				if r[j] == '\\' && j+1 < len(r) {
					j += 2
					continue
				}
				if r[j] == '"' {
					j++
					break
				}
				j++
			}
			k := j
			for k < len(r) && (r[k] == ' ' || r[k] == '\t') {
				k++
			}
			col := jsonString
			if k < len(r) && r[k] == ':' {
				col = jsonKey
			}
			for x := i; x < j && x < len(r); x++ {
				cols[x] = col
			}
			i = j
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':':
			cols[i] = jsonPunct
			i++
		case (c >= '0' && c <= '9') || c == '-':
			j := i
			for j < len(r) && ((r[j] >= '0' && r[j] <= '9') || r[j] == '.' || r[j] == '-' || r[j] == '+' || r[j] == 'e' || r[j] == 'E') {
				j++
			}
			for x := i; x < j; x++ {
				cols[x] = jsonNumber
			}
			i = j
		case c == 't' || c == 'f' || c == 'n':
			j := i
			for j < len(r) && r[j] >= 'a' && r[j] <= 'z' {
				j++
			}
			if w := string(r[i:j]); w == "true" || w == "false" || w == "null" {
				for x := i; x < j; x++ {
					cols[x] = jsonBool
				}
			}
			i = j
		default:
			i++
		}
	}
	return cols
}

func (e jsonEditor) view(w, h int, focused bool) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	var out []string
	for vr := 0; vr < h; vr++ {
		lr := e.scroll + vr
		if lr >= len(e.lines) {
			out = append(out, "")
			continue
		}
		r := []rune(e.lines[lr])
		cols := highlightJSON(r)
		var b strings.Builder
		for c := e.hoff; c < e.hoff+w; c++ {
			switch {
			case e.hoff > 0 && c == e.hoff && c < len(r):
				b.WriteString(dim.Render("…"))
			case focused && lr == e.row && c == e.col:
				cell := " "
				if c < len(r) {
					cell = string(r[c])
				}
				b.WriteString(lipgloss.NewStyle().Reverse(true).Render(cell))
			case c < len(r):
				b.WriteString(lipgloss.NewStyle().Foreground(cols[c]).Render(string(r[c])))
			default:
				b.WriteString(" ")
			}
		}
		out = append(out, b.String())
	}
	return strings.Join(out, "\n")
}
