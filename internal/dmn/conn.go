package dmn

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/core"
	"github.com/alexandria-proxy/alexandria-cli/internal/gen"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/route"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
	"github.com/alexandria-proxy/alexandria-cli/internal/sysp"
)

type proc struct {
	name  string
	path  string
	args  []string
	env   []string
	stdin string
	drop  bool
}

type conn struct {
	mu         sync.Mutex
	opmu       sync.Mutex
	wg         sync.WaitGroup
	cmds       map[string]*exec.Cmd
	connected  bool
	restarting bool
	url        string
	srvidx     int
	mode       string
	socks      int
	gen        int64
	since      time.Time
	lasterr    string
	lastcode   string
	metrics    int
	stop       chan struct{}
	stats      stattrack
}

type connstate struct {
	live       bool
	restarting bool
	url        string
	idx        int
	mode       string
	socks      int
	gen        int64
	since      time.Time
}

func cfgfile(name string) (string, error) {
	return conf.File(name)
}

func logcapfor(set conf.Logs) int64 {
	if set.Max <= 0 {
		return 0
	}
	return set.Max
}

type caplog struct {
	f   *os.File
	src string
	max int64
	n   int64
}

func (w *caplog) Write(p []byte) (int, error) {
	logcore(w.src, p)
	if w.max > 0 && w.n+int64(len(p)) > w.max {
		if err := w.f.Truncate(0); err == nil {
			w.n = 0
		}
	}
	n, err := w.f.Write(p)
	w.n += int64(n)
	return n, err
}

func (w *caplog) Close() error { return w.f.Close() }

func cleanlogs() {
	for _, name := range []string{"xray.log", "sing-box.log"} {
		if p, err := cfgfile(name); err == nil {
			_ = os.Remove(p)
		}
	}
}

func openlog(name string) *caplog {
	p, err := cfgfile(name + ".log")
	if err != nil {
		return nil
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	conf.Own(p)
	n := int64(0)
	if fi, err := f.Stat(); err == nil {
		n = fi.Size()
	}
	return &caplog{f: f, src: name, max: logcapfor(logs.settings()), n: n}
}

func genopts(s conf.Settings) gen.Opts {
	return gen.Opts{
		Metrics:    gen.Defaults().Metrics,
		Socksport:  s.Socksport,
		Sniffing:   s.Advanced.Sniffing,
		Preferip:   s.Preferip,
		Frag:       s.Fragment.On,
		Fragpkt:    s.Fragment.Packets,
		Fraglen:    s.Fragment.Length,
		Fragint:    s.Fragment.Interval,
		Mux:        s.Mux.On,
		Muxconc:    s.Mux.Concurrency,
		Localdns:   s.Advanced.Localdns,
		Jsondns:    s.Advanced.Jsondns,
		Resolvesrv: s.Advanced.Resolvesrv,
	}
}

func (c *conn) isconnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *conn) state() connstate {
	c.mu.Lock()
	defer c.mu.Unlock()
	return connstate{
		live:       c.connected,
		restarting: c.restarting,
		url:        c.url,
		idx:        c.srvidx,
		mode:       c.mode,
		socks:      c.socks,
		gen:        c.gen,
		since:      c.since,
	}
}

func (c *conn) superseded(gen int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen != gen || !c.connected
}

func (c *conn) setrestarting(v bool) {
	c.mu.Lock()
	c.restarting = v
	c.mu.Unlock()
}

func (c *conn) status() ipc.Response {
	c.mu.Lock()
	live := c.connected || c.restarting
	r := ipc.Response{Ok: true, Connected: live, Mode: c.mode, Error: c.lasterr, Code: c.lastcode}
	if live {
		r.Activeurl, r.Activesrv = c.url, c.srvidx
		r.Since = c.since.Unix()
	}
	metrics := c.metrics
	c.mu.Unlock()

	if live && metrics > 0 {
		s := c.stats.snapshot()
		r.Hasstats = true
		r.Uptotal, r.Downtotal = s.up, s.down
		r.Uprate, r.Downrate = s.uprate, s.downrate
	}
	return r
}

func (c *conn) disconnect() ipc.Response {
	c.stopop()
	return ipc.Response{Ok: true, Connected: false}
}

func (c *conn) stopop() {
	c.opmu.Lock()
	defer c.opmu.Unlock()
	c.stopnow()
}

func (c *conn) stopnow() {
	c.mu.Lock()
	stop := c.stop
	c.stop, c.cmds = nil, nil
	c.connected = false
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	c.wg.Wait()

	clearsysproxy()
	tuncleanup(gen.Tunname)
}

func gracefulstop(p *os.Process, done chan struct{}) {
	_ = terminate(p)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = p.Kill()
		<-done
	}
}

func (c *conn) connect(srv sub.Server, url string, idx int, mode string) ipc.Response {
	c.opmu.Lock()
	defer c.opmu.Unlock()

	if mode == "tun" && !conf.Elevated() {
		return ipc.Response{Error: "tun mode needs the daemon to run as root", Code: "needelevate"}
	}
	uc, _ := conf.Load()
	set := uc.Settings
	logs.configure(set.Logs)

	opts := genopts(set)
	if mode == "tun" {
		c.stopnow()
		opts.Pinsrv = true
	}
	cfg, err := gen.Xray(srv.Raw, opts)
	if errors.Is(err, gen.Errresolve) {
		return ipc.Response{Error: err.Error(), Code: "noresolve"}
	}
	if err != nil {
		return ipc.Response{Error: "unsupported server: " + err.Error(), Code: "unsupported"}
	}
	xpath, err := core.Xray()
	if err != nil {
		return ipc.Response{Error: "xray core not found", Code: "nocore"}
	}
	procs := []proc{{
		name:  "xray",
		path:  xpath,
		args:  []string{"run", "-c", "stdin:"},
		env:   append(os.Environ(), "XRAY_LOCATION_ASSET="+filepath.Dir(xpath)),
		stdin: cfg,
		drop:  true,
	}}

	socksport := gen.Socksport(cfg)

	if mode == "tun" {
		sbpath, err := core.Singbox()
		if err != nil {
			return ipc.Response{Error: "sing-box (tun engine) not found", Code: "nosingbox"}
		}
		tuncfg, err := writetun(set, socksport)
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		procs = append(procs, proc{
			name: "sing-box",
			path: sbpath,
			args: []string{"run", "-c", tuncfg},
			env:  os.Environ(),
		})
	}

	resp := c.start(url, idx, mode, procs, socksport, gen.Metricsport(cfg))
	if resp.Ok && resp.Connected {
		logevent("connected · " + mode + " · " + srv.Name)
		applysysproxy(set)
	}
	return resp
}

func tunconfig(set conf.Settings, socks int) string {
	if set.Advanced.Tunconfig == "custom" {
		if p, err := route.Custompath(); err == nil {
			if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
				return tame(b)
			}
		}
		logs.push("daemon", "warn", "tun.config is custom but "+route.Custom+" is missing or empty, using the default")
	}
	excluded, bad, err := route.Load()
	if err != nil {
		logerr("route: " + err.Error())
	}
	for _, b := range bad {
		logs.push("daemon", "warn", "route line "+strconv.Itoa(b.Line)+": skipping '"+b.Text+"'")
	}
	return gen.Tun(gen.Tunopts{
		Socksport: socks,
		Stack:     set.Advanced.Tunstack,
		Dnson:     set.Advanced.Tundnson,
		Dns:       set.Advanced.Tundns,
		Preferip:  set.Preferip,
		Excluded:  excluded,
	})
}

func tame(raw []byte) string {
	var cfg map[string]any
	if json.Unmarshal(raw, &cfg) != nil {
		return string(raw)
	}
	if lg, ok := cfg["log"].(map[string]any); ok {
		delete(lg, "output")
	}
	if ex, ok := cfg["experimental"].(map[string]any); ok {
		delete(ex, "cache_file")
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func writetun(set conf.Settings, socks int) (string, error) {
	p, err := cfgfile("tun.json")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(tunconfig(set, socks)), 0600); err != nil {
		return "", err
	}
	conf.Own(p)
	return p, nil
}

func (c *conn) reloadtun() {
	c.opmu.Lock()
	defer c.opmu.Unlock()

	c.mu.Lock()
	live, mode, socks := c.connected, c.mode, c.socks
	var cmd *exec.Cmd
	if c.cmds != nil {
		cmd = c.cmds["sing-box"]
	}
	c.mu.Unlock()
	if !live || mode != "tun" {
		return
	}

	uc, _ := conf.Load()
	if _, err := writetun(uc.Settings, socks); err != nil {
		logerr("tun reload: " + err.Error())
		return
	}
	logevent("routes changed, restarting sing-box")
	if cmd != nil && cmd.Process != nil {
		_ = terminate(cmd.Process)
	}
}

func applysysproxy(s conf.Settings) {
	if !s.Advanced.Sysproxy {
		return
	}
	if err := sysp.Enable(sysp.Opts{Host: "127.0.0.1", Socks: s.Socksport, Http: s.Socksport + 1}); err != nil {
		logerr("system proxy: " + err.Error())
	}
}

func clearsysproxy() {
	uc, err := conf.Load()
	if err != nil || !uc.Settings.Advanced.Sysproxy {
		return
	}
	if err := sysp.Disable(); err != nil {
		logerr("system proxy: " + err.Error())
	}
}

func (c *conn) start(url string, idx int, mode string, procs []proc, socks, metrics int) ipc.Response {
	c.stopnow()

	stop := make(chan struct{})
	c.mu.Lock()
	c.connected = true
	c.url, c.srvidx, c.mode = url, idx, mode
	c.socks = socks
	c.gen++
	c.since = time.Now()
	c.lasterr, c.lastcode = "", ""
	c.metrics = metrics
	c.stop = stop
	c.cmds = make(map[string]*exec.Cmd, len(procs))
	c.mu.Unlock()

	c.stats.reset()

	c.wg.Add(len(procs))
	for _, p := range procs {
		go c.supervise(p, stop)
	}
	if metrics > 0 {
		c.wg.Add(1)
		go c.pollstats(metrics, stop)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		ok, lasterr, lastcode := c.connected, c.lasterr, c.lastcode
		c.mu.Unlock()
		if !ok {
			if lasterr == "" {
				lasterr = "failed to start"
			}
			return ipc.Response{Error: lasterr, Code: lastcode}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return c.status()
}

func (c *conn) supervise(p proc, stop chan struct{}) {
	defer c.wg.Done()
	fails := 0
	for {
		select {
		case <-stop:
			return
		default:
		}

		cmd := exec.Command(p.path, p.args...)
		cmd.Env = p.env
		cmd.SysProcAttr = childattr(p.drop)
		if p.stdin != "" {
			cmd.Stdin = strings.NewReader(p.stdin)
		} else {
			cmd.Stdin = nil
		}
		lf := openlog(p.name)
		if lf != nil {
			cmd.Stdout, cmd.Stderr = lf, lf
		}

		start := time.Now()
		if err := cmd.Start(); err != nil {
			if lf != nil {
				lf.Close()
			}
			c.fail("could not start " + p.name + ": " + err.Error())
			return
		}
		c.mu.Lock()
		if c.cmds != nil {
			c.cmds[p.name] = cmd
		}
		c.mu.Unlock()

		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			if lf != nil {
				lf.Close()
			}
			close(done)
		}()

		select {
		case <-stop:
			gracefulstop(cmd.Process, done)
			return
		case <-done:
		}

		select {
		case <-stop:
			return
		default:
		}

		if time.Since(start) < 2*time.Second {
			if fails++; fails >= 3 {
				c.failcode(p.name+" keeps exiting — check the config or a port conflict", "crashloop")
				return
			}
			logs.push(p.name, "error", "exited after "+time.Since(start).Truncate(time.Millisecond).String()+", restarting")
		} else {
			fails = 0
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *conn) fail(msg string) {
	c.failcode(msg, "")
}

func (c *conn) failcode(msg, code string) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}
	c.connected = false
	c.lasterr = msg
	c.lastcode = code
	stop := c.stop
	c.stop, c.cmds = nil, nil
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
}
