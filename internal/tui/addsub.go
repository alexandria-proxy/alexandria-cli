package tui

import (
	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addField int

const (
	fieldType addField = iota
	fieldName
	fieldURL
	fieldSubmit
	fieldCount
)

type formResult int

const (
	formNone formResult = iota
	formCancel
	formSubmit
)

type addForm struct {
	tr        i18n.Strings
	focus     addField
	typeIdx   int
	typeOpen  bool
	optCursor int
	name      string
	url       string
	loading   bool
	err       string
}

func newAddForm(tr i18n.Strings) addForm {
	return addForm{tr: tr}
}

func (f addForm) typeOptions() []string {
	return []string{f.tr.TypeSubscription, f.tr.TypeConfig, f.tr.TypeJSON}
}

func (f addForm) update(msg tea.KeyMsg) (addForm, formResult) {
	s := msg.String()

	if f.focus == fieldType && f.typeOpen {
		n := len(f.typeOptions())
		switch s {
		case "up":
			f.optCursor = (f.optCursor - 1 + n) % n
		case "down":
			f.optCursor = (f.optCursor + 1) % n
		case "enter", " ":
			f.typeIdx = f.optCursor
			f.typeOpen = false
		case "esc", "left":
			f.typeOpen = false
		}
		return f, formNone
	}

	switch s {
	case "esc":
		return f, formCancel
	case "right":
		if f.focus == fieldType {
			f.typeOpen = true
			f.optCursor = f.typeIdx
		}
		return f, formNone
	case "tab", "down":
		f.focus = (f.focus + 1) % fieldCount
		return f, formNone
	case "shift+tab", "up":
		f.focus = (f.focus - 1 + fieldCount) % fieldCount
		return f, formNone
	case "enter":
		switch f.focus {
		case fieldType:
			f.typeOpen = true
			f.optCursor = f.typeIdx
		case fieldSubmit:
			return f, formSubmit
		default:
			f.focus = (f.focus + 1) % fieldCount
		}
		return f, formNone
	case "backspace":
		switch f.focus {
		case fieldName:
			f.name = dropLast(f.name)
		case fieldURL:
			f.url = dropLast(f.url)
		}
		return f, formNone
	}

	typed := ""
	if msg.Type == tea.KeyRunes {
		typed = string(msg.Runes)
	} else if s == " " {
		typed = " "
	}
	if typed != "" {
		switch f.focus {
		case fieldName:
			f.name += typed
		case fieldURL:
			f.url += typed
		}
	}
	return f, formNone
}

func (f addForm) render(width int) string {
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	btn := lipgloss.PlaceHorizontal(usable-2, lipgloss.Center, f.submitButton())
	body := lipgloss.JoinVertical(lipgloss.Left,
		panelTitleSt.Render(f.tr.AddSubTitle), "",
		f.typeField(usable), "",
		labeledInput(f.tr.FieldName, f.name, f.focus == fieldName, usable), "",
		labeledInput(f.tr.FieldURL, f.url, f.focus == fieldURL, usable), "",
		btn,
	)
	if f.loading {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", fieldLabel(f.tr.Fetching))
	} else if f.err != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", errStyle.Render(" "+f.err))
	}
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func (f addForm) typeField(usable int) string {
	w := usable - 2
	opts := f.typeOptions()
	label := fieldLabel(f.tr.FieldType)

	if f.focus == fieldType && f.typeOpen {
		rows := make([]string, len(opts))
		for i, o := range opts {
			bg, fg := lipgloss.Color("236"), lipgloss.Color("250")
			if i == f.optCursor {
				bg, fg = btnGray, lipgloss.Color("16")
			}
			rows[i] = chipLine(o, w, bg, fg)
		}
		return lipgloss.JoinVertical(lipgloss.Left, label, lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	bg, fg := lipgloss.Color("237"), lipgloss.Color("250")
	if f.focus == fieldType {
		bg, fg = btnGray, lipgloss.Color("16")
	}
	chip := chipLine(spread(opts[f.typeIdx], "▼", w-2), w, bg, fg)
	return lipgloss.JoinVertical(lipgloss.Left, label, chip)
}

func (f addForm) submitButton() string {
	st := connectBtn
	if f.focus != fieldSubmit {
		st = connectBtnBlur
	}
	return st.Render(f.tr.AddBtn)
}

func labeledInput(label, val string, focused bool, usable int) string {
	border := panelDim
	var text string
	switch {
	case focused:
		border = btnGray
		text = lipgloss.NewStyle().Foreground(btnGray).Render(val) + cursorGlyph()
	case val != "":
		text = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(val)
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, true, false).
		BorderForeground(border).
		Width(usable - 2).
		Render(text)
	return lipgloss.JoinVertical(lipgloss.Left, fieldLabel(label), box)
}

func chipLine(text string, w int, bg, fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Width(w).
		Background(bg).
		Foreground(fg).
		Render(" " + padLine(text, w-2) + " ")
}

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0A6AC"))

func fieldLabel(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(" " + s)
}

func dropLast(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}
