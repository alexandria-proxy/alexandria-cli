package tui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infolink struct {
	label string
	url   string
}

type infoview struct {
	tr    i18n.Strings
	sub   subscription.Subscription
	focus int
	hover int
}

func newinfoview(tr i18n.Strings, sub subscription.Subscription) infoview {
	v := infoview{tr: tr, sub: sub, focus: 0, hover: -1}
	if len(v.links()) == 0 {
		v.focus = -1
	}
	return v
}

func (v infoview) links() []infolink {
	var out []infolink
	if v.sub.SupportURL != "" {
		out = append(out, infolink{"Support", v.sub.SupportURL})
	}
	if v.sub.AnnounceURL != "" {
		out = append(out, infolink{"Announcements", v.sub.AnnounceURL})
	}
	return out
}

func (v *infoview) focusup() {
	if n := len(v.links()); n > 0 {
		v.focus = (v.focus - 1 + n) % n
	}
}

func (v *infoview) focusdown() {
	if n := len(v.links()); n > 0 {
		v.focus = (v.focus + 1) % n
	}
}

func (v infoview) currentlink() string {
	links := v.links()
	if v.focus >= 0 && v.focus < len(links) {
		return links[v.focus].url
	}
	return ""
}

func (v infoview) statrows(w int) string {
	tr := v.tr
	s := v.sub

	type kv struct{ k, val string }
	rows := []kv{
		{tr.Used, humanbytes(s.UsedBytes) + " / " + totallabel(s.TotalBytes)},
		{tr.Expires, dateor(s.Expires.Format("02.01.2006"), s.Expires.IsZero())},
		{tr.Updated, s.UpdatedAt.Format("02.01.2006 15:04")},
		{tr.Autoupdate, fmtdur(s.AutoUpdate)},
	}
	if s.Note != "" {
		rows = append(rows, kv{tr.NoteLabel, oneline(s.Note)})
	}

	labelw := 0
	for _, r := range rows {
		if lw := lipgloss.Width(r.k); lw > labelw {
			labelw = lw
		}
	}

	valst := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	out := make([]string, len(rows))
	for i, r := range rows {
		valw := w - labelw - 2
		if valw < 1 {
			valw = 1
		}
		out[i] = panelfaint.Render(padline(r.k, labelw)) + "  " + valst.Render(cliprunes(r.val, valw))
	}
	return strings.Join(out, "\n")
}

func (v infoview) linkrow(l infolink, w int, focused, hovered bool) string {
	labelst := panelfaint
	urlst := panelfaint
	arrow := panelfaint.Render("›")
	if focused {
		labelst = lipgloss.NewStyle().Bold(true).Foreground(panelaccent)
		urlst = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		arrow = labelst.Render("›")
	}
	url := urlst.Render(shorturl(l.url))
	if hovered {
		url = urlst.Underline(true).Render(shorturl(l.url))
	}
	left := labelst.Render(padline(l.label, 13)) + "  " + url
	return spread(left, arrow, w)
}

func (v infoview) parts(usable int) []formpart {
	w := usable - 2
	if w < 1 {
		w = 1
	}
	name := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253")).Render(v.sub.Name)
	if v.sub.Pinned {
		name += panelfaint.Render(" 🖈")
	}

	ps := []formpart{
		{"title", paneltitlest.Render(v.tr.InfoTitle)},
		{"gap", ""},
		{"name", name},
		{"gap", ""},
		{"stat", v.statrows(w)},
	}
	if links := v.links(); len(links) > 0 {
		ps = append(ps, formpart{"gap", ""})
		for i, l := range links {
			ps = append(ps, formpart{"link", v.linkrow(l, w, i == v.focus, i == v.hover)})
		}
	}
	return ps
}

func (v infoview) render(width int) string {
	usable := width - 4
	if usable < 16 {
		usable = width
	}
	ps := v.parts(usable)
	views := make([]string, len(ps))
	for i, p := range ps {
		views[i] = p.view
	}
	body := lipgloss.JoinVertical(lipgloss.Left, views...)
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func dateor(s string, zero bool) string {
	if zero {
		return "∞"
	}
	return s
}

func shorturl(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.TrimSuffix(u, "/")
}

func setpointer(shape string) tea.Cmd {
	return func() tea.Msg {
		os.Stdout.WriteString("\x1b]22;" + shape + "\x07")
		return nil
	}
}

func openurl(u string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", u)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
		default:
			cmd = exec.Command("xdg-open", u)
		}
		_ = cmd.Start()
		return nil
	}
}
