package tui

import (
	"errors"
	"net"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/alexandria-proxy/alexandria-cli/internal/autostart"
	"github.com/alexandria-proxy/alexandria-cli/internal/config"
	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
)

type setscreen int

const (
	screenroot setscreen = iota
	screensubs
	screenadv
	screenroutes
	screenlogs
	screenreset
)

type rowkind int

const (
	rowsection rowkind = iota
	rowtoggle
	rowselect
	rowtext
	rownumber
	rowslider
	rowradio
	rowlink
	rowvalue
	rowentry
)

type setrow struct {
	key    string
	kind   rowkind
	label  string
	note   string
	value  string
	holder string
	on     bool
	opts   []string
	codes  []string
	idx    int
	lo     int
	hi     int
	unit   string
	target setscreen
	path   string
	danger bool
	tight  bool
	entry  int
}

func (r setrow) focusable() bool { return r.kind != rowsection }

type autostartmsg struct {
	on   bool
	err  string
	root bool
}

type resetmsg struct {
	kind string
	err  string
}

func setautostartcmd(on bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if on {
			err = autostart.Enable()
		} else {
			err = autostart.Disable()
		}
		if err != nil {
			return autostartmsg{on: !on, err: err.Error(), root: errors.Is(err, autostart.ErrNeedsRoot)}
		}
		return autostartmsg{on: on}
	}
}

func savecfgcmd(c config.Config) tea.Cmd {
	return func() tea.Msg {
		_ = config.Save(c)
		return nil
	}
}

func pickindex(codes []string, want string) int {
	for i, c := range codes {
		if c == want {
			return i
		}
	}
	return 0
}

func langlabels() []string {
	out := make([]string, len(languages))
	for i, l := range languages {
		out[i] = l.flag + " " + l.label
	}
	return out
}

func langcodes() []string {
	out := make([]string, len(languages))
	for i, l := range languages {
		out[i] = l.code
	}
	return out
}

func localip() string {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return "—"
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return "—"
}

func (m Menu) rootrows() []setrow {
	s := m.cfg.Settings
	rows := []setrow{
		{kind: rowsection, label: m.tr.SecGeneral},
		{key: "lang", kind: rowselect, label: m.tr.LblLanguage,
			opts: langlabels(), codes: langcodes(), idx: pickindex(langcodes(), m.cfg.Lang)},
		{key: "pingproto", kind: rowselect, label: m.tr.LblPingProto,
			opts: []string{m.tr.OptTCP, m.tr.OptICMP}, codes: []string{"tcp", "icmp"},
			idx: pickindex([]string{"tcp", "icmp"}, s.PingProto)},
		{key: "preferip", kind: rowselect, label: m.tr.LblPreferIP,
			opts: []string{m.tr.OptAuto, m.tr.OptIPv4, m.tr.OptIPv6}, codes: []string{"auto", "ipv4", "ipv6"},
			idx: pickindex([]string{"auto", "ipv4", "ipv6"}, s.PreferIP)},

		{kind: rowsection, label: m.tr.SecConnection},
		{key: "frag", kind: rowtoggle, label: m.tr.LblFrag, on: s.Fragment.On, note: m.tr.NoteFrag},
	}
	if s.Fragment.On {
		packets := []string{"tlshello", "1-1", "1-3"}
		rows = append(rows,
			setrow{key: "frag.packets", kind: rowselect, label: "  " + m.tr.LblFragPackets,
				opts: packets, codes: packets, idx: pickindex(packets, s.Fragment.Packets)},
			setrow{key: "frag.length", kind: rowtext, label: "  " + m.tr.LblFragLength, value: s.Fragment.Length},
			setrow{key: "frag.interval", kind: rowtext, label: "  " + m.tr.LblFragInterval, value: s.Fragment.Interval},
		)
	}
	rows = append(rows, setrow{key: "mux", kind: rowtoggle, label: m.tr.LblMux, on: s.Mux.On, note: m.tr.NoteMux})
	if s.Mux.On {
		rows = append(rows, setrow{key: "mux.conc", kind: rownumber, label: "  " + m.tr.LblMuxConc,
			value: strconv.Itoa(s.Mux.Concurrency), lo: 1, hi: 1024})
	}
	rows = append(rows,
		setrow{key: "lan", kind: rowtoggle, label: m.tr.LblLAN, on: s.LAN, note: m.tr.NoteLAN},
		setrow{key: "currentip", kind: rowvalue, label: m.tr.LblCurrentIP, value: m.localip, tight: true},
		setrow{key: "socks", kind: rowtext, label: m.tr.LblSocks, value: strconv.Itoa(s.SocksPort)},

		setrow{kind: rowsection, label: m.tr.SecOther},
		setrow{key: "nav.subs", kind: rowlink, label: m.tr.SettingsSubs, target: screensubs, tight: true},
		setrow{key: "nav.adv", kind: rowlink, label: m.tr.SettingsAdvanced, target: screenadv, tight: true},
		setrow{key: "nav.logs", kind: rowlink, label: m.tr.LogsTitle, target: screenlogs, tight: true},
		setrow{key: "nav.reset", kind: rowlink, label: m.tr.SettingsReset, target: screenreset, danger: true},
	)
	return rows
}

func (m Menu) subsrows() []setrow {
	s := m.cfg.Settings.Subs
	sorts := []string{"none", "ping", "alpha"}
	return []setrow{
		{kind: rowsection, label: m.tr.SecUpdate},
		{key: "subs.auto", kind: rowtoggle, label: m.tr.LblAutoUpdate, on: s.Auto},
		{key: "subs.interval", kind: rownumber, label: m.tr.LblUpdateEvery, value: strconv.Itoa(s.IntervalH), lo: 1, hi: 168},
		{key: "subs.timeout", kind: rowslider, label: m.tr.LblTimeout, idx: s.TimeoutSec, lo: 1, hi: 60,
			unit: m.tr.Seconds, note: m.tr.NoteUpdate},

		{key: "subs.updateopen", kind: rowtoggle, label: m.tr.LblUpdateOpen, on: s.UpdateOpen, tight: true},
		{key: "subs.pingopen", kind: rowtoggle, label: m.tr.LblPingOpen, on: s.PingOpen, tight: true},
		{key: "subs.connectopen", kind: rowtoggle, label: m.tr.LblConnectOpen, on: s.ConnectOpen, note: m.tr.NoteOnOpen},

		{key: "subs.nodupes", kind: rowtoggle, label: m.tr.LblNoDupes, on: s.NoDupes, note: m.tr.NoteNoDupes},

		{kind: rowsection, label: m.tr.SecSending},
		{key: "subs.hwid", kind: rowtoggle, label: m.tr.LblSendHWID, on: s.SendHWID, note: m.tr.NoteHWID},

		{kind: rowsection, label: m.tr.SecUserAgent},
		{key: "subs.ua", kind: rowtext, label: "", value: s.UserAgent, holder: subscription.Useragent(), note: m.tr.NoteUserAgent},

		{kind: rowsection, label: m.tr.SecServerList},
		{key: "subs.sort", kind: rowradio, label: m.tr.OptUnsorted, idx: 0, on: pickindex(sorts, s.SortBy) == 0, codes: sorts},
		{key: "subs.sort", kind: rowradio, label: m.tr.OptSortPing, idx: 1, on: pickindex(sorts, s.SortBy) == 1, codes: sorts},
		{key: "subs.sort", kind: rowradio, label: m.tr.OptSortAlpha, idx: 2, on: pickindex(sorts, s.SortBy) == 2, codes: sorts},
	}
}

func (m Menu) advrows() []setrow {
	a := m.cfg.Settings.Advanced
	stacks := []string{"mixed", "system", "gvisor"}
	confs := []string{"default", "custom"}
	rows := []setrow{
		{kind: rowsection, label: m.tr.SecOtherSettings},
		{key: "autostart", kind: rowtoggle, label: m.tr.AutostartLabel, on: m.autostart, note: m.tr.AutostartHint},
		{key: "adv.localdns", kind: rowtoggle, label: m.tr.LblLocalDNS, on: a.LocalDNS},
		{key: "adv.jsondns", kind: rowtoggle, label: m.tr.LblJSONDNS, on: a.JSONDNS},
		{key: "adv.resolve", kind: rowtoggle, label: m.tr.LblResolveSrv, on: a.ResolveSrv},
		{key: "adv.sniff", kind: rowtoggle, label: m.tr.LblSniffing, on: a.Sniffing},
		{key: "adv.sysproxy", kind: rowtoggle, label: m.tr.LblSysProxy, on: a.SysProxy},
		{key: "adv.tun", kind: rowtoggle, label: m.tr.LblTUN, on: a.TUN},
	}
	if a.TUN {
		rows = append(rows,
			setrow{key: "adv.tunprovider", kind: rowselect, label: m.tr.LblTunProvider,
				opts: []string{m.tr.OptSingbox}, codes: []string{"singbox"}, idx: 0},
			setrow{key: "adv.tunmode", kind: rowselect, label: m.tr.LblTunMode,
				opts: []string{m.tr.OptMixed, m.tr.OptSystem, m.tr.OptGvisor}, codes: stacks,
				idx: pickindex(stacks, a.TunStack)},
			setrow{key: "adv.tunconfig", kind: rowselect, label: m.tr.LblTunConfig,
				opts: []string{m.tr.OptDefault, m.tr.OptCustom}, codes: confs,
				idx: pickindex(confs, a.TunConfig)},
			setrow{key: "adv.tundnson", kind: rowtoggle, label: m.tr.LblTunDNSOn, on: a.TunDNSOn},
			setrow{key: "adv.tundns", kind: rowtext, label: m.tr.LblTunDNS, value: a.TunDNS},
		)
	}
	rows = append(rows, setrow{key: "nav.routes", kind: rowlink, label: m.tr.SettingsRoutes, target: screenroutes})
	return rows
}

func (m Menu) routerows() []setrow {
	rows := []setrow{
		{kind: rowsection, label: m.tr.SecCIDR},
		{key: "routes.add", kind: rowtext, label: "", value: "", holder: m.tr.AddCIDR, note: m.tr.NoteCIDR},
	}
	for i, cidr := range m.cfg.Settings.Advanced.Excluded {
		rows = append(rows, setrow{key: "routes.entry", kind: rowentry, label: cidr, entry: i})
	}
	return rows
}

func (m Menu) logsrows() []setrow {
	l := m.cfg.Settings.Logs
	rows := []setrow{
		{kind: rowsection, label: m.tr.SecLogging},
		{key: "logs.on", kind: rowtoggle, label: m.tr.LblLogOn, on: l.On, note: m.tr.NoteLogOn},
	}
	if !l.On {
		return rows
	}
	return append(rows,
		setrow{kind: rowsection, label: m.tr.SecSources},
		setrow{key: "logs.daemon", kind: rowtoggle, label: m.tr.LblLogDaemon, on: l.Daemon, tight: true},
		setrow{key: "logs.xray", kind: rowtoggle, label: m.tr.LblLogXray, on: l.Xray, tight: true},
		setrow{key: "logs.singbox", kind: rowtoggle, label: m.tr.LblLogSingbox, on: l.Singbox, note: m.tr.NoteLogSrc},
		setrow{kind: rowsection, label: m.tr.SecStorage},
		setrow{key: "logs.max", kind: rowtext, label: m.tr.LblLogMax,
			value: formatsize(l.Max), note: m.tr.NoteLogMax},
	)
}

func (m Menu) resetrows() []setrow {
	return []setrow{
		{key: "reset.user", kind: rowlink, label: m.tr.ResetUser, note: m.tr.NoteResetUser, danger: true},
		{key: "reset.prefs", kind: rowlink, label: m.tr.ResetPrefs, note: m.tr.NoteResetPrefs, danger: true},
		{key: "reset.tun", kind: rowlink, label: m.tr.ResetTun, note: m.tr.NoteResetTun, danger: true},
	}
}

func (m Menu) screenrows(sc setscreen) []setrow {
	switch sc {
	case screensubs:
		return m.subsrows()
	case screenadv:
		return m.advrows()
	case screenroutes:
		return m.routerows()
	case screenlogs:
		return m.logsrows()
	case screenreset:
		return m.resetrows()
	}
	return m.rootrows()
}

func (m Menu) searchresults(q string) []setrow {
	lower := strings.ToLower(strings.TrimSpace(q))
	if lower == "" {
		return nil
	}
	screens := []struct {
		sc   setscreen
		path string
	}{
		{screenroot, ""},
		{screensubs, m.tr.SettingsSubs},
		{screenadv, m.tr.SettingsAdvanced},
		{screenroutes, m.tr.SettingsRoutes},
		{screenlogs, m.tr.LogsTitle},
		{screenreset, m.tr.SettingsReset},
	}
	var out []setrow
	for _, s := range screens {
		var group []setrow
		for _, r := range m.screenrows(s.sc) {
			if !r.focusable() || r.key == "routes.add" || r.key == "routes.entry" {
				continue
			}
			if !substrmatch(r.label, lower).ok && !substrmatch(r.note, lower).ok {
				continue
			}
			group = append(group, r)
		}
		if len(group) == 0 {
			continue
		}
		if s.path != "" {
			out = append(out, setrow{kind: rowsection, label: s.path})
		}
		out = append(out, group...)
	}
	return out
}

func (m Menu) currentrows() []setrow {
	if q := strings.TrimSpace(m.setsearch.value); q != "" {
		return m.searchresults(q)
	}
	return m.screenrows(m.setscr)
}

func (m Menu) rowview(r setrow, usable int, focused bool) string {
	inner := max0(usable - 2)
	if r.kind == rowsection {
		return setsection.Render(r.label)
	}

	mark, st := "  ", setlabel
	if focused {
		mark, st = "▸ ", setlabelsel
	}
	if r.danger {
		st = setdanger
		if focused {
			st = setdangersel
		}
	}

	var head string
	switch r.kind {
	case rowradio:
		head = st.Render(mark) + radiomark(r.on, focused) + " " + st.Render(r.label)
	default:
		head = st.Render(mark + r.label)
	}
	cw := chipwidth(inner)
	var line string
	switch r.kind {
	case rowtoggle:
		line = spread(head, toggleview(r.on, focused), inner)
	case rowlink:
		line = spread(head, setarrowst.Render("›"), inner)
	case rowentry:
		line = spread(head, setarrowst.Render("✕"), inner)
	case rowvalue:
		line = spread(head, setvaluest.Render(cliprunes(r.value, cw)), inner)
	case rownumber:
		line = spread(head, stepperchip(r.value, focused, cw), inner)
	case rowslider:
		line = spread(head, sliderview(r.idx, r.lo, r.hi, focused)+"  "+setvaluest.Render(slidervalue(r)), inner)
	case rowradio:
		line = head
	case rowselect:
		value := ""
		if r.idx >= 0 && r.idx < len(r.opts) {
			value = r.opts[r.idx]
		}
		line = spread(head, selectchip(value, focused, cw), inner)
	case rowtext:
		boxw := cw
		if r.label == "" {
			boxw = max0(inner - 6)
		}
		value, view := r.value, ""
		editing := focused && m.setinputkey == r.key
		if editing {
			value = m.setinput.value
		}
		switch {
		case value == "" && r.holder != "":
			view = sethint.Render(cliprunes(r.holder, boxw))
		case editing:
			view = m.setinput.view(boxw, true, btngray)
		case value != "":
			view = textinput{value: value}.view(boxw, false, lipgloss.Color("252"))
		}
		box := textbox(view, focused, value != "", boxw)
		if r.label == "" {
			line = lipgloss.NewStyle().PaddingLeft(2).Render(box)
		} else {
			line = spread(head, box, inner)
		}
	default:
		line = head
	}

	if r.note == "" {
		return line
	}
	return lipgloss.JoinVertical(lipgloss.Left, line, notetext(r.note, inner))
}

func slidervalue(r setrow) string {
	v := strconv.Itoa(r.idx)
	for len(v) < len(strconv.Itoa(r.hi)) {
		v = " " + v
	}
	if r.unit != "" {
		v += " " + r.unit
	}
	return v
}

func notetext(note string, usable int) string {
	return sethint.Width(usable).PaddingLeft(2).Render(note)
}

func tightpair(rows []setrow, i int) bool {
	if i+1 >= len(rows) {
		return false
	}
	cur, next := rows[i], rows[i+1]
	if cur.note != "" || next.kind == rowsection {
		return false
	}
	if cur.tight {
		return true
	}
	if cur.kind != next.kind {
		return false
	}
	return cur.kind == rowradio || cur.kind == rowentry
}

func (m Menu) setviews(rows []setrow, usable int) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		view := m.rowview(r, usable, m.focus == focussettings && i == m.setidx)
		if r.focusable() && !tightpair(rows, i) {
			view += "\n"
		}
		out[i] = view
	}
	return out
}

func (m Menu) setheadh() int {
	if m.setscr == screenroot {
		return 1
	}
	return 2
}

func (m Menu) settop() int {
	return 1 + m.setheadh() + 3
}

func (m Menu) setsearchview(usable int) string {
	cw := max0(usable - 2)
	if cw < 1 {
		cw = 1
	}
	border := paneldim
	var text string
	switch {
	case m.setsearchon:
		border = btngray
		text = m.setsearch.view(cw, true, btngray)
	case m.setsearch.value != "":
		text = m.setsearch.view(cw, false, lipgloss.Color("252"))
	default:
		text = sethint.Render(cliprunes(m.tr.SetSearchHint, cw))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(cw).
		Render(text)
}

func (m Menu) rendersettings(width, height int) string {
	if width < 8 {
		return ""
	}
	usable := width - 4
	if usable < 16 {
		usable = width
	}

	rows := m.currentrows()
	views := m.setviews(rows, usable)

	body := strings.Split(lipgloss.JoinVertical(lipgloss.Left, views...), "\n")
	if len(views) == 0 {
		body = []string{notfoundst.Width(usable).Align(lipgloss.Center).Render(m.tr.NotFound)}
	}

	total := len(body)
	viewh := height - m.settop()
	if viewh < 1 {
		viewh = 1
	}
	scroll := clampint(m.setscroll, 0, max0(total-viewh))
	end := scroll + viewh
	if end > total {
		end = total
	}
	vis := append([]string(nil), body[scroll:end]...)
	fadelist(vis, scroll > 0, end < total)

	body2 := lipgloss.NewStyle().PaddingLeft(2).Render(lipgloss.JoinVertical(lipgloss.Left, m.setsearchview(usable), strings.Join(vis, "\n")))
	head := m.crumbline(m.setscr, max0(usable-4))
	panel := lipgloss.NewStyle().PaddingTop(1).Render(lipgloss.JoinVertical(lipgloss.Left, head, body2))

	if total > viewh {
		bar := strings.Split(scrollbarcol(viewh, total, scroll), "\n")
		fadelist(bar, scroll > 0, end < total)
		panel = placeoverlay(width-2, m.settop(), strings.Join(bar, "\n"), panel)
	}
	if drop, dx, dy := m.dropdown(rows, views, usable, scroll); drop != "" {
		top := m.settop() + dy
		panel = placeoverlay(dx, top, drop, panel)
		panel = dropshadow(panel, dx, top+lipgloss.Height(drop), lipgloss.Width(drop))
	}
	return panel
}

var shadowsteps = []float64{0.34, 0.62, 0.84}

func dropshadow(panel string, x, y, w int) string {
	lines := strings.Split(panel, "\n")
	for i, f := range shadowsteps {
		row := y + i
		if row < 0 || row >= len(lines) {
			break
		}
		seg := ansi.Cut(lines[row], x, x+w)
		if seg == "" {
			continue
		}
		panel = placeoverlay(x, row, dimline(seg, f), panel)
	}
	return panel
}

func (m Menu) dropdown(rows []setrow, views []string, usable, scroll int) (string, int, int) {
	if !m.setopen || m.setidx < 0 || m.setidx >= len(rows) {
		return "", 0, 0
	}
	r := rows[m.setidx]
	if r.kind != rowselect || len(r.opts) == 0 {
		return "", 0, 0
	}
	at := 0
	for i := 0; i < m.setidx; i++ {
		at += lipgloss.Height(views[i])
	}
	n := len(r.opts)
	top := clampint(at-scroll-r.idx, 0, max0(m.setviewh()-n))
	cw := chipwidth(max0(usable - 2))
	return optionlist(r.opts, m.setoptcur, cw), 2 + max0(usable-2) - cw, top
}

func (m Menu) setlistheight() int {
	rows := m.currentrows()
	usable := panelusable(m.panelwidth())
	return lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, m.setviews(rows, usable)...))
}

func (m Menu) setviewh() int {
	_, py, _ := m.panelgeom()
	h := m.height - py - m.settop()
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Menu) clampsetscroll() {
	m.setscroll = clampint(m.setscroll, 0, max0(m.setlistheight()-m.setviewh()))
}

func (m *Menu) ensuresetvisible() {
	rows := m.currentrows()
	if m.setidx < 0 || m.setidx >= len(rows) {
		return
	}
	usable := panelusable(m.panelwidth())
	views := m.setviews(rows, usable)
	start := 0
	for i := 0; i < m.setidx; i++ {
		start += lipgloss.Height(views[i])
	}
	h := lipgloss.Height(views[m.setidx])
	viewh := m.setviewh()
	if start < m.setscroll {
		m.setscroll = start
	}
	if start+h > m.setscroll+viewh {
		m.setscroll = start + h - viewh
	}
	m.clampsetscroll()
}

func (m Menu) panelwidth() int {
	_, _, pw := m.panelgeom()
	return pw
}

func (m *Menu) movefocus(dir int) tea.Cmd {
	rows := m.currentrows()
	if len(rows) == 0 {
		return nil
	}
	cmd := m.commitinput(rows)
	m.setopen = false
	i := m.setidx
	for {
		i += dir
		if i < 0 || i >= len(rows) {
			return cmd
		}
		if rows[i].focusable() {
			break
		}
	}
	m.setidx = i
	m.loadinput(rows[i])
	m.ensuresetvisible()
	return cmd
}

func (m *Menu) firstfocus() {
	rows := m.currentrows()
	m.setidx = 0
	for i, r := range rows {
		if r.focusable() {
			m.setidx = i
			m.loadinput(r)
			break
		}
	}
	m.setscroll = 0
}

func (m *Menu) remember() {
	if int(m.setscr) < len(m.setpos) && strings.TrimSpace(m.setsearch.value) == "" {
		m.setpos[m.setscr] = m.setidx
	}
}

func (m *Menu) restorefocus() {
	rows := m.currentrows()
	want := 0
	if int(m.setscr) < len(m.setpos) {
		want = m.setpos[m.setscr]
	}
	if want <= 0 || want >= len(rows) || !rows[want].focusable() {
		m.firstfocus()
		return
	}
	m.setidx = want
	m.loadinput(rows[want])
	m.setscroll = 0
	m.ensuresetvisible()
}

func (m *Menu) gotoscreen(sc setscreen) {
	m.remember()
	m.setscr = sc
	m.setsearch = textinput{}
	m.setsearchon = false
	m.setopen = false
	m.restorefocus()
}

func (m *Menu) loadinput(r setrow) {
	if r.kind != rowtext {
		m.setinputkey = ""
		return
	}
	m.setinputkey = r.key
	m.setinput = textinput{value: r.value}
	m.setinput.focusend()
}

func (m *Menu) commitinput(rows []setrow) tea.Cmd {
	if m.setinputkey == "" || m.setinputkey == "routes.add" {
		return nil
	}
	key := m.setinputkey
	value := strings.TrimSpace(m.setinput.value)
	m.setinputkey = ""

	s := &m.cfg.Settings
	switch key {
	case "frag.length":
		s.Fragment.Length = value
	case "frag.interval":
		s.Fragment.Interval = value
	case "adv.tundns":
		s.Advanced.TunDNS = value
	case "subs.ua":
		s.Subs.UserAgent = value
	case "logs.max":
		n, ok := parsesize(value)
		if !ok {
			m.pushtoast(toasterr, m.tr.ErrBadSize)
			return nil
		}
		s.Logs.Max = n
	case "socks":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 65534 {
			m.pushtoast(toasterr, m.tr.ErrBadPort)
			return nil
		}
		s.SocksPort = n
	default:
		return nil
	}
	m.cfg = config.Normalize(m.cfg)
	return savecfgcmd(m.cfg)
}

func (m *Menu) applylang(code string) tea.Cmd {
	m.cfg.Lang = code
	m.tr = i18n.T(code)
	m.panel.tr = m.tr
	m.form.tr = m.tr
	m.info.tr = m.tr
	return savecfgcmd(m.cfg)
}

func (m *Menu) pickoption(r setrow, i int) tea.Cmd {
	if i < 0 || i >= len(r.codes) {
		return nil
	}
	code := r.codes[i]
	s := &m.cfg.Settings
	switch r.key {
	case "lang":
		return m.applylang(code)
	case "pingproto":
		s.PingProto = code
	case "preferip":
		s.PreferIP = code
	case "frag.packets":
		s.Fragment.Packets = code
	case "adv.tunmode":
		s.Advanced.TunStack = code
	case "adv.tunconfig":
		s.Advanced.TunConfig = code
	case "adv.tunprovider":
		s.Advanced.TunProvider = code
	case "subs.sort":
		s.Subs.SortBy = code
		m.panel.sortby = code
		m.panel.refresh()
	default:
		return nil
	}
	m.cfg = config.Normalize(m.cfg)
	return savecfgcmd(m.cfg)
}

func (m *Menu) toggle(key string) tea.Cmd {
	s := &m.cfg.Settings
	switch key {
	case "autostart":
		m.autostart = !m.autostart
		return setautostartcmd(m.autostart)
	case "frag":
		s.Fragment.On = !s.Fragment.On
	case "mux":
		s.Mux.On = !s.Mux.On
	case "lan":
		s.LAN = !s.LAN
	case "subs.auto":
		s.Subs.Auto = !s.Subs.Auto
	case "subs.updateopen":
		s.Subs.UpdateOpen = !s.Subs.UpdateOpen
	case "subs.pingopen":
		s.Subs.PingOpen = !s.Subs.PingOpen
	case "subs.connectopen":
		s.Subs.ConnectOpen = !s.Subs.ConnectOpen
	case "subs.nodupes":
		s.Subs.NoDupes = !s.Subs.NoDupes
		m.panel.nodupes = s.Subs.NoDupes
		m.panel.refresh()
	case "subs.hwid":
		s.Subs.SendHWID = !s.Subs.SendHWID
	case "adv.localdns":
		s.Advanced.LocalDNS = !s.Advanced.LocalDNS
	case "adv.jsondns":
		s.Advanced.JSONDNS = !s.Advanced.JSONDNS
	case "adv.resolve":
		s.Advanced.ResolveSrv = !s.Advanced.ResolveSrv
	case "adv.sniff":
		s.Advanced.Sniffing = !s.Advanced.Sniffing
	case "adv.sysproxy":
		s.Advanced.SysProxy = !s.Advanced.SysProxy
	case "adv.tun":
		s.Advanced.TUN = !s.Advanced.TUN
	case "adv.tundnson":
		s.Advanced.TunDNSOn = !s.Advanced.TunDNSOn
	case "logs.on":
		s.Logs.On = !s.Logs.On
	case "logs.daemon":
		s.Logs.Daemon = !s.Logs.Daemon
	case "logs.xray":
		s.Logs.Xray = !s.Logs.Xray
	case "logs.singbox":
		s.Logs.Singbox = !s.Logs.Singbox
	default:
		return nil
	}
	m.cfg = config.Normalize(m.cfg)
	return savecfgcmd(m.cfg)
}

func (m *Menu) bump(r setrow, delta int) tea.Cmd {
	s := &m.cfg.Settings
	switch r.key {
	case "mux.conc":
		s.Mux.Concurrency = clampint(s.Mux.Concurrency+delta, r.lo, r.hi)
	case "subs.interval":
		s.Subs.IntervalH = clampint(s.Subs.IntervalH+delta, r.lo, r.hi)
	case "subs.timeout":
		s.Subs.TimeoutSec = clampint(s.Subs.TimeoutSec+delta, r.lo, r.hi)
	default:
		return nil
	}
	m.cfg = config.Normalize(m.cfg)
	return savecfgcmd(m.cfg)
}

func (m *Menu) setslider(r setrow, v int) tea.Cmd {
	if r.key != "subs.timeout" {
		return nil
	}
	m.cfg.Settings.Subs.TimeoutSec = clampint(v, r.lo, r.hi)
	return savecfgcmd(m.cfg)
}

func (m *Menu) addroute() tea.Cmd {
	value := strings.TrimSpace(m.setinput.value)
	if value == "" {
		return nil
	}
	if !validroute(value) {
		m.pushtoast(toasterr, m.tr.ErrBadCIDR)
		return nil
	}
	m.cfg.Settings.Advanced.Excluded = append(m.cfg.Settings.Advanced.Excluded, value)
	m.setinput = textinput{}
	return savecfgcmd(m.cfg)
}

func validroute(v string) bool {
	if _, _, err := net.ParseCIDR(v); err == nil {
		return true
	}
	if net.ParseIP(v) != nil {
		return true
	}
	return strings.Contains(v, ".") && !strings.ContainsAny(v, " /\\")
}

func (m *Menu) droproute(i int) tea.Cmd {
	ex := m.cfg.Settings.Advanced.Excluded
	if i < 0 || i >= len(ex) {
		return nil
	}
	m.cfg.Settings.Advanced.Excluded = append(append([]string(nil), ex[:i]...), ex[i+1:]...)
	return savecfgcmd(m.cfg)
}

func (m Menu) activate(r setrow) (tea.Model, tea.Cmd) {
	switch r.kind {
	case rowtoggle:
		cmd := m.toggle(r.key)
		m.clampsetscroll()
		return m.withtick(cmd)
	case rowradio:
		return m.withtick(m.pickoption(r, r.idx))
	case rowselect:
		if strings.HasPrefix(r.key, "adv.tunprovider") && len(r.opts) < 2 {
			return m, nil
		}
		m.setopen = !m.setopen
		m.setoptcur = r.idx
		m.ensuresetvisible()
		return m.withtick(nil)
	case rowlink:
		if strings.HasPrefix(r.key, "reset.") {
			m.setconfirm = r.key
			return m.withtick(nil)
		}
		m.gotoscreen(r.target)
		return m.withtick(nil)
	case rowentry:
		cmd := m.droproute(r.entry)
		m.firstfocus()
		return m.withtick(cmd)
	case rowtext:
		if r.key == "routes.add" {
			return m.withtick(m.addroute())
		}
		if r.key == "adv.tundns" || r.key == "socks" || r.key == "subs.ua" || r.key == "logs.max" {
			return m.withtick(nil)
		}
	}
	return m, nil
}

func (m Menu) setback() (tea.Model, tea.Cmd) {
	cmd := m.commitinput(m.currentrows())
	if m.setopen {
		m.setopen = false
		return m.withtick(cmd)
	}
	if m.setsearch.value != "" {
		m.setsearch = textinput{}
		m.restorefocus()
		return m.withtick(cmd)
	}
	switch m.setscr {
	case screenroutes:
		m.gotoscreen(screenadv)
		return m.withtick(cmd)
	case screensubs, screenadv, screenlogs, screenreset:
		m.gotoscreen(screenroot)
		return m.withtick(cmd)
	}
	m.remember()
	m.focus = focusconnect
	return m.withtick(cmd)
}

func (m Menu) updateconfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "n", "left", "h":
		m.setconfirm = ""
		return m.withtick(nil)
	case "enter", " ", "y":
		kind := strings.TrimPrefix(m.setconfirm, "reset.")
		if kind == "prefs" {
			kind = "settings"
		}
		m.setconfirm = ""
		return m.withtick(resetcmd(kind))
	}
	return m, nil
}

func (m Menu) updatesettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.setconfirm != "" {
		return m.updateconfirm(msg)
	}
	rows := m.currentrows()
	var cur setrow
	if m.setidx >= 0 && m.setidx < len(rows) {
		cur = rows[m.setidx]
	}

	if m.setopen && cur.kind == rowselect {
		n := len(cur.opts)
		switch msg.String() {
		case "up", "k":
			m.setoptcur = (m.setoptcur - 1 + n) % n
		case "down", "j":
			m.setoptcur = (m.setoptcur + 1) % n
		case "enter", " ":
			cmd := m.pickoption(cur, m.setoptcur)
			m.setopen = false
			return m.withtick(cmd)
		case "esc", "left", "h":
			m.setopen = false
		}
		return m.withtick(nil)
	}

	if m.setsearchon {
		switch msg.String() {
		case "esc":
			m.setsearchon = false
			m.setsearch = textinput{}
			m.firstfocus()
			return m.withtick(nil)
		case "enter", "down", "tab":
			m.setsearchon = false
			m.firstfocus()
			return m.withtick(nil)
		case "up", "shift+tab":
			m.setsearchon = false
			m.focus = focusburger
			return m.withtick(nil)
		case "left":
			if m.setsearch.cursorpos == 0 {
				m.setsearchon = false
				m.remember()
				m.focus = focusconnect
				return m.withtick(nil)
			}
		}
		if m.setsearch.handlekey(msg, m.setsearchwidth()) {
			m.setscroll = 0
			m.setidx = 0
			m.firstfocus()
			return m.withtick(nil)
		}
		return m.withtick(nil)
	}

	if cur.kind == rowtext && m.setinputkey == cur.key {
		switch msg.String() {
		case "enter":
			return m.activate(cur)
		case "up", "shift+tab":
			return m.withtick(m.movefocus(-1))
		case "down", "tab":
			return m.withtick(m.movefocus(1))
		case "esc":
			return m.setback()
		}
		if m.setinput.handlekey(msg, m.textwidth(cur)) {
			return m.withtick(nil)
		}
	}

	switch msg.String() {
	case "esc", "backspace":
		return m.setback()
	case "up", "k", "shift+tab":
		if m.firstfocusable(rows) == m.setidx {
			cmd := m.commitinput(rows)
			m.setsearchon = true
			m.setsearch.focusend()
			return m.withtick(cmd)
		}
		return m.withtick(m.movefocus(-1))
	case "down", "j", "tab":
		return m.withtick(m.movefocus(1))
	case "/":
		cmd := m.commitinput(rows)
		m.setsearchon = true
		m.setsearch.focusend()
		return m.withtick(cmd)
	case "left", "h":
		if cur.kind == rownumber || cur.kind == rowslider {
			return m.withtick(m.bump(cur, -1))
		}
		return m.setback()
	case "right", "l":
		if cur.kind == rownumber || cur.kind == rowslider {
			return m.withtick(m.bump(cur, 1))
		}
		return m.activate(cur)
	case "enter", " ":
		return m.activate(cur)
	case "delete":
		if cur.kind == rowentry {
			cmd := m.droproute(cur.entry)
			m.firstfocus()
			return m.withtick(cmd)
		}
	case "ctrl+y":
		if cur.key == "socks" {
			m.pushtoast(toastok, m.tr.Copied)
			return m.withtick(osc52copy(strconv.Itoa(m.cfg.Settings.SocksPort)))
		}
	}
	return m, nil
}

func (m Menu) firstfocusable(rows []setrow) int {
	for i, r := range rows {
		if r.focusable() {
			return i
		}
	}
	return -1
}

func (m Menu) setsearchwidth() int {
	return max0(panelusable(m.panelwidth()) - 4)
}

func (m Menu) textwidth(r setrow) int {
	inner := max0(panelusable(m.panelwidth()) - 2)
	if r.label == "" {
		return max0(inner - 2)
	}
	return chipwidth(inner)
}

func (m Menu) settingat(x, y int) (int, int, int) {
	px, py, pw := m.panelgeom()
	if x < px+2 || x >= px+pw {
		return -1, 0, 0
	}
	local := y - py
	top := m.settop()
	row := local - top + m.setscroll
	if local < top {
		headh := m.setheadh()
		if local == 1 && m.setscr != screenroot {
			return -3, 0, 0
		}
		if local >= 1+headh && local < 1+headh+3 {
			return -2, 0, 0
		}
		return -1, 0, 0
	}
	usable := panelusable(pw)
	rows := m.currentrows()
	views := m.setviews(rows, usable)
	if drop, dx, dy := m.dropdown(rows, views, usable, m.setscroll); drop != "" {
		n := len(rows[m.setidx].opts)
		lx := x - px - dx
		ly := local - top - dy
		if ly >= 0 && ly < n && lx >= 0 && lx < lipgloss.Width(drop) {
			return m.setidx, ly + 1, lx
		}
	}
	at := 0
	for i, v := range views {
		h := lipgloss.Height(v)
		if row < at+h {
			return i, row - at, x - px - 2
		}
		at += h
	}
	return -1, 0, 0
}

func (m Menu) clicksetting(x, y int) (tea.Model, tea.Cmd) {
	idx, local, cx := m.settingat(x, y)
	if idx == -3 {
		m.focus = focussettings
		return m.setback()
	}
	if idx == -2 {
		m.setsearchon = true
		m.setsearch.focusend()
		m.focus = focussettings
		return m.withtick(nil)
	}
	if idx < 0 {
		return m, nil
	}
	rows := m.currentrows()
	if idx >= len(rows) || !rows[idx].focusable() {
		return m, nil
	}
	m.focus = focussettings
	m.setsearchon = false

	if m.setopen && idx == m.setidx && rows[idx].kind == rowselect && local > 0 {
		if opt := local - 1; opt >= 0 && opt < len(rows[idx].opts) {
			cmd := m.pickoption(rows[idx], opt)
			m.setopen = false
			return m.withtick(cmd)
		}
		m.setopen = false
		return m.withtick(nil)
	}
	if m.setopen {
		m.setopen = false
		return m.withtick(nil)
	}

	var commit tea.Cmd
	if idx != m.setidx {
		commit = m.commitinput(rows)
		m.setopen = false
		m.setidx = idx
		m.loadinput(rows[idx])
		m.ensuresetvisible()
		if rows[idx].kind == rowtext {
			return m.withtick(commit)
		}
	}

	cur := rows[idx]
	if cur.kind == rowslider {
		inner := max0(panelusable(m.panelwidth()) - 2)
		x0 := inner - sliderw - 2 - lipgloss.Width(slidervalue(cur))
		if cx >= x0 && cx < x0+sliderw {
			return m.withtick(tea.Batch(commit, m.setslider(cur, sliderat(cx, x0, cur.lo, cur.hi))))
		}
	}
	if cur.kind == rowtext {
		return m.withtick(commit)
	}
	model, cmd := m.activate(cur)
	return model, tea.Batch(commit, cmd)
}

func (m Menu) setwheel(dir int) (tea.Model, tea.Cmd) {
	m.setscroll += dir * 3
	m.clampsetscroll()
	return m.withtick(nil)
}

func (m Menu) rendersetconfirm() string {
	var title, note string
	switch m.setconfirm {
	case "reset.user":
		title, note = m.tr.ResetUser, m.tr.NoteResetUser
	case "reset.prefs":
		title, note = m.tr.ResetPrefs, m.tr.NoteResetPrefs
	case "reset.tun":
		title, note = m.tr.ResetTun, m.tr.NoteResetTun
	}
	w := clampint(m.width*3/5, 30, 64)
	body := lipgloss.JoinVertical(lipgloss.Left,
		setdangersel.Render(title),
		"",
		sethint.Width(w).Render(note),
		"",
		lipgloss.PlaceHorizontal(w, lipgloss.Right, lipgloss.JoinHorizontal(lipgloss.Center,
			connectbtnblur.Render(m.tr.CancelBtn),
			"  ",
			disconnectbtn.Render(m.tr.ConfirmBtn),
		)),
	)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(btngray).
		Padding(1, 2).
		Render(body)
}

func (m Menu) mousesetconfirm(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	box := m.rendersetconfirm()
	bw, bh := lipgloss.Width(box), lipgloss.Height(box)
	x0, y0 := (m.width-bw)/2, (m.height-bh)/2
	if msg.X < x0 || msg.X >= x0+bw || msg.Y < y0 || msg.Y >= y0+bh {
		m.setconfirm = ""
		return m.withtick(nil)
	}
	if msg.Y != y0+bh-2 {
		return m, nil
	}
	confirmw := lipgloss.Width(disconnectbtn.Render(m.tr.ConfirmBtn))
	if msg.X >= x0+bw-3-confirmw {
		kind := strings.TrimPrefix(m.setconfirm, "reset.")
		if kind == "prefs" {
			kind = "settings"
		}
		m.setconfirm = ""
		return m.withtick(resetcmd(kind))
	}
	cancelw := lipgloss.Width(connectbtnblur.Render(m.tr.CancelBtn))
	if msg.X >= x0+bw-5-confirmw-cancelw {
		m.setconfirm = ""
		return m.withtick(nil)
	}
	return m, nil
}
