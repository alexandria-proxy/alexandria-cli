package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	"github.com/charmbracelet/lipgloss"
)

var (
	panelaccent  = lipgloss.Color("#6C7BFF")
	paneldim     = lipgloss.Color("238")
	paneltitlest = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253"))
	panelfaint   = lipgloss.NewStyle().Faint(true)
	pingst       = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	barfullst    = lipgloss.NewStyle().Foreground(panelaccent)
	baremptyst   = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
)

type serverspanel struct {
	tr             i18n.Strings
	subs           []subscription.Subscription
	cursor         int
	search         textinput
	focused        bool
	serversfocused bool
	collapsed      map[string]bool
}

type selitem struct {
	subidx int
	srvidx int
}

func (p serverspanel) items() []selitem {
	var items []selitem
	for si, sub := range p.subs {
		items = append(items, selitem{si, -1})
		if p.collapsed[sub.URL] {
			continue
		}
		for ri := range sub.Servers {
			items = append(items, selitem{si, ri})
		}
	}
	return items
}

func (p serverspanel) itemcount() int {
	n := 0
	for _, sub := range p.subs {
		n++
		if !p.collapsed[sub.URL] {
			n += len(sub.Servers)
		}
	}
	return n
}

func newserverspanel(tr i18n.Strings) serverspanel {
	return serverspanel{tr: tr, collapsed: map[string]bool{}}
}

func (p serverspanel) searchview(usable int) string {
	cw := usable - 2
	if cw < 1 {
		cw = 1
	}
	border := paneldim
	var text string
	switch {
	case p.focused:
		border = btngray
		text = p.search.view(cw, true, btngray)
	case p.search.value != "":
		text = p.search.view(cw, false, lipgloss.Color("252"))
	default:
		text = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(cliprunes(p.tr.SearchHint, cw))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(cw).
		Render(text)
}

func (p serverspanel) render(width, height int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	header := paneltitlest.Render(p.tr.ServersTitle)
	search := p.searchview(usable)

	blocks := []string{header, search}

	if len(p.subs) == 0 {
		empty := lipgloss.NewStyle().
			Faint(true).
			Width(usable).
			Align(lipgloss.Center).
			Render(p.tr.NoSubs + "\n" + p.tr.AddSubHint)
		blocks = append(blocks, "", empty)
	} else {
		sel := selitem{-1, -1}
		if items := p.items(); p.cursor >= 0 && p.cursor < len(items) {
			sel = items[p.cursor]
		}
		for si, sub := range p.subs {
			headsel := p.serversfocused && sel.subidx == si && sel.srvidx == -1
			collapsed := p.collapsed[sub.URL]
			blocks = append(blocks, p.subcard(sub, usable, headsel, collapsed))

			if !collapsed {
				var rows []string
				for ri, srv := range sub.Servers {
					srvsel := p.serversfocused && sel.subidx == si && sel.srvidx == ri
					rows = append(rows, p.servercard(srv, usable-2, srvsel))
				}
				if len(rows) > 0 {
					block := lipgloss.JoinVertical(lipgloss.Left, rows...)
					blocks = append(blocks, lipgloss.NewStyle().PaddingLeft(1).Render(block))
				}
			}
			blocks = append(blocks, "")
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func (p serverspanel) subcard(s subscription.Subscription, usable int, selected, collapsed bool) string {
	bodyw := usable - 4
	if bodyw < 1 {
		bodyw = 1
	}
	border := paneldim
	if selected {
		border = btngray
	}

	arrow := "⌄ "
	if collapsed {
		arrow = "› "
	}
	name := spread(arrow+lipgloss.NewStyle().Bold(true).Render(s.Name), "", bodyw)
	meta := panelfaint.Render(spread(
		s.UpdatedAt.Format("02.01.2006 15:04")+"  ·  "+p.tr.Autoupdate+" "+fmtdur(s.AutoUpdate),
		"", bodyw))
	usage := spread(
		usagebar(s.UsedBytes, s.TotalBytes)+"  "+humanbytes(s.UsedBytes)+" / "+totallabel(s.TotalBytes),
		panelfaint.Render(p.tr.Expires+" "+s.Expires.Format("02.01.2006")),
		bodyw)

	lines := []string{name, meta, usage}
	if s.Note != "" {
		lines = append(lines, panelfaint.Italic(true).Render(spread(s.Note, "", bodyw)))
	}
	return cardbox(lines, border, usable)
}

func (p serverspanel) servercard(s subscription.Server, w int, selected bool) string {
	bodyw := w - 4
	if bodyw < 1 {
		bodyw = 1
	}
	border := paneldim
	namest := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	arrowst := panelfaint
	if selected {
		border = btngray
		arrowst = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	}

	flag, name := splitflag(s.Name)

	ping := arrowst.Render("›")
	if s.PingMs > 0 {
		ping = pingst.Render(fmt.Sprintf("%dms", s.PingMs)) + " " + arrowst.Render("›")
	}

	proto := strings.ToLower(s.Protocol)
	if isjsonconfig(s.Raw) {
		proto += " / json"
	}

	textw := bodyw
	if flag != "" {
		textw -= lipgloss.Width(flag) + 1
	}
	if textw < 1 {
		textw = 1
	}

	nameline := spread(namest.Render(name), ping, textw)
	protoline := panelfaint.Render(spread(proto, "", textw))
	content := lipgloss.JoinVertical(lipgloss.Left, nameline, protoline)

	if flag != "" {
		flagcol := lipgloss.Place(lipgloss.Width(flag), lipgloss.Height(content), lipgloss.Center, lipgloss.Center, flag)
		content = lipgloss.JoinHorizontal(lipgloss.Top, flagcol, " ", content)
	}

	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		lines[i] = " " + padline(ln, bodyw) + " "
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(border).
		Width(w - 2).
		Render(strings.Join(lines, "\n"))
}

func splitflag(name string) (string, string) {
	runes := []rune(name)
	n := flaglen(runes)
	if n == 0 {
		return "", name
	}
	rest := strings.TrimSpace(string(runes[n:]))
	if rest == "" {
		return "", name
	}
	return string(runes[:n]), rest
}

func flaglen(runes []rune) int {
	if len(runes) >= 2 && isregional(runes[0]) && isregional(runes[1]) {
		return 2
	}
	if len(runes) > 0 && (runes[0] == 0x1F3F4 || runes[0] == 0x1F3F3) {
		n := 1
		for n < len(runes) {
			switch r := runes[n]; {
			case r == 0x200D, r == 0xFE0F, r == 0x2620, r == 0x1F308, r >= 0xE0020 && r <= 0xE007F:
				n++
			default:
				return n
			}
		}
		return n
	}
	return 0
}

func isregional(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func isjsonconfig(raw string) bool {
	s := strings.TrimSpace(raw)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func cardbox(lines []string, border lipgloss.Color, usable int) string {
	bodyw := usable - 4
	if bodyw < 1 {
		bodyw = 1
	}
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = " " + padline(ln, bodyw) + " "
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(usable - 2).
		Render(strings.Join(out, "\n"))
}

func spread(left, right string, w int) string {
	g := w - lipgloss.Width(left) - lipgloss.Width(right)
	if g < 1 {
		g = 1
	}
	return left + strings.Repeat(" ", g) + right
}

func padline(s string, w int) string {
	if d := lipgloss.Width(s); d < w {
		return s + strings.Repeat(" ", w-d)
	}
	return s
}

func fmtdur(d time.Duration) string {
	h := int(d.Hours())
	if h >= 24 && h%24 == 0 {
		return fmt.Sprintf("%dd", h/24)
	}
	return fmt.Sprintf("%dh", h)
}

func humanbytes(n int64) string {
	switch {
	case n >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(n)/(1<<40))
	case n >= 1<<30:
		return fmt.Sprintf("%d GB", n/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%d MB", n/(1<<20))
	default:
		return fmt.Sprintf("%d KB", n/(1<<10))
	}
}

func totallabel(n int64) string {
	if n <= 0 {
		return "∞"
	}
	return humanbytes(n)
}

func usagebar(used, total int64) string {
	const seg = 10
	frac := 0.12
	if total > 0 {
		frac = float64(used) / float64(total)
	}
	if frac > 1 {
		frac = 1
	}
	full := int(frac*seg + 0.5)
	return barfullst.Render(strings.Repeat("▰", full)) + baremptyst.Render(strings.Repeat("▱", seg-full))
}
