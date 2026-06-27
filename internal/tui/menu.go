package tui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/daemon"
	"github.com/alexandria-proxy/alexandria-cli/internal/i18n"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	revealtick       = 40 * time.Millisecond
	revealframes     = 28
	revealframesback = 12
	revealedge       = 0.22
	revealpeak       = 0.85

	idletick       = 80 * time.Millisecond
	busymin        = time.Second
	flashdur       = 2 * time.Second
	ringcycle      = 5.0
	ringsweep      = 1.5
	ringmax        = 0.85
	ringwidth      = 0.06
	ringpeak       = 0.15
	ringdelay      = 2.0
	ringretractdur = 0.18

	twocolmin = 96
)

var (
	connectbtn        = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(btngray).Foreground(lipgloss.Color("16"))
	connectbtnblur    = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("#7E868D")).Foreground(lipgloss.Color("237"))
	disconnectbtn     = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("#E0A6AC")).Foreground(lipgloss.Color("16"))
	disconnectbtnblur = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("#9C7A7E")).Foreground(lipgloss.Color("237"))
	timerstyle        = lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(btngray)

	modebtnsel = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(btngray).Foreground(lipgloss.Color("16"))
	modebtnon  = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("252"))
	modebtnoff = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("241"))

	actionrow    = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("250"))
	actionrowsel = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(btngray).Foreground(lipgloss.Color("16"))
	actiondanger = lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.Color("#E0A6AC")).Foreground(lipgloss.Color("16"))
)

type focuszone int

const (
	focusconnect focuszone = iota
	focusmode
	focussearch
	focusservers
)

type panelmode int

const (
	modelist panelmode = iota
	modeadd
	modeedit
	modeactions
)

type actionid int

const (
	actupdate actionid = iota
	actping
	actpin
	actcopy
	actedit
	actremove
)

type editzone int

const (
	editbody editzone = iota
	editsave
)

type menutickmsg struct{}

type subsloadedmsg struct{ subs []subscription.Subscription }

type addresultmsg struct {
	subs []subscription.Subscription
	err  string
}

type editsavedmsg struct {
	subs []subscription.Subscription
	err  string
}

type subsresultmsg struct {
	subs []subscription.Subscription
	err  string
}

func subscmd(req ipc.Request) tea.Cmd {
	return func() tea.Msg {
		resp, err := ipc.Send(req)
		if err != nil {
			return subsresultmsg{err: err.Error()}
		}
		if !resp.OK {
			return subsresultmsg{err: resp.Error}
		}
		return subsresultmsg{subs: resp.Subscriptions}
	}
}

func updatesubcmd(url string) tea.Cmd {
	return subscmd(ipc.Request{Cmd: "add_subscription", URL: url})
}

func pingsubcmd(url string) tea.Cmd {
	return subscmd(ipc.Request{Cmd: "ping_subscription", URL: url})
}

func togglepincmd(url string) tea.Cmd {
	return subscmd(ipc.Request{Cmd: "toggle_pin", URL: url})
}

func removesubcmd(url string) tea.Cmd {
	return subscmd(ipc.Request{Cmd: "remove_subscription", URL: url})
}

func saveservercmd(url string, srvidx int, raw string) tea.Cmd {
	return func() tea.Msg {
		resp, err := ipc.Send(ipc.Request{Cmd: "update_server", URL: url, SrvIdx: srvidx, Raw: raw})
		if err != nil {
			return editsavedmsg{err: err.Error()}
		}
		if !resp.OK {
			return editsavedmsg{err: resp.Error}
		}
		return editsavedmsg{subs: resp.Subscriptions}
	}
}

func prettyjson(s string) string {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(b)
}

func loadsubscmd() tea.Msg {
	_ = daemon.Ensure()
	resp, err := ipc.Send(ipc.Request{Cmd: "list"})
	if err != nil {
		return subsloadedmsg{}
	}
	return subsloadedmsg{subs: resp.Subscriptions}
}

type connectresultmsg struct {
	connected bool
	err       string
}

type statusmsg struct {
	connected bool
	mode      string
}

func connectcmd(url string, idx int, mode string) tea.Cmd {
	return func() tea.Msg {
		if err := daemon.Ensure(); err != nil {
			return connectresultmsg{err: err.Error()}
		}
		resp, err := ipc.Send(ipc.Request{Cmd: "connect", URL: url, SrvIdx: idx, Mode: mode})
		if err != nil {
			return connectresultmsg{err: err.Error()}
		}
		if !resp.OK {
			return connectresultmsg{err: resp.Error}
		}
		return connectresultmsg{connected: resp.Connected}
	}
}

func disconnectcmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := ipc.Send(ipc.Request{Cmd: "disconnect"})
		if err != nil {
			return connectresultmsg{err: err.Error()}
		}
		return connectresultmsg{connected: resp.Connected}
	}
}

func statuscmd() tea.Msg {
	resp, err := ipc.Send(ipc.Request{Cmd: "status"})
	if err != nil || !resp.OK {
		return statusmsg{}
	}
	return statusmsg{connected: resp.Connected, mode: resp.Mode}
}

func addsubcmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := daemon.Ensure(); err != nil {
			return addresultmsg{err: err.Error()}
		}
		resp, err := ipc.Send(ipc.Request{Cmd: "add_subscription", URL: url})
		if err != nil {
			return addresultmsg{err: err.Error()}
		}
		if !resp.OK {
			return addresultmsg{err: resp.Error}
		}
		return addresultmsg{subs: resp.Subscriptions}
	}
}

type Menu struct {
	tr         i18n.Strings
	colorcells [][]cell
	monocells  [][]cell
	logow      int
	connected  bool
	revealing  bool
	reverse    bool
	frame      int
	since      time.Time
	ringat     time.Time
	retracting bool
	retractat  time.Time
	ringfrom   float64
	ringto     float64
	panel      serverspanel
	focus      focuszone
	mode       panelmode
	form       addform
	width      int
	height     int
	ticking    bool
	editor     jsoneditor
	editsuburl string
	editsrvidx int
	editfocus  editzone
	editerr    string
	editname   string
	editproto  string

	editordrag    bool
	editordragdir int

	actionsuburl  string
	actionidx     int
	actionbusy    actionid
	actionrunning bool
	actiondone    bool
	actionconfirm bool
	actionstart   time.Time
	actionmsg     string

	connmode   string
	connecting bool
	connerr    string
	flashat    time.Time
}

func NewMenu(lang, mono, color string) Menu {
	monocells, w := parselogo(mono)
	colorcells, _ := parselogo(color)
	tr := i18n.T(lang)
	return Menu{tr: tr, monocells: monocells, colorcells: colorcells, logow: w, panel: newserverspanel(tr), ticking: true, connmode: "proxy"}
}

func (m Menu) Init() tea.Cmd {
	return tea.Batch(tea.HideCursor, m.tick(), loadsubscmd, statuscmd)
}

func (m Menu) tick() tea.Cmd {
	d := idletick
	if m.revealing || m.actionrunning {
		d = revealtick
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return menutickmsg{} })
}

func (m Menu) animating() bool {
	return m.revealing || m.retracting || m.connected ||
		m.focus == focussearch || m.mode == modeadd ||
		(m.editordrag && m.editordragdir != 0) ||
		m.actionrunning || m.flashlevel() > 0
}

func (m Menu) flashlevel() float64 {
	if m.flashat.IsZero() {
		return 0
	}
	e := time.Since(m.flashat)
	if e >= flashdur {
		return 0
	}
	return 1 - float64(e)/float64(flashdur)
}

func (m Menu) withtick(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if m.animating() && !m.ticking {
		m.ticking = true
		if cmd == nil {
			return m, m.tick()
		}
		return m, tea.Batch(cmd, m.tick())
	}
	return m, cmd
}

func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case menutickmsg:
		if m.revealing {
			m.frame++
			end := revealframes
			if m.reverse {
				end = revealframesback
			}
			if m.frame >= end {
				m.revealing = false
				if m.connected {
					m.ringat = time.Now()
				}
			}
		}
		if m.retracting && time.Since(m.retractat).Seconds() >= ringretractdur {
			m.retracting = false
		}
		if m.mode == modeedit && m.editordrag && m.editordragdir != 0 {
			ew, eh := m.editordims()
			m.editor.dragextend(m.editordragdir, ew, eh)
		}
		if m.actionrunning && m.actiondone && time.Since(m.actionstart) >= busymin {
			m.finishaction()
		}
		if !m.animating() {
			m.ticking = false
			return m, nil
		}
		return m, m.tick()
	case subsloadedmsg:
		m.panel.subs = msg.subs
		return m, nil
	case addresultmsg:
		m.form.loading = false
		if msg.err != "" {
			m.form.err = msg.err
			return m, nil
		}
		m.panel.subs = msg.subs
		m.mode = modelist
		return m, nil
	case editsavedmsg:
		if msg.err != "" {
			m.editerr = msg.err
			return m, nil
		}
		m.panel.subs = msg.subs
		m.editerr = ""
		m.mode = modelist
		return m, nil
	case subsresultmsg:
		if msg.err == "" {
			m.panel.subs = msg.subs
			if n := m.panel.itemcount(); m.panel.cursor >= n {
				m.panel.cursor = max0(n - 1)
			}
		}
		m.actiondone = true
		faked := m.actionbusy == actupdate || m.actionbusy == actping
		if !faked || time.Since(m.actionstart) >= busymin {
			m.finishaction()
		}
		return m.withtick(nil)
	case connectresultmsg:
		m.connecting = false
		if msg.err != "" {
			m.connerr = msg.err
			return m.withtick(nil)
		}
		m.connerr = ""
		if msg.connected != m.connected {
			m.animconnect(msg.connected)
		}
		return m.withtick(nil)
	case statusmsg:
		if msg.mode != "" {
			m.connmode = msg.mode
		}
		if msg.connected && !m.connected && !m.connecting {
			m.connected = true
			m.revealing = false
			m.reverse = false
			m.since = time.Now()
			m.ringat = time.Now()
		}
		return m.withtick(nil)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, tea.HideCursor
	case tea.MouseMsg:
		if m.mode == modeedit {
			m.editmouse(msg)
			return m.withtick(nil)
		}
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeedit {
			return m.updateeditor(msg)
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == modeactions {
			return m.updateactions(msg)
		}
		if m.mode == modeadd {
			f, res := m.form.update(msg, m.searchwidth())
			m.form = f
			switch res {
			case formcancel:
				m.mode = modelist
			case formsubmit:
				m.form.err = ""
				m.form.loading = true
				return m.withtick(addsubcmd(strings.TrimSpace(m.form.url.value)))
			}
			return m.withtick(nil)
		}
		if msg.String() == "ctrl+a" && m.focus != focussearch {
			m.mode = modeadd
			m.form = newaddform(m.tr)
			m.focus = focusconnect
			m.panel.focused = false
			return m.withtick(nil)
		}
		if m.focus == focussearch {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "tab":
				m.focus = focusconnect
				m.panel.focused = false
				return m, nil
			case "esc":
				m.panel.search = textinput{}
				if m.panel.itemcount() > 0 {
					m.focus = focusservers
					m.panel.focused = false
					m.panel.serversfocused = true
					m.panel.cursor = 0
				} else {
					m.focus = focusconnect
					m.panel.focused = false
				}
				return m, nil
			case "ctrl+down":
				if m.panel.itemcount() > 0 {
					m.focus = focusservers
					m.panel.focused = false
					m.panel.serversfocused = true
					m.panel.cursor = 0
				}
				return m, nil
			case "left":
				if m.panel.search.cursorpos == 0 {
					m.focus = focusconnect
					m.panel.focused = false
					return m, nil
				}
			case "down":
				if m.panel.search.value == "" {
					if m.panel.itemcount() > 0 {
						m.focus = focusservers
						m.panel.focused = false
						m.panel.serversfocused = true
						m.panel.cursor = 0
					}
					return m, nil
				}
			}
			m.panel.search.handlekey(msg, m.searchwidth())
			return m, nil
		}
		if m.focus == focusservers {
			items := m.panel.items()
			hasit := m.panel.cursor >= 0 && m.panel.cursor < len(items)
			var it selitem
			if hasit {
				it = items[m.panel.cursor]
			}
			header := hasit && it.srvidx < 0

			switch msg.String() {
			case "esc":
				m.focus = focusconnect
				m.panel.serversfocused = false
				m.panel.btnidx = -1
				return m.withtick(nil)
			case "up", "k", "ctrl+up":
				if m.panel.cursor == 0 {
					m.focus = focussearch
					m.panel.serversfocused = false
					m.panel.focused = true
					m.panel.btnidx = -1
					m.panel.search.focusend()
					return m.withtick(nil)
				}
				m.panel.cursor--
				m.panel.btnidx = -1
				return m, nil
			case "down", "j", "ctrl+down":
				if n := m.panel.itemcount(); m.panel.cursor < n-1 {
					m.panel.cursor++
					m.panel.btnidx = -1
				}
				return m, nil
			case "left", "h":
				if header && m.panel.btnidx > 0 {
					m.panel.btnidx--
					return m, nil
				}
				if header && m.panel.btnidx == 0 {
					m.panel.btnidx = -1
					return m, nil
				}
				m.focus = focusconnect
				m.panel.serversfocused = false
				m.panel.btnidx = -1
				return m.withtick(nil)
			case "right", "l":
				if header {
					if m.panel.btnidx < 0 {
						m.panel.btnidx = 0
					} else if m.panel.btnidx < headerbtns-1 {
						m.panel.btnidx++
					}
					return m, nil
				}
				if hasit {
					return m.openeditor(it)
				}
				return m, nil
			case " ":
				if header {
					url := m.panel.subs[it.subidx].URL
					m.panel.collapsed[url] = !m.panel.collapsed[url]
					return m, nil
				}
				if hasit {
					return m.openeditor(it)
				}
				return m, nil
			case "enter":
				if !hasit {
					return m, nil
				}
				if header {
					if m.panel.btnidx < 0 {
						url := m.panel.subs[it.subidx].URL
						m.panel.collapsed[url] = !m.panel.collapsed[url]
						return m, nil
					}
					return m.runheaderbtn(it)
				}
				return m.openeditor(it)
			}
			return m, nil
		}
		if m.focus == focusmode {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc", "up":
				m.focus = focusconnect
				return m.withtick(nil)
			case "left", "h":
				m.connmode = "proxy"
				return m, nil
			case "right", "l":
				m.connmode = "tun"
				return m, nil
			case "enter", " ":
				if m.connmode == "proxy" {
					m.connmode = "tun"
				} else {
					m.connmode = "proxy"
				}
				return m, nil
			case "tab", "down":
				if m.panel.itemcount() > 0 {
					m.focus = focusservers
					m.panel.serversfocused = true
					m.panel.cursor = 0
					m.panel.btnidx = -1
				}
				return m.withtick(nil)
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "right", "tab":
			if m.panel.itemcount() > 0 {
				m.focus = focusservers
				m.panel.serversfocused = true
				m.panel.cursor = 0
				m.panel.btnidx = -1
			} else {
				m.focus = focussearch
				m.panel.focused = true
				m.panel.search.focusend()
			}
			return m.withtick(nil)
		case "down":
			m.focus = focusmode
			return m.withtick(nil)
		case "enter", " ":
			if m.connecting {
				return m, nil
			}
			if m.connected {
				m.connecting = true
				return m.withtick(disconnectcmd())
			}
			url, idx, ok := m.connserver()
			if !ok {
				m.flashat = time.Now()
				return m.withtick(nil)
			}
			m.connecting = true
			m.connerr = ""
			return m.withtick(connectcmd(url, idx, m.connmode))
		}
	}
	return m, nil
}

func (m Menu) connserver() (string, int, bool) {
	items := m.panel.items()
	if m.panel.cursor >= 0 && m.panel.cursor < len(items) {
		it := items[m.panel.cursor]
		if it.srvidx >= 0 {
			return m.panel.subs[it.subidx].URL, it.srvidx, true
		}
	}
	for _, sub := range m.panel.subs {
		if len(sub.Servers) > 0 {
			return sub.URL, 0, true
		}
	}
	return "", -1, false
}

func (m *Menu) animconnect(target bool) {
	p := m.phase()
	rnow, ron := m.ring()
	m.connected = target
	m.reverse = !m.connected
	m.revealing = true
	if m.reverse {
		m.frame = int(p / 2.0 * float64(revealframesback))
	} else {
		m.frame = int((2.0 - p) / 2.0 * float64(revealframes))
	}
	m.ringat = time.Time{}
	if m.connected {
		m.since = time.Now()
		m.retracting = false
	} else {
		m.retracting = ron
		m.ringfrom = rnow
		m.retractat = time.Now()
		if rnow/ringmax > 0.4 {
			m.ringto = ringmax + ringwidth
		} else {
			m.ringto = 0
		}
	}
}

func (m Menu) View() string {
	if m.mode == modeedit && m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.rendereditmodal())
	}

	logo := m.renderlogo()

	onconnect := m.focus == focusconnect && m.mode == modelist
	var btn string
	if m.connected {
		db := disconnectbtn
		if !onconnect {
			db = disconnectbtnblur
		}
		btn = lipgloss.JoinHorizontal(
			lipgloss.Center,
			db.Render(m.tr.Disconnect),
			timerstyle.Render("⏱"+elapsed(time.Since(m.since))),
		)
	} else {
		cb := connectbtn
		if !onconnect {
			cb = connectbtnblur
		}
		btn = cb.Render(m.tr.Connect)
	}

	unit := lipgloss.JoinVertical(lipgloss.Center, logo, "", btn, "", m.rendermode())
	if m.connerr != "" {
		unit = lipgloss.JoinVertical(lipgloss.Center, unit, "", errstyle.Render(m.connerr))
	}
	if m.width == 0 || m.height == 0 {
		return unit
	}

	busyurl, busybtn := "", -1
	if m.actionrunning {
		busyurl = m.actionsuburl
		switch m.actionbusy {
		case actupdate:
			busybtn = 0
		case actping:
			busybtn = 1
		}
	}
	dropdown, anchorurl := "", ""
	if m.mode == modeactions {
		dropdown = m.renderactions()
		anchorurl = m.actionsuburl
	}

	flash := m.flashlevel()
	if m.width < twocolmin {
		top := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, unit)
		content := m.panel.render(m.width, m.height, busyurl, busybtn, dropdown, anchorurl, flash)
		if m.mode == modeadd {
			content = m.form.render(m.width)
		}
		return lipgloss.JoinVertical(lipgloss.Left, top, content)
	}

	leftw := m.width / 2
	rightw := m.width - leftw
	rightcontent := m.panel.render(rightw, m.height, busyurl, busybtn, dropdown, anchorurl, flash)
	if m.mode == modeadd {
		rightcontent = m.form.render(rightw)
	}
	left := lipgloss.Place(leftw, m.height, lipgloss.Center, lipgloss.Center, unit)
	right := lipgloss.Place(rightw, m.height, lipgloss.Left, lipgloss.Top, rightcontent)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Menu) editmodalsize() (int, int) {
	boxw := m.width * 4 / 5
	if boxw > 110 {
		boxw = 110
	}
	if boxw < 24 {
		boxw = max0(m.width - 2)
	}
	boxh := m.height * 4 / 5
	if boxh < 10 {
		boxh = max0(m.height - 2)
	}
	return boxw, boxh
}

func (m Menu) editordims() (int, int) {
	boxw, boxh := m.editmodalsize()
	w := boxw - 6
	h := boxh - 9
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func (m Menu) updateeditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modelist
		m.editerr = ""
		m.editordrag = false
		m.editordragdir = 0
		return m.withtick(nil)
	case "ctrl+s":
		m.editerr = ""
		return m.withtick(saveservercmd(m.editsuburl, m.editsrvidx, m.editor.value()))
	case "ctrl+c", "ctrl+shift+c":
		return m, osc52copy(m.editor.copytext())
	case "tab", "shift+tab":
		m.editfocus = (m.editfocus + 1) % 2
		return m.withtick(nil)
	}
	if m.editfocus == editsave {
		switch msg.String() {
		case "enter", " ":
			m.editerr = ""
			return m.withtick(saveservercmd(m.editsuburl, m.editsrvidx, m.editor.value()))
		case "left", "up":
			m.editfocus = editbody
			return m.withtick(nil)
		}
		return m.withtick(nil)
	}
	ew, eh := m.editordims()
	m.editor.handlekey(msg, ew, eh)
	return m.withtick(nil)
}

func (m Menu) openeditor(it selitem) (tea.Model, tea.Cmd) {
	srv := m.panel.subs[it.subidx].Servers[it.srvidx]
	m.editor = newjsoneditor(prettyjson(srv.Raw))
	m.editsuburl = m.panel.subs[it.subidx].URL
	m.editsrvidx = it.srvidx
	m.editfocus = editbody
	m.editerr = ""
	m.editname = srv.Name
	m.editproto = strings.ToLower(srv.Protocol)
	if isjsonconfig(srv.Raw) {
		m.editproto += " / json"
	}
	m.mode = modeedit
	return m.withtick(nil)
}

func (m *Menu) startbusy(url string, a actionid) {
	m.actionsuburl = url
	m.actionbusy = a
	m.actionrunning = true
	m.actiondone = false
	m.actionstart = time.Now()
}

func (m *Menu) finishaction() {
	m.actionrunning = false
	m.actiondone = false
	m.actionconfirm = false
	if m.mode == modeactions {
		m.mode = modelist
	}
}

func (m Menu) runheaderbtn(it selitem) (tea.Model, tea.Cmd) {
	url := m.panel.subs[it.subidx].URL
	switch m.panel.btnidx {
	case 0:
		m.startbusy(url, actupdate)
		return m.withtick(updatesubcmd(url))
	case 1:
		m.startbusy(url, actping)
		return m.withtick(pingsubcmd(url))
	default:
		m.actionsuburl = url
		m.actionidx = 0
		m.actionrunning = false
		m.actionmsg = ""
		m.actionconfirm = false
		m.mode = modeactions
		return m.withtick(nil)
	}
}

func (m Menu) actionitems() []actionid {
	return []actionid{actupdate, actping, actpin, actcopy, actedit, actremove}
}

func (m Menu) subbyurl(url string) (subscription.Subscription, bool) {
	for _, s := range m.panel.subs {
		if s.URL == url {
			return s, true
		}
	}
	return subscription.Subscription{}, false
}

func actionicon(a actionid) string {
	switch a {
	case actupdate:
		return "↻"
	case actping:
		return "⏱"
	case actpin:
		return "🖈"
	case actcopy:
		return "⧉"
	case actedit:
		return "✎"
	case actremove:
		return "✕"
	}
	return " "
}

func (m Menu) actionlabel(a actionid, sub subscription.Subscription) string {
	switch a {
	case actupdate:
		return m.tr.ActionUpdate
	case actping:
		return m.tr.ActionTestPing
	case actpin:
		if sub.Pinned {
			return m.tr.ActionUnpin
		}
		return m.tr.ActionPin
	case actcopy:
		return m.tr.ActionCopyURL
	case actedit:
		return m.tr.ActionEdit
	case actremove:
		return m.tr.ActionRemove
	}
	return ""
}

func (m Menu) actionstatus(a actionid) string {
	switch a {
	case actupdate:
		return m.tr.Updating
	case actping:
		return m.tr.Pinging
	}
	return ""
}

func (m Menu) updateactions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.actionrunning {
		return m.withtick(nil)
	}
	items := m.actionitems()
	switch msg.String() {
	case "esc", "left", "h", "a", "q":
		m.mode = modelist
		m.actionmsg = ""
		m.actionconfirm = false
		return m.withtick(nil)
	case "up", "k", "ctrl+up":
		m.actionidx = (m.actionidx - 1 + len(items)) % len(items)
		m.actionmsg = ""
		m.actionconfirm = false
		return m, nil
	case "down", "j", "ctrl+down":
		m.actionidx = (m.actionidx + 1) % len(items)
		m.actionmsg = ""
		m.actionconfirm = false
		return m, nil
	case "enter", " ", "right", "l":
		a := items[m.actionidx]
		if a == actremove && !m.actionconfirm {
			m.actionconfirm = true
			return m.withtick(nil)
		}
		return m.runaction(a)
	}
	return m, nil
}

func (m Menu) runaction(a actionid) (tea.Model, tea.Cmd) {
	sub, ok := m.subbyurl(m.actionsuburl)
	if !ok {
		m.mode = modelist
		return m, nil
	}
	switch a {
	case actupdate:
		m.startbusy(sub.URL, actupdate)
		return m.withtick(updatesubcmd(sub.URL))
	case actping:
		m.startbusy(sub.URL, actping)
		return m.withtick(pingsubcmd(sub.URL))
	case actpin:
		m.actionbusy, m.actionrunning = actpin, true
		return m.withtick(togglepincmd(sub.URL))
	case actcopy:
		m.actionmsg = m.tr.Copied
		return m, osc52copy(sub.URL)
	case actedit:
		m.mode = modeadd
		m.form = newaddform(m.tr)
		m.form.editing = true
		m.form.url.value = sub.URL
		m.form.url.focusend()
		m.form.focus = fieldurl
		m.focus = focusconnect
		m.panel.focused = false
		m.panel.serversfocused = false
		return m.withtick(nil)
	case actremove:
		m.actionbusy, m.actionrunning = actremove, true
		return m.withtick(removesubcmd(sub.URL))
	}
	return m, nil
}

func (m Menu) renderactions() string {
	sub, _ := m.subbyurl(m.actionsuburl)

	items := m.actionitems()
	menuw := 14
	for _, a := range items {
		if w := lipgloss.Width(actionicon(a)) + 2 + lipgloss.Width(m.actionlabel(a, sub)) + 3; w > menuw {
			menuw = w
		}
	}

	var rows []string
	for i, a := range items {
		busy := m.actionrunning && a == m.actionbusy
		label := m.actionlabel(a, sub)
		if a == actcopy && m.actionmsg != "" {
			label = m.actionmsg
		}
		if a == actremove && m.actionconfirm {
			label = m.actionlabel(actremove, sub) + "?"
		}
		text := actionicon(a) + "  " + label
		st := actionrow
		switch {
		case busy:
			status := m.actionstatus(a)
			if status == "" {
				status = m.actionlabel(a, sub)
			}
			text = actionicon(a) + "  " + shimmer(status)
		case m.actionidx == i && a == actremove:
			st = actiondanger
		case m.actionidx == i:
			st = actionrowsel
		}
		rows = append(rows, st.Width(menuw).Render(text))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(btngray).
		Render(body)
}

func (m *Menu) editmouse(msg tea.MouseMsg) {
	m.editfocus = editbody
	ew, eh := m.editordims()
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.editor.scrollby(-3, eh)
		return
	case tea.MouseButtonWheelDown:
		m.editor.scrollby(3, eh)
		return
	}

	innerw := ew + 2
	cw := innerw
	if w := 1 + lipgloss.Width(m.editname); w > cw {
		cw = w
	}
	hinttext, hintst := m.tr.EditHint, panelfaint
	if m.editerr != "" {
		hinttext, hintst = m.editerr, errstyle
	}
	hinth := lipgloss.Height(hintst.Width(innerw).Render(hinttext))
	modalw := cw + 4
	modalh := 3 + (eh + 2) + hinth + 1 + 2
	textx0 := (m.width-modalw)/2 + 3
	texty0 := (m.height-modalh)/2 + 5
	vr := msg.Y - texty0
	vc := msg.X - textx0

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return
		}
		m.editmoveto(vr, vc, ew, eh)
		m.editor.selrow, m.editor.selcol = m.editor.row, m.editor.col
		m.editordrag = true
		m.editordragdir = 0
	case tea.MouseActionMotion:
		if !m.editordrag {
			return
		}
		switch {
		case vr < 0:
			m.editordragdir = -1
		case vr >= eh:
			m.editordragdir = 1
		default:
			m.editordragdir = 0
		}
		m.editmoveto(vr, vc, ew, eh)
	case tea.MouseActionRelease:
		m.editordrag = false
		m.editordragdir = 0
	}
}

func (m *Menu) editmoveto(vr, vc, w, h int) {
	e := &m.editor
	e.row = clampint(e.scroll+clampint(vr, 0, max0(h-1)), 0, len(e.lines)-1)
	e.col = clampint(e.hoff+clampint(vc, 0, max0(w-1)), 0, len(e.line()))
	e.clamp(w, h)
}

func osc52copy(s string) tea.Cmd {
	return func() tea.Msg {
		os.Stdout.WriteString("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\x07")
		return nil
	}
}

func (m Menu) rendereditmodal() string {
	ew, eh := m.editordims()
	innerw := ew + 2

	title := paneltitlest.Render(m.tr.EditServerTitle)
	name := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).PaddingLeft(1).Render(m.editname)
	proto := panelfaint.PaddingLeft(3).Render(m.editproto)

	editborder := paneldim
	if m.editfocus == editbody {
		editborder = btngray
	}
	editorbox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(editborder).
		Width(ew).
		Render(m.editor.view(ew, eh, m.editfocus == editbody))

	hint := panelfaint.Width(innerw).Render(m.tr.EditHint)
	if m.editerr != "" {
		hint = errstyle.Width(innerw).Render(m.editerr)
	}

	savest := connectbtnblur
	if m.editfocus == editsave {
		savest = connectbtn
	}
	savebtn := lipgloss.JoinHorizontal(lipgloss.Center, savest.Render(m.tr.SaveBtn), panelfaint.Render("(tab)"))
	save := lipgloss.PlaceHorizontal(innerw, lipgloss.Right, savebtn)

	content := lipgloss.JoinVertical(lipgloss.Left, title, name, proto, editorbox, hint, save)
	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(content)
}

func (m Menu) searchwidth() int {
	rightw := m.width - m.width/2
	cw := rightw - 6
	if cw < 1 {
		cw = 1
	}
	return cw
}

func (m Menu) rendermode() string {
	focused := m.focus == focusmode && m.mode == modelist
	pill := func(label, mode string) string {
		switch {
		case m.connmode == mode && focused:
			return modebtnsel.Render(label)
		case m.connmode == mode:
			return modebtnon.Render(label)
		default:
			return modebtnoff.Render(label)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, pill("proxy", "proxy"), "  ", pill("tun", "tun"))
}

func (m Menu) renderlogo() string {
	wf, hf := float64(m.logow), float64(len(m.colorcells))
	phase := m.phase()
	ringr, ringon := m.ring()

	var b strings.Builder
	for r := range m.colorcells {
		for c := range m.colorcells[r] {
			cl := m.colorcells[r][c]
			if m.revealing {
				s := float64(c)/wf + float64(r)/hf
				switch {
				case s >= phase:
					if d := s - phase; d < revealedge && lit(cl.fg, cl.bg) {
						t := (1 - d/revealedge) * revealpeak
						cl.fg, cl.bg = boost(cl.fg, t), boost(cl.bg, t)
					}
				case r < len(m.monocells) && c < len(m.monocells[r]):
					cl = m.monocells[r][c]
				}
			} else if !m.connected && r < len(m.monocells) && c < len(m.monocells[r]) {
				cl = m.monocells[r][c]
			}
			if ringon && lit(cl.fg, cl.bg) {
				nx, ny := float64(c)/wf-0.5, float64(r)/hf-0.5
				if d := math.Abs(math.Hypot(nx, ny) - ringr); d < ringwidth {
					t := (1 - d/ringwidth) * ringpeak
					cl.fg, cl.bg = glint(cl.fg, t), glint(cl.bg, t)
				}
			}
			b.WriteString(sgr(cl.fg, cl.bg, cl.reverse))
			b.WriteRune(cl.ch)
		}
		b.WriteString("\x1b[0m")
		if r < len(m.colorcells)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func glint(c *rgb, t float64) *rgb {
	if c == nil {
		return nil
	}
	if 0.299*float64(c.r)+0.587*float64(c.g)+0.114*float64(c.b) < 160 {
		return boost(c, t)
	}
	f := 1 - t
	return &rgb{int(float64(c.r)*f + 0.5), int(float64(c.g)*f + 0.5), int(float64(c.b)*f + 0.5)}
}

func (m Menu) ring() (float64, bool) {
	if m.retracting {
		el := time.Since(m.retractat).Seconds()
		if el >= ringretractdur {
			return 0, false
		}
		return m.ringfrom + (m.ringto-m.ringfrom)*(el/ringretractdur), true
	}
	if !m.connected || m.revealing || m.ringat.IsZero() {
		return 0, false
	}
	el := time.Since(m.ringat).Seconds() - ringdelay
	if el < 0 {
		return 0, false
	}
	cyc := math.Mod(el, ringcycle)
	if cyc > ringsweep {
		return 0, false
	}
	return cyc / ringsweep * ringmax, true
}

func (m Menu) phase() float64 {
	switch {
	case m.revealing && m.reverse:
		return float64(m.frame) / float64(revealframesback) * 2.0
	case m.revealing:
		return 2.0 - float64(m.frame)/float64(revealframes)*2.0
	case m.connected:
		return 0.0
	default:
		return 2.0
	}
}

func elapsed(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s%3600/60, s%60)
}
