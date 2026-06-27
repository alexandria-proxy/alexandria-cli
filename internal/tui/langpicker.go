package tui

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type language struct {
	code  string
	flag  string
	label string
}

var languages = []language{
	{"en", "🇺🇸", "English"},
	{"ru", "🇷🇺", "Русский"},
	{"fa", "🇮🇷", "فارسی"},
}

var (
	btngray     = lipgloss.Color("#B9C2C9")
	titlestyle  = lipgloss.NewStyle().Bold(true).Foreground(btngray)
	optionstyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	selbase     = lipgloss.NewStyle().Bold(true).Background(btngray).Foreground(lipgloss.Color("16"))
	hintstyle   = lipgloss.NewStyle().Faint(true).PaddingRight(1)
)

var (
	ansiseq   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cursorseq = regexp.MustCompile(`\x1b\[\?25[lh]`)
)

const (
	shinetickrate = 45 * time.Millisecond
	shineperiod   = 80 // frames per full cycle
	shinesweep    = 30 // frames the band actually spends crossing the logo
	shineband     = 0.30
	shinepeak     = 0.92
	shinefloor    = 24 // colors dimmer than this stay put
)

type shinemsg time.Time

type rgb struct{ r, g, b int }

type cell struct {
	ch      rune
	fg, bg  *rgb
	reverse bool
}

type LangPicker struct {
	cells  [][]cell
	logow  int
	frame  int
	cursor int
	chosen string
	width  int
	height int
}

func NewLangPicker(logo string) LangPicker {
	cells, w := parselogo(logo)
	return LangPicker{cells: cells, logow: w}
}

func parselogo(s string) ([][]cell, int) {
	lines := strings.Split(trimblanklines(s), "\n")
	cells := make([][]cell, len(lines))
	w := 0
	for i, ln := range lines {
		cells[i] = parsecells(ln)
		if len(cells[i]) > w {
			w = len(cells[i])
		}
	}
	return cells, w
}

func (m LangPicker) Init() tea.Cmd { return tea.Batch(tea.HideCursor, shinetick()) }

func shinetick() tea.Cmd {
	return tea.Tick(shinetickrate, func(t time.Time) tea.Msg { return shinemsg(t) })
}

func (m LangPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shinemsg:
		m.frame = (m.frame + 1) % shineperiod
		return m, shinetick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, tea.HideCursor
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			m.cursor = (m.cursor - 1 + len(languages)) % len(languages)
		case "down", "j":
			m.cursor = (m.cursor + 1) % len(languages)
		case "enter", " ":
			m.chosen = languages[m.cursor].code
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m LangPicker) View() string {
	tr := i18n.T(languages[m.cursor].code)

	pad := selbase.Render(" ")
	rows := make([]string, len(languages))
	for i, lang := range languages {
		if i == m.cursor {
			marker := selbase.Render("●")
			text := selbase.Width(11).Render(lang.flag + lang.label)
			rows[i] = pad + marker + selbase.Render(" ") + text
		} else {
			rows[i] = optionstyle.Render("○ " + lang.flag + lang.label)
		}
	}

	title := titlestyle.Render(tr.ChooseLanguage)
	logo := m.renderlogo()
	logolines := strings.Split(logo, "\n")
	top := logo
	titlerow := title
	if len(logolines) > 1 {
		top = strings.Join(logolines[:len(logolines)-1], "\n")
		titlerow = stampcenter(logolines[len(logolines)-1], title, m.logow)
	}

	body := lipgloss.JoinVertical(
		lipgloss.Center,
		top,
		titlerow,
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
	hint := hintstyle.Render(tr.Hint)

	if m.width == 0 || m.height == 0 {
		return body + "\n" + hint
	}
	placed := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, body)
	bottom := lipgloss.PlaceHorizontal(m.width, lipgloss.Right, hint)
	return placed + "\n" + bottom
}

func (m LangPicker) Chosen() string { return m.chosen }

// renderlogo paints the wordmark and drops the moving glint band on top.
func (m LangPicker) renderlogo() string {
	wf, hf := float64(m.logow), float64(len(m.cells))
	prog := float64(m.frame) / float64(shinesweep)
	hi := 2.0 + shineband
	phase := hi - prog*(hi+shineband) // rides from bottom-right down to top-left

	var b strings.Builder
	for r, row := range m.cells {
		for c, cl := range row {
			fg, bg := cl.fg, cl.bg
			s := float64(c)/wf + float64(r)/hf
			if d := math.Abs(s - phase); d < shineband && lit(fg, bg) {
				t := (1 - d/shineband) * shinepeak
				fg, bg = boost(fg, t), boost(bg, t)
			}
			b.WriteString(sgr(fg, bg, cl.reverse))
			b.WriteRune(cl.ch)
		}
		b.WriteString("\x1b[0m")
		if r < len(m.cells)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func parsecells(line string) []cell {
	var cells []cell
	var fg, bg *rgb
	reverse := false
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if loc := ansiseq.FindStringIndex(line[i:]); loc != nil && loc[0] == 0 {
				fg, bg, reverse = applysgr(line[i+2:i+loc[1]-1], fg, bg, reverse)
				i += loc[1]
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		cells = append(cells, cell{ch: r, fg: fg, bg: bg, reverse: reverse})
		i += size
	}
	return cells
}

func applysgr(params string, fg, bg *rgb, rev bool) (*rgb, *rgb, bool) {
	if params == "" {
		params = "0"
	}
	tok := strings.Split(params, ";")
	for j := 0; j < len(tok); j++ {
		switch tok[j] {
		case "0":
			fg, bg, rev = nil, nil, false
		case "7":
			rev = true
		case "27":
			rev = false
		case "38", "48":
			if j+4 < len(tok) && tok[j+1] == "2" {
				c := &rgb{atoi(tok[j+2]), atoi(tok[j+3]), atoi(tok[j+4])}
				if tok[j] == "38" {
					fg = c
				} else {
					bg = c
				}
				j += 4
			}
		}
	}
	return fg, bg, rev
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func lit(fg, bg *rgb) bool { return maxc(fg) > shinefloor || maxc(bg) > shinefloor }

func maxc(c *rgb) int {
	if c == nil {
		return 0
	}
	m := c.r
	if c.g > m {
		m = c.g
	}
	if c.b > m {
		m = c.b
	}
	return m
}

func boost(c *rgb, t float64) *rgb {
	if c == nil {
		return nil
	}
	return &rgb{
		c.r + int(float64(255-c.r)*t),
		c.g + int(float64(255-c.g)*t),
		c.b + int(float64(255-c.b)*t),
	}
}

func writeint(b *strings.Builder, n int) {
	var buf [8]byte
	b.Write(strconv.AppendInt(buf[:0], int64(n), 10))
}

func writesgr(b *strings.Builder, fg, bg *rgb, rev bool) {
	b.WriteString("\x1b[0")
	if rev {
		b.WriteString(";7")
	}
	if fg != nil {
		b.WriteString(";38;2;")
		writeint(b, fg.r)
		b.WriteByte(';')
		writeint(b, fg.g)
		b.WriteByte(';')
		writeint(b, fg.b)
	}
	if bg != nil {
		b.WriteString(";48;2;")
		writeint(b, bg.r)
		b.WriteByte(';')
		writeint(b, bg.g)
		b.WriteByte(';')
		writeint(b, bg.b)
	}
	b.WriteByte('m')
}

func sgr(fg, bg *rgb, rev bool) string {
	var b strings.Builder
	writesgr(&b, fg, bg, rev)
	return b.String()
}

func stampcenter(base, over string, width int) string {
	left, leftw := ansirstrip(base)
	overw := lipgloss.Width(over)
	start := (width - overw) / 2
	if start < leftw+1 {
		start = leftw + 1 // never collide with the feet
	}
	gap := strings.Repeat(" ", start-leftw)
	tail := width - start - overw
	if tail < 0 {
		tail = 0
	}
	return left + "\x1b[0m" + gap + over + strings.Repeat(" ", tail)
}

func ansirstrip(s string) (string, int) {
	var lastidx, lastcol, col, i int
	for i < len(s) {
		if loc := ansiseq.FindStringIndex(s[i:]); loc != nil && loc[0] == 0 {
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		col++
		i += size
		if r != ' ' {
			lastidx = i
			lastcol = col
		}
	}
	return s[:lastidx], lastcol
}

func trimblanklines(s string) string {
	s = cursorseq.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for len(lines) > 0 {
		last := ansiseq.ReplaceAllString(lines[len(lines)-1], "")
		if strings.TrimSpace(last) != "" {
			break
		}
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
