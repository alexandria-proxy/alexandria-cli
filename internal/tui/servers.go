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
	panelAccent  = lipgloss.Color("#6C7BFF")
	panelDim     = lipgloss.Color("238")
	panelTitleSt = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253"))
	panelFaint   = lipgloss.NewStyle().Faint(true)
	pingSt       = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	barFullSt    = lipgloss.NewStyle().Foreground(panelAccent)
	barEmptySt   = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
)

type serversPanel struct {
	tr             i18n.Strings
	subs           []subscription.Subscription
	cursor         int
	search         textInput
	focused        bool
	serversFocused bool
}

type selItem struct {
	subIdx int
	srvIdx int
}

func (p serversPanel) items() []selItem {
	var items []selItem
	for si, sub := range p.subs {
		items = append(items, selItem{si, -1})
		for ri := range sub.Servers {
			items = append(items, selItem{si, ri})
		}
	}
	return items
}

func (p serversPanel) itemCount() int {
	n := 0
	for _, sub := range p.subs {
		n += 1 + len(sub.Servers)
	}
	return n
}

func newServersPanel(tr i18n.Strings) serversPanel {
	return serversPanel{tr: tr}
}

func (p serversPanel) searchView(usable int) string {
	cw := usable - 2
	if cw < 1 {
		cw = 1
	}
	border := panelDim
	var text string
	switch {
	case p.focused:
		border = btnGray
		text = p.search.view(cw, true, btnGray)
	case p.search.value != "":
		text = p.search.view(cw, false, lipgloss.Color("252"))
	default:
		text = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(clipRunes(p.tr.SearchHint, cw))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(cw).
		Render(text)
}

func (p serversPanel) render(width, height int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	header := panelTitleSt.Render(p.tr.ServersTitle)
	search := p.searchView(usable)

	blocks := []string{header, search}

	if len(p.subs) == 0 {
		empty := lipgloss.NewStyle().
			Faint(true).
			Width(usable).
			Align(lipgloss.Center).
			Render(p.tr.NoSubs + "\n" + p.tr.AddSubHint)
		blocks = append(blocks, "", empty)
	} else {
		sel := selItem{-1, -1}
		if items := p.items(); p.cursor >= 0 && p.cursor < len(items) {
			sel = items[p.cursor]
		}
		for si, sub := range p.subs {
			headSel := p.serversFocused && sel.subIdx == si && sel.srvIdx == -1
			blocks = append(blocks, p.subCard(sub, usable, headSel))

			var rows []string
			for ri, srv := range sub.Servers {
				srvSel := p.serversFocused && sel.subIdx == si && sel.srvIdx == ri
				rows = append(rows, p.serverCard(srv, usable-2, srvSel))
			}
			if len(rows) > 0 {
				block := lipgloss.JoinVertical(lipgloss.Left, rows...)
				blocks = append(blocks, lipgloss.NewStyle().PaddingLeft(1).Render(block))
			}
			blocks = append(blocks, "")
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func (p serversPanel) subCard(s subscription.Subscription, usable int, selected bool) string {
	bodyW := usable - 4
	if bodyW < 1 {
		bodyW = 1
	}
	border := panelDim
	if selected {
		border = btnGray
	}

	name := spread("⌄ "+lipgloss.NewStyle().Bold(true).Render(s.Name), "", bodyW)
	meta := panelFaint.Render(spread(
		s.UpdatedAt.Format("02.01.2006 15:04")+"  ·  "+p.tr.Autoupdate+" "+fmtDur(s.AutoUpdate),
		"", bodyW))
	usage := spread(
		usageBar(s.UsedBytes, s.TotalBytes)+"  "+humanBytes(s.UsedBytes)+" / "+totalLabel(s.TotalBytes),
		panelFaint.Render(p.tr.Expires+" "+s.Expires.Format("02.01.2006")),
		bodyW)

	lines := []string{name, meta, usage}
	if s.Note != "" {
		lines = append(lines, panelFaint.Italic(true).Render(spread(s.Note, "", bodyW)))
	}
	return cardBox(lines, border, usable)
}

func (p serversPanel) serverCard(s subscription.Server, w int, selected bool) string {
	bodyW := w - 4
	if bodyW < 1 {
		bodyW = 1
	}
	border := panelDim
	nameSt := lipgloss.NewStyle().Bold(true)
	if selected {
		border = btnGray
		nameSt = nameSt.Foreground(btnGray)
	}

	flag, name := splitFlag(s.Name)

	ping := panelFaint.Render("›")
	if s.PingMs > 0 {
		ping = pingSt.Render(fmt.Sprintf("%dms", s.PingMs)) + " ›"
	}

	proto := strings.ToLower(s.Protocol)
	if isJSONConfig(s.Raw) {
		proto += " / json"
	}

	textW := bodyW
	if flag != "" {
		textW -= lipgloss.Width(flag) + 1
	}
	if textW < 1 {
		textW = 1
	}

	nameLine := spread(nameSt.Render(name), ping, textW)
	protoLine := panelFaint.Render(spread(proto, "", textW))
	content := lipgloss.JoinVertical(lipgloss.Left, nameLine, protoLine)

	if flag != "" {
		flagCol := lipgloss.Place(lipgloss.Width(flag), lipgloss.Height(content), lipgloss.Center, lipgloss.Center, flag)
		content = lipgloss.JoinHorizontal(lipgloss.Top, flagCol, " ", content)
	}

	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		lines[i] = " " + padLine(ln, bodyW) + " "
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(border).
		Width(w - 2).
		Render(strings.Join(lines, "\n"))
}

func splitFlag(name string) (string, string) {
	runes := []rune(name)
	n := flagLen(runes)
	if n == 0 {
		return "", name
	}
	rest := strings.TrimSpace(string(runes[n:]))
	if rest == "" {
		return "", name
	}
	return string(runes[:n]), rest
}

func flagLen(runes []rune) int {
	if len(runes) >= 2 && isRegional(runes[0]) && isRegional(runes[1]) {
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

func isRegional(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func isJSONConfig(raw string) bool {
	s := strings.TrimSpace(raw)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func cardBox(lines []string, border lipgloss.Color, usable int) string {
	bodyW := usable - 4
	if bodyW < 1 {
		bodyW = 1
	}
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = " " + padLine(ln, bodyW) + " "
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

func padLine(s string, w int) string {
	if d := lipgloss.Width(s); d < w {
		return s + strings.Repeat(" ", w-d)
	}
	return s
}

func fmtDur(d time.Duration) string {
	h := int(d.Hours())
	if h >= 24 && h%24 == 0 {
		return fmt.Sprintf("%dd", h/24)
	}
	return fmt.Sprintf("%dh", h)
}

func humanBytes(n int64) string {
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

func totalLabel(n int64) string {
	if n <= 0 {
		return "∞"
	}
	return humanBytes(n)
}

func usageBar(used, total int64) string {
	const seg = 10
	frac := 0.12
	if total > 0 {
		frac = float64(used) / float64(total)
	}
	if frac > 1 {
		frac = 1
	}
	full := int(frac*seg + 0.5)
	return barFullSt.Render(strings.Repeat("▰", full)) + barEmptySt.Render(strings.Repeat("▱", seg-full))
}
