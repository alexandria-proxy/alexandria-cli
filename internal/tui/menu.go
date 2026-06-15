package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	revealTick       = 40 * time.Millisecond
	revealFrames     = 28
	revealFramesBack = 16
	revealEdge       = 0.22
	revealPeak       = 0.85
)

var (
	connectBtn    = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(btnGray).Foreground(lipgloss.Color("16"))
	disconnectBtn = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("#E0A6AC")).Foreground(lipgloss.Color("16"))
	timerStyle    = lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(btnGray)
)

type menuTickMsg struct{}

type Menu struct {
	tr         i18n.Strings
	colorCells [][]cell
	monoCells  [][]cell
	logoW      int
	connected  bool
	revealing  bool
	reverse    bool
	frame      int
	since      time.Time
	width      int
	height     int
}

func NewMenu(lang, mono, color string) Menu {
	monoCells, w := parseLogo(mono)
	colorCells, _ := parseLogo(color)
	return Menu{tr: i18n.T(lang), monoCells: monoCells, colorCells: colorCells, logoW: w}
}

func (m Menu) Init() tea.Cmd { return tea.HideCursor }

func (m Menu) tick() tea.Cmd {
	d := time.Second
	if m.revealing {
		d = revealTick
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return menuTickMsg{} })
}

func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case menuTickMsg:
		if m.revealing {
			m.frame++
			end := revealFrames
			if m.reverse {
				end = revealFramesBack
			}
			if m.frame >= end {
				m.revealing = false
			}
		}
		if m.connected || m.revealing {
			return m, m.tick()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, tea.HideCursor
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "enter", " ":
			p := m.phase()
			m.connected = !m.connected
			m.reverse = !m.connected
			m.revealing = true
			if m.reverse {
				m.frame = int(p / 2.0 * float64(revealFramesBack)) // resume from current coverage
			} else {
				m.frame = int((2.0 - p) / 2.0 * float64(revealFrames))
			}
			if m.connected {
				m.since = time.Now()
			}
			return m, m.tick()
		}
	}
	return m, nil
}

func (m Menu) View() string {
	logo := m.renderLogo()

	var btn string
	if m.connected {
		btn = lipgloss.JoinHorizontal(
			lipgloss.Center,
			disconnectBtn.Render(m.tr.Disconnect),
			timerStyle.Render("⏱"+elapsed(time.Since(m.since))),
		)
	} else {
		btn = connectBtn.Render(m.tr.Connect)
	}

	unit := lipgloss.JoinVertical(lipgloss.Center, logo, "", btn)
	if m.width == 0 || m.height == 0 {
		return unit
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Center, unit)
}

func (m Menu) renderLogo() string {
	if !m.revealing {
		if m.connected {
			return cellsToString(m.colorCells)
		}
		return cellsToString(m.monoCells)
	}

	wf, hf := float64(m.logoW), float64(len(m.colorCells))
	phase := m.phase()

	var b strings.Builder
	for r := range m.colorCells {
		for c := range m.colorCells[r] {
			cc := m.colorCells[r][c]
			s := float64(c)/wf + float64(r)/hf
			cl := cc
			switch {
			case s >= phase:
				if d := s - phase; d < revealEdge && lit(cc.fg, cc.bg) {
					t := (1 - d/revealEdge) * revealPeak
					cl.fg, cl.bg = boost(cc.fg, t), boost(cc.bg, t)
				}
			case r < len(m.monoCells) && c < len(m.monoCells[r]):
				cl = m.monoCells[r][c]
			}
			b.WriteString(sgr(cl.fg, cl.bg, cl.reverse))
			b.WriteRune(cl.ch)
		}
		b.WriteString("\x1b[0m")
		if r < len(m.colorCells)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Menu) phase() float64 {
	switch {
	case m.revealing && m.reverse:
		return float64(m.frame) / float64(revealFramesBack) * 2.0
	case m.revealing:
		return 2.0 - float64(m.frame)/float64(revealFrames)*2.0
	case m.connected:
		return 0.0
	default:
		return 2.0
	}
}

func cellsToString(grid [][]cell) string {
	var b strings.Builder
	for r, row := range grid {
		for _, cl := range row {
			b.WriteString(sgr(cl.fg, cl.bg, cl.reverse))
			b.WriteRune(cl.ch)
		}
		b.WriteString("\x1b[0m")
		if r < len(grid)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func elapsed(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s%3600/60, s%60)
}
