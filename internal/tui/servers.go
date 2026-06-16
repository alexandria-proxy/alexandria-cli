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
	tr      i18n.Strings
	subs    []subscription.Subscription
	cursor  int
	query   string
	focused bool
}

func newServersPanel(tr i18n.Strings) serversPanel {
	return serversPanel{tr: tr}
}

func (p *serversPanel) backspace() {
	r := []rune(p.query)
	if len(r) > 0 {
		p.query = string(r[:len(r)-1])
	}
}

func (p serversPanel) searchView(usable int) string {
	border := panelDim
	var text string
	switch {
	case p.focused:
		border = btnGray
		text = lipgloss.NewStyle().Foreground(btnGray).Render(p.query) + cursorGlyph()
	case p.query != "":
		text = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(p.query)
	default:
		text = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(p.tr.SearchHint)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(usable - 2).
		Render(text)
}

func cursorGlyph() string {
	if (time.Now().UnixMilli()/500)%2 == 0 {
		return lipgloss.NewStyle().Reverse(true).Render(" ")
	}
	return " "
}

func (p serversPanel) serverCount() int {
	n := 0
	for _, s := range p.subs {
		n += len(s.Servers)
	}
	return n
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
		idx := 0
		for _, sub := range p.subs {
			blocks = append(blocks, p.subCard(sub, usable))
			for _, srv := range sub.Servers {
				blocks = append(blocks, p.serverCard(srv, usable, idx == p.cursor))
				idx++
			}
			blocks = append(blocks, "")
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
}

func (p serversPanel) subCard(s subscription.Subscription, usable int) string {
	bodyW := usable - 4
	if bodyW < 1 {
		bodyW = 1
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
	return cardBox(lines, panelDim, usable)
}

func (p serversPanel) serverCard(s subscription.Server, usable int, selected bool) string {
	bodyW := usable - 4
	if bodyW < 1 {
		bodyW = 1
	}

	border := panelDim
	bar := "  "
	nameSt := lipgloss.NewStyle().Bold(true)
	if selected {
		border = panelAccent
		bar = lipgloss.NewStyle().Foreground(panelAccent).Render("▌ ")
		nameSt = nameSt.Foreground(panelAccent)
	}

	top := spread(bar+s.Flag+" "+nameSt.Render(s.Name), pingSt.Render(fmt.Sprintf("%dms", s.PingMs))+" ›", bodyW)
	proto := panelFaint.Render(spread("   "+s.Protocol, "", bodyW))
	return cardBox([]string{top, proto}, border, usable)
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
