package tui

import (
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	drawerw      = 18
	drawerframes = 6
	drawertop    = 2
	burgerw      = 3
)

type navid int

const (
	navservers navid = iota
	navsettings
	navlogs
)

var navitems = []navid{navservers, navsettings, navlogs}

var (
	burgerst  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	burgerhot = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(btngray)

	drawerrule = lipgloss.NewStyle().Foreground(paneldim)
	navrow     = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("250"))
	navrowsel  = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(btngray).Foreground(lipgloss.Color("16"))
	navrowact  = lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(lipgloss.Color("255"))
)

func (m Menu) contentw() int {
	w := m.width - m.drawervisible()
	if w < 1 {
		w = 1
	}
	return w
}

func (m Menu) burgerx() int {
	return max0(m.contentw() - burgerw - 2)
}

func (m Menu) burgery() int {
	return 1
}

func (m Menu) hitburger(x, y int) bool {
	return m.width > 0 && y == m.burgery() && x >= m.burgerx() && x < m.burgerx()+burgerw
}

func (m Menu) renderburger() string {
	st := burgerst
	if m.draweropen || m.burgerhover || m.focus == focusburger {
		st = burgerhot
	}
	return st.Render(" ≡ ")
}

func (m Menu) drawerwidth() int {
	w := drawerw
	if w > m.width-2 {
		w = m.width - 2
	}
	if w < 10 {
		w = max0(m.width)
	}
	return w
}

func (m Menu) drawerprogress() float64 {
	t := float64(m.drawerframe) / float64(drawerframes)
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return 1 - (1-t)*(1-t)
}

func (m Menu) draweranimating() bool {
	if m.draweropen {
		return m.drawerframe < drawerframes
	}
	return m.drawerframe > 0
}

func (m Menu) drawervisible() int {
	return int(math.Round(m.drawerprogress() * float64(m.drawerwidth())))
}

func (m Menu) navlabel(id navid) string {
	switch id {
	case navsettings:
		return m.tr.SettingsTitle
	case navlogs:
		return m.tr.LogsTitle
	}
	return m.tr.ServersTitle
}

func (m Menu) navrowview(id navid, inner int) string {
	mark := "  "
	if m.navsect == id {
		mark = "▸ "
	}
	st := navrow
	switch {
	case m.navidx == id:
		st = navrowsel
	case m.navsect == id:
		st = navrowact
	}
	return st.Width(inner).Render(mark + m.navlabel(id))
}

func (m Menu) renderdrawer() string {
	dw := m.drawerwidth()
	inner := max0(dw - 1)
	blank := strings.Repeat(" ", inner)
	rows := make([]string, 0, m.height)
	for y := 0; y < m.height; y++ {
		body := blank
		if i := y - drawertop; i >= 0 && i < len(navitems) {
			body = m.navrowview(navitems[i], inner)
		}
		rows = append(rows, drawerrule.Render("│")+body)
	}
	return strings.Join(rows, "\n")
}

func (m Menu) withdrawer(view string) string {
	if m.width == 0 || m.height == 0 {
		return view
	}
	if vw := m.drawervisible(); vw > 0 {
		lines := strings.Split(m.renderdrawer(), "\n")
		for i, l := range lines {
			lines[i] = ansi.Truncate(l, vw, "")
		}
		view = placeoverlay(m.contentw(), 0, strings.Join(lines, "\n"), view)
	}
	return placeoverlay(m.burgerx(), m.burgery(), m.renderburger(), view)
}

func (m Menu) navat(x, y int) int {
	if m.drawerframe < drawerframes || m.width == 0 {
		return -1
	}
	x0 := m.contentw()
	if x <= x0 || x >= m.width {
		return -1
	}
	if i := y - drawertop; i >= 0 && i < len(navitems) {
		return i
	}
	return -1
}

func (m Menu) opendrawer() (tea.Model, tea.Cmd) {
	var commit tea.Cmd
	if m.focus == focussettings {
		commit = m.commitinput(m.currentrows())
	}
	m.draweropen = true
	m.navidx = m.navsect
	m.burgerhover = true
	return m.withtick(commit)
}

func (m Menu) closedrawer() (tea.Model, tea.Cmd) {
	m.draweropen = false
	m.burgerhover = false
	return m.withtick(setpointer("default"))
}

func (m Menu) picknav() (tea.Model, tea.Cmd) {
	m.navsect = m.navidx
	m.mode = modelist
	m.enterright()
	return m.closedrawer()
}

func (m Menu) updatedrawer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "left", "h", "ctrl+b":
		return m.closedrawer()
	case "up", "k", "shift+tab":
		if m.navidx > navservers {
			m.navidx--
		}
		return m.withtick(nil)
	case "down", "j", "tab":
		if int(m.navidx) < len(navitems)-1 {
			m.navidx++
		}
		return m.withtick(nil)
	case "enter", " ", "right", "l":
		return m.picknav()
	}
	return m.withtick(nil)
}

func (m Menu) mousedrawer(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionMotion:
		hov := m.navat(msg.X, msg.Y)
		burger := m.hitburger(msg.X, msg.Y)
		if hov < 0 && !burger && !m.burgerhover {
			return m, nil
		}
		changed := burger != m.burgerhover
		if hov >= 0 && navid(hov) != m.navidx {
			m.navidx = navid(hov)
			changed = true
		}
		m.burgerhover = burger
		if !changed {
			return m, nil
		}
		shape := "default"
		if hov >= 0 || burger {
			shape = "pointer"
		}
		return m.withtick(setpointer(shape))
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if m.hitburger(msg.X, msg.Y) {
			return m.closedrawer()
		}
		if hov := m.navat(msg.X, msg.Y); hov >= 0 {
			m.navidx = navid(hov)
			return m.picknav()
		}
		if msg.X < m.contentw() {
			return m.closedrawer()
		}
	}
	return m, nil
}
