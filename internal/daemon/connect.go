package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	"github.com/alexandria-proxy/alexandria-cli/internal/xray"
)

type conn struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	connected bool
	url       string
	srvidx    int
	mode      string
	lasterr   string
	stop      chan struct{}
}

func activeconfigpath() (string, error) {
	p, err := ipc.SocketPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "active.json"), nil
}

func isxrayjson(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), "{")
}

func (c *conn) status() ipc.Response {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ipc.Response{OK: true, Connected: c.connected, Mode: c.mode, Error: c.lasterr}
}

func (c *conn) disconnect() ipc.Response {
	c.stopnow()
	return ipc.Response{OK: true, Connected: false}
}

func (c *conn) stopnow() {
	c.mu.Lock()
	stop, cmd := c.stop, c.cmd
	c.stop, c.cmd = nil, nil
	c.connected = false
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (c *conn) connect(srv subscription.Server, url string, idx int, mode string) ipc.Response {
	if mode == "tun" {
		return ipc.Response{Error: "tun mode is not implemented yet"}
	}
	if !isxrayjson(srv.Raw) {
		return ipc.Response{Error: "this server has no full config to connect with yet"}
	}
	xpath, err := xray.Ensure()
	if err != nil {
		return ipc.Response{Error: "xray core not found"}
	}
	cfgpath, err := activeconfigpath()
	if err != nil {
		return ipc.Response{Error: err.Error()}
	}
	if err := os.WriteFile(cfgpath, []byte(srv.Raw), 0600); err != nil {
		return ipc.Response{Error: err.Error()}
	}

	c.stopnow()

	stop := make(chan struct{})
	c.mu.Lock()
	c.connected = true
	c.url, c.srvidx, c.mode = url, idx, mode
	c.lasterr = ""
	c.stop = stop
	c.mu.Unlock()

	go c.supervise(xpath, cfgpath, stop)
	return c.status()
}

func (c *conn) supervise(xpath, cfgpath string, stop chan struct{}) {
	asset := filepath.Dir(xpath)
	fails := 0
	for {
		cmd := exec.Command(xpath, "run", "-c", cfgpath)
		cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+asset)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil

		start := time.Now()
		if err := cmd.Start(); err != nil {
			c.fail("could not start xray: " + err.Error())
			return
		}
		c.mu.Lock()
		c.cmd = cmd
		c.mu.Unlock()

		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()

		select {
		case <-stop:
			_ = cmd.Process.Kill()
			<-done
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
				c.fail("xray keeps exiting — check the server config")
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
	c.connected = false
	c.lasterr = msg
	c.stop, c.cmd = nil, nil
	c.mu.Unlock()
}
