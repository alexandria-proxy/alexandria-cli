package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
)

const (
	logtop     = 6
	logsrcw    = 9
	loglvlw    = 6
	logpolling = 700 * time.Millisecond
)

var (
	logtimest = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	logsrcst  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	logownst  = lipgloss.NewStyle().Foreground(panelaccent)
	logerrst  = lipgloss.NewStyle().Foreground(dangerfg)
	logwarnst = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0D6A6"))
	loginfost = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	logtextst = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	logmetast = lipgloss.NewStyle().Faint(true)
)

type logsmsg struct {
	lines []ipc.Logline
	seq   int64
}

type logview struct {
	lines    []ipc.Logline
	seq      int64
	scroll   int
	follow   bool
	filter   textinput
	filteron bool
}

func newlogview() logview {
	return logview{follow: true}
}

var logsources = []string{"daemon", "xray", "sing-box"}

var loglevels = []string{"info", "warn", "error"}

type logquery struct {
	srcs  []string
	lvls  []string
	terms []string
	nots  []string
}

func inlist(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func parselogquery(raw string) logquery {
	var q logquery
	for _, word := range strings.Fields(strings.ToLower(raw)) {
		if neg := strings.TrimLeft(word, "-!"); neg != word {
			if neg != "" {
				q.nots = append(q.nots, neg)
			}
			continue
		}
		switch {
		case inlist(logsources, word):
			q.srcs = append(q.srcs, word)
		case inlist(loglevels, word):
			q.lvls = append(q.lvls, word)
		default:
			q.terms = append(q.terms, word)
		}
	}
	return q
}

func (q logquery) empty() bool {
	return len(q.srcs) == 0 && len(q.lvls) == 0 && len(q.terms) == 0 && len(q.nots) == 0
}

func (q logquery) match(l ipc.Logline) bool {
	if len(q.srcs) > 0 && !inlist(q.srcs, strings.ToLower(l.Src)) {
		return false
	}
	if len(q.lvls) > 0 && !inlist(q.lvls, strings.ToLower(l.Lvl)) {
		return false
	}
	hay := strings.ToLower(l.Src + " " + l.Lvl + " " + l.Text)
	for _, t := range q.terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	for _, n := range q.nots {
		if strings.Contains(hay, n) {
			return false
		}
	}
	return true
}

func (v logview) visible() []ipc.Logline {
	q := parselogquery(v.filter.value)
	if q.empty() {
		return v.lines
	}
	out := make([]ipc.Logline, 0, len(v.lines))
	for _, l := range v.lines {
		if q.match(l) {
			out = append(out, l)
		}
	}
	return out
}

func logscmd(since int64) tea.Cmd {
	return func() tea.Msg {
		resp, err := ipc.Send(ipc.Request{Cmd: "logs", Since: since})
		if err != nil || !resp.OK {
			return logsmsg{seq: since}
		}
		return logsmsg{lines: resp.Logs, seq: resp.Seq}
	}
}

func logstick() tea.Cmd {
	return tea.Tick(logpolling, func(time.Time) tea.Msg { return logtickmsg{} })
}

type logtickmsg struct{}

func (v *logview) absorb(msg logsmsg) {
	if msg.seq < v.seq {
		v.lines = nil
		v.seq = 0
	}
	if len(msg.lines) > 0 {
		v.lines = append(v.lines, msg.lines...)
		if n := len(v.lines) - logring; n > 0 {
			v.lines = append(v.lines[:0], v.lines[n:]...)
		}
	}
	v.seq = msg.seq
}

const logring = 2000

func lvlstyle(lvl string) lipgloss.Style {
	switch lvl {
	case "error":
		return logerrst
	case "warn":
		return logwarnst
	}
	return loginfost
}

func srcstyle(src string) lipgloss.Style {
	if src == "daemon" {
		return logownst
	}
	return logsrcst
}

func (v logview) rowview(l ipc.Logline, usable int) string {
	stamp := time.Unix(l.At, 0).Format("15:04:05")
	head := logtimest.Render(stamp) + "  " +
		srcstyle(l.Src).Render(padline(cliprunes(l.Src, logsrcw), logsrcw)) + " " +
		lvlstyle(l.Lvl).Render(padline(cliprunes(l.Lvl, loglvlw), loglvlw)) + " "

	textw := max0(usable - lipgloss.Width(head))
	if textw < 12 {
		return logtextst.Width(usable).Render(cliprunes(l.Text, usable))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, head, logtextst.Width(textw).Render(l.Text))
}

func (v logview) body(usable int) []string {
	var out []string
	for _, l := range v.visible() {
		out = append(out, strings.Split(v.rowview(l, usable), "\n")...)
	}
	return out
}

func (m Menu) logfilterview(usable int) string {
	cw := max0(usable - 2)
	if cw < 1 {
		cw = 1
	}
	border := paneldim
	var text string
	switch {
	case m.logs.filteron:
		border = btngray
		text = m.logs.filter.view(cw, true, btngray)
	case m.logs.filter.value != "":
		text = m.logs.filter.view(cw, false, lipgloss.Color("252"))
	default:
		text = sethint.Render(cliprunes(m.tr.LogsFilterHint, cw))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(cw).
		Render(text)
}

func (m Menu) renderlogs(width, height int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	state := m.tr.LogsPaused
	if m.logs.follow {
		state = m.tr.LogsFollow
	}
	shown := len(m.logs.visible())
	count := strconv.Itoa(shown)
	if shown != len(m.logs.lines) {
		count += "/" + strconv.Itoa(len(m.logs.lines))
	}
	meta := logmetast.Render(state + " · " + count + " " + m.tr.LogsLines + " · " + m.tr.LogsHint)
	head := lipgloss.JoinVertical(lipgloss.Left,
		paneltitlest.Render(m.tr.LogsTitle), meta, m.logfilterview(usable))

	body := m.logs.body(usable)
	total := len(body)
	viewh := height - logtop
	if viewh < 1 {
		viewh = 1
	}
	if total == 0 {
		note := m.tr.LogsEmpty
		if m.logs.filter.value != "" {
			note = m.tr.NotFound
		}
		empty := sethint.Width(usable).Align(lipgloss.Center).Render(note)
		return lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(
			lipgloss.JoinVertical(lipgloss.Left, head, empty))
	}

	scroll := clampint(m.logs.scroll, 0, max0(total-viewh))
	end := scroll + viewh
	if end > total {
		end = total
	}
	vis := append([]string(nil), body[scroll:end]...)
	fadelist(vis, scroll > 0, end < total)

	panel := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(
		lipgloss.JoinVertical(lipgloss.Left, head, strings.Join(vis, "\n")))

	if total > viewh {
		bar := strings.Split(scrollbarcol(viewh, total, scroll), "\n")
		fadelist(bar, scroll > 0, end < total)
		panel = placeoverlay(width-2, logtop, strings.Join(bar, "\n"), panel)
	}
	return panel
}

func (m Menu) logviewh() int {
	_, py, _ := m.panelgeom()
	h := m.height - py - logtop
	if h < 1 {
		h = 1
	}
	return h
}

func (m Menu) logtotal() int {
	return len(m.logs.body(panelusable(m.panelwidth())))
}

func (m *Menu) clamplogs() {
	m.logs.scroll = clampint(m.logs.scroll, 0, max0(m.logtotal()-m.logviewh()))
}

func (m *Menu) tailfollow() {
	if m.logs.follow {
		m.logs.scroll = max0(m.logtotal() - m.logviewh())
	}
	m.clamplogs()
}

func (m Menu) logfilterwidth() int {
	return max0(panelusable(m.panelwidth()) - 4)
}

func (m Menu) updatelogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.logs.filteron {
		switch msg.String() {
		case "esc":
			m.logs.filteron = false
			m.logs.filter = textinput{}
			m.tailfollow()
			return m.withtick(nil)
		case "enter", "down", "tab":
			m.logs.filteron = false
			m.tailfollow()
			return m.withtick(nil)
		case "up", "shift+tab":
			m.logs.filteron = false
			m.focus = focusburger
			return m.withtick(nil)
		case "left":
			if m.logs.filter.cursorpos == 0 {
				m.logs.filteron = false
				m.focus = focusconnect
				return m.withtick(nil)
			}
		}
		if m.logs.filter.handlekey(msg, m.logfilterwidth()) {
			m.logs.scroll = 0
			m.tailfollow()
		}
		return m.withtick(nil)
	}
	switch msg.String() {
	case "esc", "left", "h":
		if m.logs.filter.value != "" {
			m.logs.filter = textinput{}
			m.tailfollow()
			return m.withtick(nil)
		}
		m.focus = focusconnect
		return m.withtick(nil)
	case "/":
		m.logs.filteron = true
		m.logs.filter.focusend()
		return m.withtick(nil)
	case "up", "k":
		if m.logs.scroll == 0 {
			m.logs.filteron = true
			m.logs.filter.focusend()
			return m.withtick(nil)
		}
		return m.logscrollby(-1)
	case "down", "j":
		return m.logscrollby(1)
	case "pgup", "ctrl+b":
		return m.logscrollby(-m.logviewh())
	case "pgdown", "ctrl+f":
		return m.logscrollby(m.logviewh())
	case "ctrl+u":
		return m.logscrollby(-max0(m.logviewh() / 2))
	case "ctrl+d":
		return m.logscrollby(max0(m.logviewh() / 2))
	case "home", "g":
		m.logs.follow = false
		m.logs.scroll = 0
		return m.withtick(nil)
	case "ctrl+e", "end", "G":
		m.logs.follow = true
		m.tailfollow()
		return m.withtick(nil)
	case "c":
		m.logs.lines = nil
		m.logs.scroll = 0
		m.logs.follow = true
		return m.withtick(nil)
	}
	return m, nil
}

func (m Menu) logscrollby(delta int) (tea.Model, tea.Cmd) {
	if delta < 0 {
		m.logs.follow = false
	}
	m.logs.scroll += delta
	m.clamplogs()
	if m.logs.scroll >= max0(m.logtotal()-m.logviewh()) {
		m.logs.follow = true
	}
	return m.withtick(nil)
}

func (m Menu) logwheel(dir int) (tea.Model, tea.Cmd) {
	return m.logscrollby(dir * 3)
}

func (m Menu) clicklogs(x, y int) (tea.Model, tea.Cmd) {
	px, py, pw := m.panelgeom()
	if x < px+2 || x >= px+pw {
		return m, nil
	}
	m.focus = focuslogs
	if local := y - py; local >= 3 && local < 6 {
		m.logs.filteron = true
		m.logs.filter.clickto(local-4, x-px-3, m.logfilterwidth())
		return m.withtick(nil)
	}
	m.logs.filteron = false
	return m.withtick(nil)
}
