package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	revealTick       = 40 * time.Millisecond
	revealFrames     = 28
	revealFramesBack = 12
	revealEdge       = 0.22
	revealPeak       = 0.85

	idleTick       = 80 * time.Millisecond
	ringCycle      = 5.0
	ringSweep      = 1.5
	ringMax        = 0.85
	ringWidth      = 0.06
	ringPeak       = 0.15
	ringDelay      = 2.0
	ringRetractDur = 0.18

	twoColMin = 96
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
	ringAt     time.Time
	retracting bool
	retractAt  time.Time
	ringFrom   float64
	ringTo     float64
	panel      serversPanel
	width      int
	height     int
}

func NewMenu(lang, mono, color string) Menu {
	monoCells, w := parseLogo(mono)
	colorCells, _ := parseLogo(color)
	tr := i18n.T(lang)
	return Menu{tr: tr, monoCells: monoCells, colorCells: colorCells, logoW: w, panel: newServersPanel(tr)}
}

func (m Menu) Init() tea.Cmd { return tea.Batch(tea.HideCursor, m.tick()) }

func (m Menu) tick() tea.Cmd {
	d := idleTick
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
				if m.connected {
					m.ringAt = time.Now()
				}
			}
		}
		if m.retracting && time.Since(m.retractAt).Seconds() >= ringRetractDur {
			m.retracting = false
		}
		return m, m.tick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, tea.HideCursor
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if n := m.panel.serverCount(); n > 0 {
				m.panel.cursor = (m.panel.cursor - 1 + n) % n
			}
			return m, nil
		case "down", "j":
			if n := m.panel.serverCount(); n > 0 {
				m.panel.cursor = (m.panel.cursor + 1) % n
			}
			return m, nil
		case "enter", " ":
			p := m.phase()
			rNow, rOn := m.ring()
			m.connected = !m.connected
			m.reverse = !m.connected
			m.revealing = true
			if m.reverse {
				m.frame = int(p / 2.0 * float64(revealFramesBack)) // resume from current coverage
			} else {
				m.frame = int((2.0 - p) / 2.0 * float64(revealFrames))
			}
			m.ringAt = time.Time{}
			if m.connected {
				m.since = time.Now()
				m.retracting = false
			} else {
				m.retracting = rOn
				m.ringFrom = rNow
				m.retractAt = time.Now()
				if rNow/ringMax > 0.4 {
					m.ringTo = ringMax + ringWidth
				} else {
					m.ringTo = 0
				}
			}
			return m, nil
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
	if m.width < twoColMin {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, unit)
	}

	leftW := m.width / 2
	rightW := m.width - leftW
	left := lipgloss.Place(leftW, m.height, lipgloss.Center, lipgloss.Center, unit)
	right := lipgloss.Place(rightW, m.height, lipgloss.Left, lipgloss.Top, m.panel.render(rightW, m.height))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Menu) renderLogo() string {
	wf, hf := float64(m.logoW), float64(len(m.colorCells))
	phase := m.phase()
	ringR, ringOn := m.ring()

	var b strings.Builder
	for r := range m.colorCells {
		for c := range m.colorCells[r] {
			cl := m.colorCells[r][c]
			if m.revealing {
				s := float64(c)/wf + float64(r)/hf
				switch {
				case s >= phase:
					if d := s - phase; d < revealEdge && lit(cl.fg, cl.bg) {
						t := (1 - d/revealEdge) * revealPeak
						cl.fg, cl.bg = boost(cl.fg, t), boost(cl.bg, t)
					}
				case r < len(m.monoCells) && c < len(m.monoCells[r]):
					cl = m.monoCells[r][c]
				}
			} else if !m.connected && r < len(m.monoCells) && c < len(m.monoCells[r]) {
				cl = m.monoCells[r][c]
			}
			if ringOn && lit(cl.fg, cl.bg) {
				nx, ny := float64(c)/wf-0.5, float64(r)/hf-0.5
				if d := math.Abs(math.Hypot(nx, ny) - ringR); d < ringWidth {
					t := (1 - d/ringWidth) * ringPeak
					cl.fg, cl.bg = glint(cl.fg, t), glint(cl.bg, t)
				}
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

func glint(c *rgb, t float64) *rgb {
	if c == nil {
		return nil
	}
	if 0.299*float64(c.r)+0.587*float64(c.g)+0.114*float64(c.b) < 160 {
		return boost(c, t)
	}
	f := 1 - t
	return &rgb{int(float64(c.r)*f + 0.5), int(float64(c.g)*f + 0.5), int(float64(c.b)*f + 0.5)}
}

func (m Menu) ring() (float64, bool) {
	if m.retracting {
		el := time.Since(m.retractAt).Seconds()
		if el >= ringRetractDur {
			return 0, false
		}
		return m.ringFrom + (m.ringTo-m.ringFrom)*(el/ringRetractDur), true
	}
	if !m.connected || m.revealing || m.ringAt.IsZero() {
		return 0, false
	}
	el := time.Since(m.ringAt).Seconds() - ringDelay
	if el < 0 {
		return 0, false
	}
	cyc := math.Mod(el, ringCycle)
	if cyc > ringSweep {
		return 0, false
	}
	return cyc / ringSweep * ringMax, true
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

func elapsed(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s%3600/60, s%60)
}
