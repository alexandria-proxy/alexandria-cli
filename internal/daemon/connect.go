package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/singbox"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	"github.com/alexandria-proxy/alexandria-cli/internal/xray"
	"github.com/alexandria-proxy/alexandria-cli/internal/xraygen"
)

type proc struct {
	name  string
	path  string
	args  []string
	env   []string
	stdin string
}

type conn struct {
	mu         sync.Mutex
	wg         sync.WaitGroup
	cmds       map[string]*exec.Cmd
	connected  bool
	restarting bool
	url        string
	srvidx     int
	mode       string
	since      time.Time
	lasterr    string
	lastcode   string
	metrics    int
	stop       chan struct{}
	stats      stattrack
}

func cfgfile(name string) (string, error) {
	p, err := ipc.SocketPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), name), nil
}

const logcap = 200 << 20

type caplog struct {
	f   *os.File
	max int64
	n   int64
}

func (w *caplog) Write(p []byte) (int, error) {
	if w.n+int64(len(p)) > w.max {
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
	for _, name := range []string{"xray.log", "sing-box.log", "active.json"} {
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
	n := int64(0)
	if fi, err := f.Stat(); err == nil {
		n = fi.Size()
	}
	return &caplog{f: f, max: logcap, n: n}
}

func isxrayjson(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), "{")
}

func buildxray(raw string) (string, error) {
	if isxrayjson(raw) {
		return raw, nil
	}
	return xraygen.Build(raw)
}

func (c *conn) isconnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *conn) setrestarting(v bool) {
	c.mu.Lock()
	c.restarting = v
	c.mu.Unlock()
}

func (c *conn) status() ipc.Response {
	c.mu.Lock()
	live := c.connected || c.restarting
	r := ipc.Response{OK: true, Connected: live, Mode: c.mode, Error: c.lasterr, Code: c.lastcode}
	if live {
		r.ActiveURL, r.ActiveSrv = c.url, c.srvidx
		r.Since = c.since.Unix()
	}
	metrics := c.metrics
	c.mu.Unlock()

	if live && metrics > 0 {
		s := c.stats.snapshot()
		r.HasStats = true
		r.UpTotal, r.DownTotal = s.up, s.down
		r.UpRate, r.DownRate = s.uprate, s.downrate
	}
	return r
}

func (c *conn) disconnect() ipc.Response {
	c.stopnow()
	return ipc.Response{OK: true, Connected: false}
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

	tuncleanup(singbox.TunName)
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

func (c *conn) connect(srv subscription.Server, url string, idx int, mode string) ipc.Response {
	cfg, err := buildxray(srv.Raw)
	if err != nil {
		return ipc.Response{Error: "this server's protocol isn't supported yet: " + err.Error(), Code: "unsupported"}
	}
	xpath, err := xray.Ensure()
	if err != nil {
		return ipc.Response{Error: "xray core not found", Code: "nocore"}
	}
	procs := []proc{{
		name:  "xray",
		path:  xpath,
		args:  []string{"run", "-c", "stdin:"},
		env:   append(os.Environ(), "XRAY_LOCATION_ASSET="+filepath.Dir(xpath)),
		stdin: cfg,
	}}

	if mode == "tun" {
		if !iselevated() {
			return ipc.Response{Error: "tun mode needs elevated privileges — " + elevatehint(), Code: "needelevate"}
		}
		sbpath, err := xray.EnsureSingbox()
		if err != nil {
			return ipc.Response{Error: "sing-box (tun engine) not found", Code: "nosingbox"}
		}
		tuncfg, err := cfgfile("tun.json")
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		if err := os.WriteFile(tuncfg, []byte(singbox.Config(singbox.SocksPort(cfg))), 0600); err != nil {
			return ipc.Response{Error: err.Error()}
		}
		procs = append(procs, proc{
			name: "sing-box",
			path: sbpath,
			args: []string{"run", "-c", tuncfg},
			env:  os.Environ(),
		})
	}

	return c.start(url, idx, mode, procs, xraygen.MetricsPort(cfg))
}

func (c *conn) start(url string, idx int, mode string, procs []proc, metrics int) ipc.Response {
	c.stopnow()

	stop := make(chan struct{})
	c.mu.Lock()
	c.connected = true
	c.url, c.srvidx, c.mode = url, idx, mode
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
		cmd.SysProcAttr = childattr()
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
