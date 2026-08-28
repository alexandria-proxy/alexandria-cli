package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	backglyph = "‹"
	chipw     = 18
	sliderw   = 12
	dangerfg  = lipgloss.Color("#E0A6AC")
)

var (
	setlabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	setlabelsel  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	sethint      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	setsection   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253"))
	setvaluest   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	setarrowst   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	setdanger    = lipgloss.NewStyle().Foreground(dangerfg)
	setdangersel = lipgloss.NewStyle().Bold(true).Foreground(dangerfg)
	setpathst    = lipgloss.NewStyle().Faint(true)

	railst    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	knoboff   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	knobfocus = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	knobon    = lipgloss.NewStyle().Foreground(btngray)

	chipidle = lipgloss.Color("237")
	chiptext = lipgloss.Color("250")
	chiplist = lipgloss.Color("236")

	crumbroot = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253"))
	crumbback = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	crumbleaf = lipgloss.NewStyle().Bold(true).Foreground(panelaccent)
)

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

func chipwidth(usable int) int {
	w := chipw
	if room := usable - 18; w > room {
		w = room
	}
	if w < 8 {
		w = 8
	}
	return w
}

func selectchip(value string, focused bool, w int) string {
	bg, fg := chipidle, chiptext
	if focused {
		bg, fg = btngray, lipgloss.Color("16")
	}
	return chipline(spread(cliprunes(value, w-4), "⌄", w-2), w, bg, fg)
}

func stepperchip(value string, focused bool, w int) string {
	bg, fg := chipidle, chiptext
	if focused {
		bg, fg = btngray, lipgloss.Color("16")
	}
	return chipline(spread(cliprunes(value, w-5), "⌃⌄", w-2), w, bg, fg)
}

func optionlist(opts []string, cur int, w int) string {
	rows := make([]string, len(opts))
	for i, o := range opts {
		bg, fg := chiplist, chiptext
		if i == cur {
			bg, fg = btngray, lipgloss.Color("16")
		}
		rows[i] = chipline(cliprunes(o, w-2), w, bg, fg)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func sliderview(v, lo, hi int, focused bool) string {
	if hi <= lo {
		hi = lo + 1
	}
	pos := (v - lo) * (sliderw - 1) / (hi - lo)
	pos = clampint(pos, 0, sliderw-1)

	knob := "●"
	kst := knobfocus
	if focused {
		kst = knobon
	}
	var b strings.Builder
	b.WriteString(barfullst.Render(strings.Repeat("━", pos)))
	b.WriteString(kst.Render(knob))
	b.WriteString(baremptyst.Render(strings.Repeat("━", sliderw-1-pos)))
	return b.String()
}

func sliderat(cx, x0, lo, hi int) int {
	pos := clampint(cx-x0, 0, sliderw-1)
	return lo + pos*(hi-lo)/(sliderw-1)
}

func radiomark(on, focused bool) string {
	if on {
		if focused {
			return knobon.Render("◉")
		}
		return knobfocus.Render("◉")
	}
	return railst.Render("○")
}

func textbox(view string, focused, filled bool, w int) string {
	border := paneldim
	if focused {
		border = btngray
	} else if !filled {
		border = lipgloss.Color("237")
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true).
		BorderForeground(border).
		Width(w).
		Render(view)
}

func (m Menu) crumbparent(sc setscreen) string {
	if sc == screenroutes {
		return m.tr.SettingsAdvanced
	}
	return m.tr.SettingsTitle
}

func (m Menu) crumbleafname(sc setscreen) string {
	switch sc {
	case screensubs:
		return m.tr.SettingsSubs
	case screenadv:
		return m.tr.SettingsAdvanced
	case screenroutes:
		return m.tr.SettingsRoutes
	case screenlogs:
		return m.tr.LogsTitle
	case screenreset:
		return m.tr.SettingsReset
	}
	return ""
}

func (m Menu) crumbline(sc setscreen, avail int) string {
	if sc == screenroot {
		return "  " + crumbroot.Render(cliprunes(m.tr.SettingsTitle, avail))
	}
	back := crumbback.Render(backglyph+" ") + crumbroot.Render(cliprunes(m.crumbparent(sc), max0(avail-2)))
	leaf := "    " + crumbleaf.Render(cliprunes(m.crumbleafname(sc), max0(avail-4)))
	return lipgloss.JoinVertical(lipgloss.Left, back, leaf)
}
