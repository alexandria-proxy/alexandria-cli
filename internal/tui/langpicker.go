package tui

import (
	"regexp"
	"strings"
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

type LangPicker struct {
	logo   string
	cursor int
	chosen string
	width  int
	height int
}

func NewLangPicker(logo string) LangPicker {
	return LangPicker{logo: trimBlankLines(logo)}
}

func (m LangPicker) Init() tea.Cmd { return tea.HideCursor }

func (m LangPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	logoLines := strings.Split(m.logo, "\n")
	top := m.logo
	titleRow := title
	if len(logoLines) > 1 {
		top = strings.Join(logoLines[:len(logoLines)-1], "\n")
		titleRow = stampCenter(logoLines[len(logoLines)-1], title, lipgloss.Width(m.logo))
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
