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
	name string
	path string
	args []string
	env  []string
}

type conn struct {
	mu        sync.Mutex
	wg        sync.WaitGroup
	cmds      map[string]*exec.Cmd
	connected bool
	url       string
	srvidx    int
	mode      string
	lasterr   string
	stop      chan struct{}
}

func cfgfile(name string) (string, error) {
	p, err := ipc.SocketPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), name), nil
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

func (c *conn) status() ipc.Response {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := ipc.Response{OK: true, Connected: c.connected, Mode: c.mode, Error: c.lasterr}
	if c.connected {
		r.ActiveURL, r.ActiveSrv = c.url, c.srvidx
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
		return ipc.Response{Error: "this server's protocol isn't supported yet: " + err.Error()}
	}
	xpath, err := xray.Ensure()
	if err != nil {
		return ipc.Response{Error: "xray core not found"}
	}
	xcfg, err := cfgfile("active.json")
	if err != nil {
		return ipc.Response{Error: err.Error()}
	}
	if err := os.WriteFile(xcfg, []byte(cfg), 0600); err != nil {
		return ipc.Response{Error: err.Error()}
	}

	procs := []proc{{
		name: "xray",
		path: xpath,
		args: []string{"run", "-c", xcfg},
		env:  append(os.Environ(), "XRAY_LOCATION_ASSET="+filepath.Dir(xpath)),
	}}

	if mode == "tun" {
		if !iselevated() {
			return ipc.Response{Error: "tun mode needs root — run alexandria with sudo/doas"}
		}
		sbpath, err := xray.EnsureSingbox()
		if err != nil {
			return ipc.Response{Error: "sing-box (tun engine) not found"}
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

	return c.start(url, idx, mode, procs)
}

func (c *conn) start(url string, idx int, mode string, procs []proc) ipc.Response {
	c.stopnow()

	stop := make(chan struct{})
	c.mu.Lock()
	c.connected = true
	c.url, c.srvidx, c.mode = url, idx, mode
	c.lasterr = ""
	c.stop = stop
	c.cmds = make(map[string]*exec.Cmd, len(procs))
	c.mu.Unlock()

	c.wg.Add(len(procs))
	for _, p := range procs {
		go c.supervise(p, stop)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		ok, lasterr := c.connected, c.lasterr
		c.mu.Unlock()
		if !ok {
			if lasterr == "" {
				lasterr = "failed to start"
			}
			return ipc.Response{Error: lasterr}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return c.status()
}

func (c *conn) supervise(p proc, stop chan struct{}) {
	defer c.wg.Done()
	fails := 0
	for {
		cmd := exec.Command(p.path, p.args...)
		cmd.Env = p.env
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil

		start := time.Now()
		if err := cmd.Start(); err != nil {
			c.fail("could not start " + p.name + ": " + err.Error())
			return
		}
		c.mu.Lock()
		if c.cmds != nil {
			c.cmds[p.name] = cmd
		}
		c.mu.Unlock()

		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()

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
				c.fail(p.name + " keeps exiting — check the config or a port conflict")
				return
			}
		} else {
			fails = 0
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *conn) fail(msg string) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}
	c.connected = false
	c.lasterr = msg
	stop := c.stop
	c.stop, c.cmds = nil, nil
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
}
