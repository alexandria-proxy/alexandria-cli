package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type jsoneditor struct {
	lines  []string
	row    int
	col    int
	scroll int
	hoff   int
	selrow int
	selcol int
}

func newjsoneditor(s string) jsoneditor {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return jsoneditor{lines: lines, selrow: -1}
}

func (e jsoneditor) value() string { return strings.Join(e.lines, "\n") }

func (e *jsoneditor) line() []rune { return []rune(e.lines[e.row]) }

func (e *jsoneditor) setline(r []rune) { e.lines[e.row] = string(r) }

func (e *jsoneditor) handlekey(msg tea.KeyMsg, w, h int) {
	switch msg.String() {
	case "left":
		e.clearsel()
		e.moveleft()
	case "right":
		e.clearsel()
		e.moveright()
	case "up":
		e.clearsel()
		e.moveup()
	case "down":
		e.clearsel()
		e.movedown()
	case "shift+left":
		e.startsel()
		e.moveleft()
	case "shift+right":
		e.startsel()
		e.moveright()
	case "shift+up":
		e.startsel()
		e.moveup()
	case "shift+down":
		e.startsel()
		e.movedown()
	case "home", "ctrl+a":
		e.clearsel()
		e.col = 0
	case "end", "ctrl+e":
		e.clearsel()
		e.col = len(e.line())
	case "shift+home":
		e.startsel()
		e.col = 0
	case "shift+end":
		e.startsel()
		e.col = len(e.line())
	case "ctrl+left", "alt+b":
		e.clearsel()
		e.wordleft()
	case "ctrl+right", "alt+f":
		e.clearsel()
		e.wordright()
	case "enter":
		e.clearsel()
		e.newline()
	case "backspace":
		e.clearsel()
		e.backspace()
	case "ctrl+w", "ctrl+h", "ctrl+backspace", "alt+backspace":
		e.clearsel()
		e.deleteword()
	case "alt+d":
		e.clearsel()
		e.deletewordforward()
	case "tab":
		e.clearsel()
		e.insert("  ")
	case " ":
		e.clearsel()
		e.insert(" ")
	default:
		if msg.Type == tea.KeyRunes {
			e.clearsel()
			e.insert(string(msg.Runes))
		}
	}
	e.clamp(w, h)
}

func (e *jsoneditor) moveleft() {
	if e.col > 0 {
		e.col--
	} else if e.row > 0 {
		e.row--
		e.col = len(e.line())
	}
}

func (e *jsoneditor) moveright() {
	if e.col < len(e.line()) {
		e.col++
	} else if e.row < len(e.lines)-1 {
		e.row++
		e.col = 0
	}
}

func (e *jsoneditor) moveup() {
	if e.row > 0 {
		e.row--
		e.col = clampint(e.col, 0, len(e.line()))
	}
}

func (e *jsoneditor) movedown() {
	if e.row < len(e.lines)-1 {
		e.row++
		e.col = clampint(e.col, 0, len(e.line()))
	}
}

func (e *jsoneditor) startsel() {
	if e.selrow < 0 {
		e.selrow, e.selcol = e.row, e.col
	}
}

func (e *jsoneditor) clearsel() { e.selrow = -1 }

func (e *jsoneditor) hassel() bool {
	return e.selrow >= 0 && (e.selrow != e.row || e.selcol != e.col)
}

func (e *jsoneditor) selrange() (int, int, int, int) {
	ar, ac, br, bc := e.selrow, e.selcol, e.row, e.col
	if ar > br || (ar == br && ac > bc) {
		ar, ac, br, bc = br, bc, ar, ac
	}
	return ar, ac, br, bc
}

func (e *jsoneditor) selectedtext() string {
	if !e.hassel() {
		return ""
	}
	ar, ac, br, bc := e.selrange()
	if ar == br {
		r := []rune(e.lines[ar])
		return string(r[clampint(ac, 0, len(r)):clampint(bc, 0, len(r))])
	}
	first := []rune(e.lines[ar])
	parts := []string{string(first[clampint(ac, 0, len(first)):])}
	for i := ar + 1; i < br; i++ {
		parts = append(parts, e.lines[i])
	}
	last := []rune(e.lines[br])
	parts = append(parts, string(last[:clampint(bc, 0, len(last))]))
	return strings.Join(parts, "\n")
}

func (e *jsoneditor) copytext() string {
	if e.hassel() {
		return e.selectedtext()
	}
	return e.value()
}

func (e *jsoneditor) scrollby(d, h int) {
	e.scroll = clampint(e.scroll+d, 0, max0(len(e.lines)-1))
	e.row = clampint(e.row, e.scroll, e.scroll+max0(h-1))
	e.row = clampint(e.row, 0, len(e.lines)-1)
	e.col = clampint(e.col, 0, len(e.line()))
}

func (e *jsoneditor) dragextend(dir, w, h int) {
	if dir < 0 {
		e.moveup()
	} else if dir > 0 {
		e.movedown()
	}
	e.clamp(w, h)
}

func (e *jsoneditor) insert(s string) {
	r := e.line()
	cp := clampint(e.col, 0, len(r))
	ins := []rune(s)
	out := make([]rune, 0, len(r)+len(ins))
	out = append(out, r[:cp]...)
	out = append(out, ins...)
	out = append(out, r[cp:]...)
	e.setline(out)
	e.col = cp + len(ins)
}

func (e *jsoneditor) newline() {
	r := e.line()
	cp := clampint(e.col, 0, len(r))
	head, tail := string(r[:cp]), string(r[cp:])
	e.lines[e.row] = head
	rest := append([]string{tail}, e.lines[e.row+1:]...)
	e.lines = append(e.lines[:e.row+1], rest...)
	e.row++
	e.col = 0
}

func (e *jsoneditor) backspace() {
	r := e.line()
	cp := clampint(e.col, 0, len(r))
	if cp > 0 {
		e.setline(append(r[:cp-1], r[cp:]...))
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

func (e *jsoneditor) deleteword() {
	r := e.line()
	cp := clampint(e.col, 0, len(r))
	if cp == 0 {
		e.backspace()
		return
	}
	i := cp
	for i > 0 && iswordsep(r[i-1]) {
		i--
	}
	for i > 0 && !iswordsep(r[i-1]) {
		i--
	}
	e.setline(append(r[:i], r[cp:]...))
	e.col = i
}

func (e *jsoneditor) deletewordforward() {
	r := e.line()
	cp := clampint(e.col, 0, len(r))
	if cp >= len(r) {
		return
	}
	i := cp
	for i < len(r) && iswordsep(r[i]) {
		i++
	}
	for i < len(r) && !iswordsep(r[i]) {
		i++
	}
	e.setline(append(r[:cp], r[i:]...))
}

func (e *jsoneditor) wordleft() {
	r := e.line()
	i := clampint(e.col, 0, len(r))
	for i > 0 && iswordsep(r[i-1]) {
		i--
	}
	for i > 0 && !iswordsep(r[i-1]) {
		i--
	}
	e.col = i
}

func (e *jsoneditor) wordright() {
	r := e.line()
	n := len(r)
	i := clampint(e.col, 0, n)
	for i < n && iswordsep(r[i]) {
		i++
	}
	for i < n && !iswordsep(r[i]) {
		i++
	}
	e.col = i
}

func (e *jsoneditor) clamp(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	e.row = clampint(e.row, 0, len(e.lines)-1)
	e.col = clampint(e.col, 0, len(e.line()))
	if e.selrow >= len(e.lines) {
		e.selrow = -1
	}
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
	jsontext   = lipgloss.Color("#ABB2BF")
	jsonkey    = lipgloss.Color("#61AFEF")
	jsonstring = lipgloss.Color("#98C379")
	jsonnumber = lipgloss.Color("#D19A66")
	jsonbool   = lipgloss.Color("#C678DD")
	jsonpunct  = lipgloss.Color("#7E868D")
	selstyle   = lipgloss.NewStyle().Background(lipgloss.Color("#6B7286"))
)

func highlightjson(r []rune) []lipgloss.Color {
	cols := make([]lipgloss.Color, len(r))
	for i := range cols {
		cols[i] = jsontext
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
			col := jsonstring
			if k < len(r) && r[k] == ':' {
				col = jsonkey
			}
			for x := i; x < j && x < len(r); x++ {
				cols[x] = col
			}
			i = j
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':':
			cols[i] = jsonpunct
			i++
		case (c >= '0' && c <= '9') || c == '-':
			j := i
			for j < len(r) && ((r[j] >= '0' && r[j] <= '9') || r[j] == '.' || r[j] == '-' || r[j] == '+' || r[j] == 'e' || r[j] == 'E') {
				j++
			}
			for x := i; x < j; x++ {
				cols[x] = jsonnumber
			}
			i = j
		case c == 't' || c == 'f' || c == 'n':
			j := i
			for j < len(r) && r[j] >= 'a' && r[j] <= 'z' {
				j++
			}
			if w := string(r[i:j]); w == "true" || w == "false" || w == "null" {
				for x := i; x < j; x++ {
					cols[x] = jsonbool
				}
			}
			i = j
		default:
			i++
		}
	}
	return cols
}

func (e jsoneditor) view(w, h int, focused bool) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	selon := focused && e.hassel()
	var ar, ac, br, bc int
	if selon {
		ar, ac, br, bc = e.selrange()
	}

	var out []string
	for vr := 0; vr < h; vr++ {
		lr := e.scroll + vr
		if lr >= len(e.lines) {
			out = append(out, "")
			continue
		}
		r := []rune(e.lines[lr])
		cols := highlightjson(r)
		var b strings.Builder
		for c := e.hoff; c < e.hoff+w; c++ {
			sel := selon && (lr > ar || (lr == ar && c >= ac)) && (lr < br || (lr == br && c < bc))
			switch {
			case e.hoff > 0 && c == e.hoff && c < len(r):
				b.WriteString(dim.Render("…"))
			case focused && lr == e.row && c == e.col:
				cell := " "
				if c < len(r) {
					cell = string(r[c])
				}
				b.WriteString(lipgloss.NewStyle().Reverse(true).Render(cell))
			case sel:
				if c < len(r) {
					b.WriteString(selstyle.Foreground(cols[c]).Render(string(r[c])))
				} else {
					b.WriteString(selstyle.Render(" "))
				}
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
