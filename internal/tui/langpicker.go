package tui

import (
	"fmt"
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
	btnGray     = lipgloss.Color("#B9C2C9")
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(btnGray)
	optionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	selBase     = lipgloss.NewStyle().Bold(true).Background(btnGray).Foreground(lipgloss.Color("16"))
	hintStyle   = lipgloss.NewStyle().Faint(true).PaddingRight(1)
)

var (
	ansiSeq   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cursorSeq = regexp.MustCompile(`\x1b\[\?25[lh]`)
)

const (
	shineTickRate = 45 * time.Millisecond
	shinePeriod   = 80 // frames per full cycle
	shineSweep    = 30 // frames the band actually spends crossing the logo
	shineBand     = 0.30
	shinePeak     = 0.92
	shineFloor    = 24 // colors dimmer than this stay put
)

type shineMsg time.Time

type rgb struct{ r, g, b int }

type cell struct {
	ch      rune
	fg, bg  *rgb
	reverse bool
}

type LangPicker struct {
	cells  [][]cell
	logoW  int
	frame  int
	cursor int
	chosen string
	width  int
	height int
}

func NewLangPicker(logo string) LangPicker {
	lines := strings.Split(trimBlankLines(logo), "\n")
	cells := make([][]cell, len(lines))
	w := 0
	for i, ln := range lines {
		cells[i] = parseCells(ln)
		if len(cells[i]) > w {
			w = len(cells[i])
		}
	}
	return LangPicker{cells: cells, logoW: w}
}

func (m LangPicker) Init() tea.Cmd { return tea.Batch(tea.HideCursor, shineTick()) }

func shineTick() tea.Cmd {
	return tea.Tick(shineTickRate, func(t time.Time) tea.Msg { return shineMsg(t) })
}

func (m LangPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shineMsg:
		m.frame = (m.frame + 1) % shinePeriod
		return m, shineTick()
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

	pad := selBase.Render(" ")
	rows := make([]string, len(languages))
	for i, lang := range languages {
		if i == m.cursor {
			marker := selBase.Render("●")
			text := selBase.Width(11).Render(lang.flag + lang.label)
			rows[i] = pad + marker + selBase.Render(" ") + text
		} else {
			rows[i] = optionStyle.Render("○ " + lang.flag + lang.label)
		}
	}

	title := titleStyle.Render(tr.ChooseLanguage)
	logo := m.renderLogo()
	logoLines := strings.Split(logo, "\n")
	top := logo
	titleRow := title
	if len(logoLines) > 1 {
		top = strings.Join(logoLines[:len(logoLines)-1], "\n")
		titleRow = stampCenter(logoLines[len(logoLines)-1], title, m.logoW)
	}

	body := lipgloss.JoinVertical(
		lipgloss.Center,
		top,
		titleRow,
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
	hint := hintStyle.Render(tr.Hint)

	if m.width == 0 || m.height == 0 {
		return body + "\n" + hint
	}
	placed := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, body)
	bottom := lipgloss.PlaceHorizontal(m.width, lipgloss.Right, hint)
	return placed + "\n" + bottom
}

func (m LangPicker) Chosen() string { return m.chosen }

// renderLogo paints the wordmark and drops the moving glint band on top.
func (m LangPicker) renderLogo() string {
	wf, hf := float64(m.logoW), float64(len(m.cells))
	prog := float64(m.frame) / float64(shineSweep)
	hi := 2.0 + shineBand
	phase := hi - prog*(hi+shineBand) // rides from bottom-right down to top-left

	var b strings.Builder
	for r, row := range m.cells {
		for c, cl := range row {
			fg, bg := cl.fg, cl.bg
			s := float64(c)/wf + float64(r)/hf
			if d := math.Abs(s - phase); d < shineBand && lit(fg, bg) {
				t := (1 - d/shineBand) * shinePeak
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

func parseCells(line string) []cell {
	var cells []cell
	var fg, bg *rgb
	reverse := false
	for i := 0; i < len(line); {
		if loc := ansiSeq.FindStringIndex(line[i:]); loc != nil && loc[0] == 0 {
			fg, bg, reverse = applySGR(line[i+2:i+loc[1]-1], fg, bg, reverse)
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		cells = append(cells, cell{ch: r, fg: fg, bg: bg, reverse: reverse})
		i += size
	}
	return cells
}

func applySGR(params string, fg, bg *rgb, rev bool) (*rgb, *rgb, bool) {
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

func lit(fg, bg *rgb) bool { return maxc(fg) > shineFloor || maxc(bg) > shineFloor }

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

func sgr(fg, bg *rgb, rev bool) string {
	p := []string{"0"}
	if rev {
		p = append(p, "7")
	}
	if fg != nil {
		p = append(p, fmt.Sprintf("38;2;%d;%d;%d", fg.r, fg.g, fg.b))
	}
	if bg != nil {
		p = append(p, fmt.Sprintf("48;2;%d;%d;%d", bg.r, bg.g, bg.b))
	}
	return "\x1b[" + strings.Join(p, ";") + "m"
}

func stampCenter(base, over string, width int) string {
	left, leftW := ansiRStrip(base)
	overW := lipgloss.Width(over)
	start := (width - overW) / 2
	if start < leftW+1 {
		start = leftW + 1 // never collide with the feet
	}
	gap := strings.Repeat(" ", start-leftW)
	tail := width - start - overW
	if tail < 0 {
		tail = 0
	}
	return left + "\x1b[0m" + gap + over + strings.Repeat(" ", tail)
}

func ansiRStrip(s string) (string, int) {
	var lastIdx, lastCol, col, i int
	for i < len(s) {
		if loc := ansiSeq.FindStringIndex(s[i:]); loc != nil && loc[0] == 0 {
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		col++
		i += size
		if r != ' ' {
			lastIdx = i
			lastCol = col
		}
	}
	return s[:lastIdx], lastCol
}

func trimBlankLines(s string) string {
	s = cursorSeq.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for len(lines) > 0 {
		last := ansiSeq.ReplaceAllString(lines[len(lines)-1], "")
		if strings.TrimSpace(last) != "" {
			break
		}
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
