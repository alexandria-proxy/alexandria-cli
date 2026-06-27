package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const headerbtns = 3

var headerglyphs = [headerbtns]string{"↻", "⏱", "⋯"}

var (
	panelaccent  = lipgloss.Color("#6C7BFF")
	paneldim     = lipgloss.Color("238")
	paneltitlest = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("253"))
	panelfaint   = lipgloss.NewStyle().Faint(true)
	barfullst    = lipgloss.NewStyle().Foreground(panelaccent)
	baremptyst   = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

	headerbtnst  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	headerbtnsel = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(btngray)

	pinggood  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E0AC"))
	pingok    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0D6A6"))
	pingbad   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0A6AC"))
	pingworst = lipgloss.NewStyle().Foreground(lipgloss.Color("#C98A8F"))
	pingdead  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A86A70"))
)

func pingtext(ms int, tr i18n.Strings) string {
	if ms < 0 {
		return pingdead.Render(tr.Dead)
	}
	st := pingworst
	switch {
	case ms <= 100:
		st = pinggood
	case ms <= 200:
		st = pingok
	case ms <= 350:
		st = pingbad
	}
	return st.Render(fmt.Sprintf("%dms", ms))
}

type serverspanel struct {
	tr             i18n.Strings
	subs           []subscription.Subscription
	cursor         int
	btnidx         int
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
	items := make([]selitem, 0, p.itemcount())
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

func (p serverspanel) render(width, height int, busyurl string, busybtn int, dropdown, anchorurl string, flash float64) string {
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
	anchoridx := -1

	if len(p.subs) == 0 {
		v := 0x80 + int(flash*float64(0xff-0x80))
		empty := lipgloss.NewStyle().
			Bold(flash > 0.3).
			Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", v, v, v))).
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
			bb := -1
			if sub.URL == busyurl {
				bb = busybtn
			}
			if dropdown != "" && sub.URL == anchorurl {
				anchoridx = len(blocks)
			}
			blocks = append(blocks, p.subcard(sub, usable, headsel, collapsed, bb))

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
	panel := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)

	if anchoridx >= 0 {
		y := 1
		for _, blk := range blocks[:anchoridx] {
			y += lipgloss.Height(blk)
		}
		x := 2 + usable - lipgloss.Width(dropdown)
		if x < 2 {
			x = 2
		}
		panel = placeoverlay(x, y, dropdown, panel)
	}
	return panel
}

func (p serverspanel) headerstrip(selected bool) string {
	cells := make([]string, headerbtns)
	for i, g := range headerglyphs {
		if selected && p.btnidx == i {
			cells[i] = headerbtnsel.Render(g)
		} else {
			cells[i] = headerbtnst.Render(g)
		}
	}
	return strings.Join(cells, " ")
}

func (p serverspanel) subcard(s subscription.Subscription, usable int, selected, collapsed bool, busybtn int) string {
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

	busy := busybtn >= 0
	title := lipgloss.NewStyle().Bold(true).Render(s.Name)
	if s.Pinned {
		title += panelfaint.Render(" 🖈")
	}
	name := spread(arrow+title, p.headerstrip(selected), bodyw)
	meta := panelfaint.Render(spread(
		"  "+s.UpdatedAt.Format("02.01.2006 15:04")+" · "+p.tr.Autoupdate+" "+fmtdur(s.AutoUpdate),
		"", bodyw))

	var left string
	if s.TotalBytes > 0 {
		left = usagebar(s.UsedBytes, s.TotalBytes) + "  " + panelfaint.Render(humanbytes(s.UsedBytes)+" / "+totallabel(s.TotalBytes))
	} else {
		left = panelfaint.Render(p.tr.Used + " " + humanbytes(s.UsedBytes) + " (" + p.tr.Of + " ∞)")
	}
	usage := spread("  "+left,
		panelfaint.Render(p.tr.Expires+" "+s.Expires.Format("02.01.2006")),
		bodyw)

	lines := []string{name, meta, usage}
	italics := []bool{false, false, false}
	if s.Note != "" {
		lines = append(lines, panelfaint.Italic(true).Width(bodyw).Align(lipgloss.Center).Render(s.Note))
		italics = append(italics, true)
	}
	if busy {
		return busycard(lines, italics, usable, busyphase())
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
	if s.PingMs != 0 {
		ping = pingtext(s.PingMs, p.tr) + " " + arrowst.Render("›")
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

func busyphase() float64 {
	const cyc = 600
	return float64(time.Now().UnixMilli()%cyc) / cyc
}

func shimmershade(t float64) lipgloss.Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	v := 110 + int(t*145)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", v, v, v))
}

func busycard(lines []string, italics []bool, usable int, phase float64) string {
	bodyw := usable - 4
	if bodyw < 1 {
		bodyw = 1
	}
	width := usable
	pos := phase * float64(width)
	colat := func(c int) lipgloss.Style {
		d := math.Abs(float64(c) - pos)
		if half := float64(width) / 2; d > half {
			d = float64(width) - d
		}
		return lipgloss.NewStyle().Foreground(shimmershade(1 - d/5.0))
	}
	paint := func(s string, start int, italic bool) string {
		var b strings.Builder
		c := start
		for _, r := range s {
			st := colat(c)
			if italic {
				st = st.Italic(true)
			}
			b.WriteString(st.Render(string(r)))
			c += lipgloss.Width(string(r))
		}
		return b.String()
	}

	rows := []string{paint("╭"+strings.Repeat("─", width-2)+"╮", 0, false)}
	for i, ln := range lines {
		inner := " " + padline(ansi.Strip(ln), bodyw) + " "
		rows = append(rows, paint("│", 0, false)+paint(inner, 1, italics[i])+paint("│", width-1, false))
	}
	rows = append(rows, paint("╰"+strings.Repeat("─", width-2)+"╯", 0, false))
	return strings.Join(rows, "\n")
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
		return fmt.Sprintf("%.1ftb", float64(n)/(1<<40))
	case n >= 1<<30:
		return fmt.Sprintf("%.1fgb", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1fmb", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%dkb", n/(1<<10))
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
