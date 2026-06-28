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

type addfield int

const (
	fieldtype addfield = iota
	fieldname
	fieldurl
	fieldsubmit
	fieldcount
)

type formresult int

const (
	formnone formresult = iota
	formcancel
	formsubmit
)

type addform struct {
	tr        i18n.Strings
	focus     addfield
	typeidx   int
	typeopen  bool
	optcursor int
	name      textinput
	url       textinput
	loading   bool
	editing   bool
}

func newaddform(tr i18n.Strings) addform {
	return addform{tr: tr}
}

func (f addform) typeoptions() []string {
	return []string{f.tr.TypeSubscription, f.tr.TypeConfig, f.tr.TypeJSON}
}

func (f addform) update(msg tea.KeyMsg, cw int) (addform, formresult) {
	s := msg.String()

	if f.focus == fieldtype && f.typeopen {
		n := len(f.typeoptions())
		switch s {
		case "up":
			f.optcursor = (f.optcursor - 1 + n) % n
		case "down":
			f.optcursor = (f.optcursor + 1) % n
		case "enter", " ":
			f.typeidx = f.optcursor
			f.typeopen = false
		case "esc", "left":
			f.typeopen = false
		}
		return f, formnone
	}

	if f.focus == fieldname && f.name.handlekey(msg, cw) {
		return f, formnone
	}
	if f.focus == fieldurl && f.url.handlekey(msg, cw) {
		return f, formnone
	}

	switch s {
	case "esc":
		return f, formcancel
	case "right":
		if f.focus == fieldtype {
			f.typeopen = true
			f.optcursor = f.typeidx
		}
		return f, formnone
	case "tab", "down", "ctrl+down":
		f.focus = (f.focus + 1) % fieldcount
		return f, formnone
	case "shift+tab", "up", "ctrl+up":
		f.focus = (f.focus - 1 + fieldcount) % fieldcount
		return f, formnone
	case "enter":
		switch f.focus {
		case fieldtype:
			f.typeopen = true
			f.optcursor = f.typeidx
		case fieldsubmit:
			return f, formsubmit
		default:
			f.focus = (f.focus + 1) % fieldcount
		}
		return f, formnone
	}
	return f, formnone
}

func (f addform) render(width int) string {
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	title := f.tr.AddSubTitle
	if f.editing {
		title = f.tr.EditSubTitle
	}
	parts := []string{
		paneltitlest.Render(title), "",
		f.typefield(usable), "",
		labeledinput(f.tr.FieldName, f.name, f.focus == fieldname, usable), "",
		labeledinput(f.tr.FieldURL, f.url, f.focus == fieldurl, usable), "",
	}
	parts = append(parts, lipgloss.PlaceHorizontal(usable-2, lipgloss.Center, f.submitbutton()))

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func shimmer(text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return text
	}
	const radius = 3.0
	frac := float64(time.Now().UnixMilli()%1300) / 1300.0
	pos := -radius + frac*(float64(n)+2*radius)

	var b strings.Builder
	for i, r := range runes {
		t := 1 - math.Abs(float64(i)-pos)/radius
		if t < 0 {
			t = 0
		}
		shade := 120 + int(t*135)
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", shade, shade, shade))).
			Render(string(r)))
	}
	return b.String()
}

func (f addform) typefield(usable int) string {
	w := usable - 2
	opts := f.typeoptions()
	label := fieldlabel(f.tr.FieldType)

	if f.focus == fieldtype && f.typeopen {
		rows := make([]string, len(opts))
		for i, o := range opts {
			bg, fg := lipgloss.Color("236"), lipgloss.Color("250")
			if i == f.optcursor {
				bg, fg = btngray, lipgloss.Color("16")
			}
			rows[i] = chipline(o, w, bg, fg)
		}
		return lipgloss.JoinVertical(lipgloss.Left, label, lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	bg, fg := lipgloss.Color("237"), lipgloss.Color("250")
	if f.focus == fieldtype {
		bg, fg = btngray, lipgloss.Color("16")
	}
	chip := chipline(spread(opts[f.typeidx], "▼", w-2), w, bg, fg)
	return lipgloss.JoinVertical(lipgloss.Left, label, chip)
}

func (f addform) submitbutton() string {
	if f.loading {
		return shimmer(f.tr.Fetching)
	}
	st := connectbtn
	if f.focus != fieldsubmit {
		st = connectbtnblur
	}
	label := f.tr.AddBtn
	if f.editing {
		label = f.tr.SaveBtn
	}
	return st.Render(label)
}

func labeledinput(label string, ti textinput, focused bool, usable int) string {
	cw := usable - 2
	if cw < 1 {
		cw = 1
	}
	border := paneldim
	var text string
	switch {
	case focused:
		border = btngray
		text = ti.view(cw, true, btngray)
	case ti.value != "":
		text = ti.view(cw, false, lipgloss.Color("252"))
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, true, false).
		BorderForeground(border).
		Width(cw).
		Render(text)
	return lipgloss.JoinVertical(lipgloss.Left, fieldlabel(label), box)
}

func chipline(text string, w int, bg, fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Width(w).
		Background(bg).
		Foreground(fg).
		Render(" " + padline(text, w-2) + " ")
}

var errstyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0A6AC"))

func fieldlabel(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(" " + s)
}
