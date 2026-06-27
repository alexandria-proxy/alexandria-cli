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

	scrollthumbst = lipgloss.NewStyle().Bold(true).Foreground(btngray)
	scrolltrackst = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

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
	scroll         int
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

func (p serverspanel) subcardheight(s subscription.Subscription) int {
	h := 5
	if s.Note != "" {
		h++
	}
	return h
}

func (p serverspanel) hittest(row, scroll int) (int, string, int) {
	if row < 0 {
		return -1, "", 0
	}
	if row < 1 {
		return -1, "title", 0
	}
	if row < 4 {
		return -1, "search", row - 1
	}
	target := row - 4 + scroll
	y := 0
	idx := 0
	for si, sub := range p.subs {
		if si > 0 {
			y++
		}
		h := p.subcardheight(sub)
		if target < y+h {
			return idx, "header", target - y
		}
		y += h
		idx++
		if !p.collapsed[sub.URL] {
			for range sub.Servers {
				if target < y+3 {
					return idx, "server", target - y
				}
				y += 3
				idx++
			}
		}
	}
	return -1, "", 0
}

func (p serverspanel) listrow(url string) (int, bool) {
	y := 0
	for si, sub := range p.subs {
		if si > 0 {
			y++
		}
		if sub.URL == url {
			return y, true
		}
		y += p.subcardheight(sub)
		if !p.collapsed[sub.URL] {
			y += 3 * len(sub.Servers)
		}
	}
	return 0, false
}

func (p serverspanel) itemspan(idx int) (int, int) {
	y := 0
	i := 0
	for si, sub := range p.subs {
		if si > 0 {
			y++
		}
		h := p.subcardheight(sub)
		if i == idx {
			return y, h
		}
		y += h
		i++
		if !p.collapsed[sub.URL] {
			for range sub.Servers {
				if i == idx {
					return y, 3
				}
				y += 3
				i++
			}
		}
	}
	return 0, 0
}

func (p serverspanel) listheight() int {
	y := 0
	for si, sub := range p.subs {
		if si > 0 {
			y++
		}
		y += p.subcardheight(sub)
		if !p.collapsed[sub.URL] {
			y += 3 * len(sub.Servers)
		}
	}
	return y
}

func (p *serverspanel) clampscroll(viewh int) {
	max := p.listheight() - viewh
	if max < 0 {
		max = 0
	}
	if p.scroll > max {
		p.scroll = max
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
}

func (p *serverspanel) ensurevisible(viewh int) {
	if viewh < 1 {
		return
	}
	start, h := p.itemspan(p.cursor)
	if start < p.scroll {
		p.scroll = start
	}
	if start+h > p.scroll+viewh {
		p.scroll = start + h - viewh
	}
	p.clampscroll(viewh)
}

func headerbtnat(cx, cardleft, usable int) int {
	x := cardleft + usable - 3
	for i := headerbtns - 1; i >= 0; i-- {
		w := lipgloss.Width(headerglyphs[i])
		start := x - w + 1
		if cx >= start && cx <= x {
			return i
		}
		x = start - 2
	}
	return -1
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

func (p serverspanel) render(width, height int, busyurl string, busybtn int, dropdown, anchorurl string, flash float64, chosenurl string, chosenidx int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	header := paneltitlest.Render(p.tr.ServersTitle)
	search := p.searchview(usable)

	if len(p.subs) == 0 {
		v := 0x80 + int(flash*float64(0xff-0x80))
		empty := lipgloss.NewStyle().
			Bold(flash > 0.3).
			Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", v, v, v))).
			Width(usable).
			Align(lipgloss.Center).
			Render(p.tr.NoSubs + "\n" + p.tr.AddSubHint)
		body := lipgloss.JoinVertical(lipgloss.Left, header, search, "", empty)
		return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)
	}

	sel := selitem{-1, -1}
	if items := p.items(); p.cursor >= 0 && p.cursor < len(items) {
		sel = items[p.cursor]
	}

	var blocks []string
	for si, sub := range p.subs {
		if si > 0 {
			blocks = append(blocks, "")
		}
		headsel := p.serversfocused && sel.subidx == si && sel.srvidx == -1
		collapsed := p.collapsed[sub.URL]
		bb := -1
		if sub.URL == busyurl {
			bb = busybtn
		}
		blocks = append(blocks, p.subcard(sub, usable, headsel, collapsed, bb))

		if !collapsed {
			var rows []string
			for ri, srv := range sub.Servers {
				srvsel := p.serversfocused && sel.subidx == si && sel.srvidx == ri
				chosen := sub.URL == chosenurl && ri == chosenidx
				rows = append(rows, p.servercard(srv, usable-2, srvsel, chosen))
			}
			if len(rows) > 0 {
				block := lipgloss.JoinVertical(lipgloss.Left, rows...)
				blocks = append(blocks, lipgloss.NewStyle().PaddingLeft(1).Render(block))
			}
		}
	}

	listlines := strings.Split(lipgloss.JoinVertical(lipgloss.Left, blocks...), "\n")
	total := len(listlines)
	viewh := height - 5
	if viewh < 1 {
		viewh = 1
	}
	scroll := p.scroll
	if scroll > total-viewh {
		scroll = total - viewh
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + viewh
	if end > total {
		end = total
	}
	visible := strings.Join(listlines[scroll:end], "\n")

	body := lipgloss.JoinVertical(lipgloss.Left, header, search, visible)
	panel := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(body)

	if total > viewh {
		panel = placeoverlay(width-2, 5, scrollbarcol(viewh, total, scroll), panel)
	}

	if dropdown != "" {
		if lr, ok := p.listrow(anchorurl); ok && lr-scroll >= 0 && lr-scroll < viewh {
			x := 2 + usable - lipgloss.Width(dropdown)
			if x < 2 {
				x = 2
			}
			panel = placeoverlay(x, 5+lr-scroll, dropdown, panel)
		}
	}
	return panel
}

func scrollbarcol(viewh, total, scroll int) string {
	thumb := viewh * viewh / total
	if thumb < 1 {
		thumb = 1
	}
	pos := 0
	if maxoff := total - viewh; maxoff > 0 {
		pos = scroll * (viewh - thumb) / maxoff
	}
	var b strings.Builder
	for i := 0; i < viewh; i++ {
		if i >= pos && i < pos+thumb {
			b.WriteString(scrollthumbst.Render("│"))
		} else {
			b.WriteString(scrolltrackst.Render("│"))
		}
		if i < viewh-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
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
		lines = append(lines, panelfaint.Italic(true).Width(bodyw).Align(lipgloss.Center).Render(cliprunes(s.Note, bodyw)))
		italics = append(italics, true)
	}
	if busy {
		return busycard(lines, italics, usable, busyphase())
	}
	return cardbox(lines, border, usable)
}

func (p serverspanel) servercard(s subscription.Server, w int, selected, chosen bool) string {
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

	namepart := namest.Render(name)
	if chosen {
		namepart += " " + pinggood.Render("●")
	}

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

	nameline := spread(namepart, ping, textw)
	protoline := panelfaint.Render(spread(proto, "", textw))
	content := lipgloss.JoinVertical(lipgloss.Left, nameline, protoline)

	if flag != "" {
		flagcol := lipgloss.Place(lipgloss.Width(flag), lipgloss.Height(content), lipgloss.Center, lipgloss.Center, flag)
		content = lipgloss.JoinHorizontal(lipgloss.Top, flagcol, " ", content)
	}

	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		lines[i] = " " + padline(ansi.Truncate(ln, bodyw, ""), bodyw) + " "
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
		out[i] = " " + padline(ansi.Truncate(ln, bodyw, ""), bodyw) + " "
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

func writegray(b *strings.Builder, v int, italic bool) {
	b.WriteString("\x1b[0")
	if italic {
		b.WriteString(";3")
	}
	b.WriteString(";38;2;")
	writeint(b, v)
	b.WriteByte(';')
	writeint(b, v)
	b.WriteByte(';')
	writeint(b, v)
	b.WriteByte('m')
}

func busycard(lines []string, italics []bool, usable int, phase float64) string {
	bodyw := usable - 4
	if bodyw < 1 {
		bodyw = 1
	}
	width := usable
	pos := phase * float64(width)
	shadeval := func(c int) int {
		d := math.Abs(float64(c) - pos)
		if half := float64(width) / 2; d > half {
			d = float64(width) - d
		}
		t := 1 - d/5.0
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return 110 + int(t*145)
	}

	var b strings.Builder
	b.Grow(width * (len(lines) + 2) * 24)

	paint := func(s string, start int, italic bool) {
		c := start
		for _, r := range s {
			writegray(&b, shadeval(c), italic)
			b.WriteRune(r)
			if r < 0x80 {
				c++
			} else {
				c += lipgloss.Width(string(r))
			}
		}
	}

	paint("╭"+strings.Repeat("─", width-2)+"╮", 0, false)
	for i, ln := range lines {
		b.WriteString("\x1b[0m\n")
		paint("│", 0, false)
		paint(" "+padline(cliprunes(ansi.Strip(ln), bodyw), bodyw)+" ", 1, i < len(italics) && italics[i])
		paint("│", width-1, false)
	}
	b.WriteString("\x1b[0m\n")
	paint("╰"+strings.Repeat("─", width-2)+"╯", 0, false)
	b.WriteString("\x1b[0m")
	return b.String()
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

func trimdec(f float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0")
}

func humanbytes(n int64) string {
	switch {
	case n >= 1<<40:
		return trimdec(float64(n)/(1<<40)) + "tb"
	case n >= 1<<30:
		return trimdec(float64(n)/(1<<30)) + "gb"
	case n >= 1<<20:
		return trimdec(float64(n)/(1<<20)) + "mb"
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
