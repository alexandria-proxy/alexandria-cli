package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alexandria-proxy/alexandria-cli/internal/autostart"
)

const (
	settingstop  = 3
	settingsstep = 3
)

type settingid int

const (
	setautostart settingid = iota
)

var settingitems = []settingid{setautostart}

var (
	setlabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	setlabelsel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	sethint     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	railst    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	knoboff   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	knobfocus = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	knobon    = lipgloss.NewStyle().Foreground(btngray)
)

type autostartmsg struct {
	on   bool
	err  string
	root bool
}

func setautostartcmd(on bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if on {
			err = autostart.Enable()
		} else {
			err = autostart.Disable()
		}
		if err != nil {
			return autostartmsg{on: !on, err: err.Error(), root: errors.Is(err, autostart.ErrNeedsRoot)}
		}
		return autostartmsg{on: on}
	}
}

func (m Menu) settinglabel(id settingid) (string, string) {
	if id == setautostart {
		return m.tr.AutostartLabel, m.tr.AutostartHint
	}
	return "", ""
}

func (m Menu) settingvalue(id settingid) bool {
	if id == setautostart {
		return m.autostart
	}
	return false
}

func toggleview(on, focused bool) string {
	if on {
		return railst.Render("░░") + knobon.Render("██")
	}
	st := knoboff
	if focused {
		st = knobfocus
	}
	return st.Render("██") + railst.Render("░░")
}

func (m Menu) labelwidth() int {
	w := 0
	for _, id := range settingitems {
		label, _ := m.settinglabel(id)
		if lw := lipgloss.Width(label); lw > w {
			w = lw
		}
	}
	return w
}

func (m Menu) settingrow(id settingid, usable int, focused bool) string {
	label, hint := m.settinglabel(id)
	mark, st := "  ", setlabel
	if focused {
		mark, st = "▸ ", setlabelsel
	}
	toggle := toggleview(m.settingvalue(id), focused)
	gap := m.labelwidth() - lipgloss.Width(label) + 4
	if room := usable - 2 - m.labelwidth() - lipgloss.Width(toggle); room < gap {
		gap = max0(room)
	}
	if gap < 1 {
		gap = 1
	}
	hintw := max0(usable - 2)
	return lipgloss.JoinVertical(lipgloss.Left,
		st.Render(mark+label)+strings.Repeat(" ", gap)+toggle,
		"  "+sethint.Width(hintw).Render(hint),
	)
}

func (m Menu) rendersettings(width int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}
	parts := []string{paneltitlest.Render(m.tr.SettingsTitle), ""}
	for i, id := range settingitems {
		if i > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, m.settingrow(id, usable, m.focus == focussettings && m.setidx == i))
	}
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m Menu) settingat(x, y int) int {
	px, py, pw := m.panelgeom()
	if x < px+2 || x >= px+pw {
		return -1
	}
	off := y - py - settingstop
	if off < 0 || off%settingsstep != 0 {
		return -1
	}
	if i := off / settingsstep; i < len(settingitems) {
		return i
	}
	return -1
}

func (m Menu) toggleseting(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(settingitems) {
		return m, nil
	}
	switch settingitems[i] {
	case setautostart:
		m.autostart = !m.autostart
		return m.withtick(setautostartcmd(m.autostart))
	}
	return m, nil
}

func (m Menu) updatesettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "left", "h":
		m.focus = focusconnect
		return m.withtick(nil)
	case "up", "k", "shift+tab":
		if m.setidx > 0 {
			m.setidx--
		}
		return m.withtick(nil)
	case "down", "j", "tab":
		if m.setidx < len(settingitems)-1 {
			m.setidx++
		}
		return m.withtick(nil)
	case "enter", " ", "right", "l":
		return m.toggleseting(m.setidx)
	}
	return m, nil
}
