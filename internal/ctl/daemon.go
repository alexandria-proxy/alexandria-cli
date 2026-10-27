package ctl

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
)

var errrootsock = errors.New("a root daemon owns the control socket, " + hint + " once to take it over")

func Stop() {
	if p, err := ipc.Pidpath(); err == nil {
		if data, err := os.ReadFile(p); err == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && n > 0 {
				if proc, err := os.FindProcess(n); err == nil {
					_ = terminate(proc)
				}
			}
		}
	}
	for i := 0; i < 40 && ipc.Daemonup(); i++ {
		time.Sleep(25 * time.Millisecond)
	}
	if p, err := ipc.Socketpath(); err == nil {
		_ = os.Remove(p)
	}
}

func ensureuser() error {
	if ipc.Rootowned() && !conf.Elevated() {
		return errrootsock
	}
	if ipc.Daemonup() {
		resp, err := ipc.Send(ipc.Request{Cmd: ipc.Cmdping})
		if err == nil && resp.Version == ipc.Protocolversion {
			return nil
		}
		if err == nil && resp.Elevated && !conf.Elevated() {
			return errors.New("an older daemon is running as root, " + hint + " once to replace it")
		}
		Stop()
	}
	return spawn()
}

func ensureroot() error {
	if ipc.Rootowned() && !conf.Elevated() {
		return errrootsock
	}
	up := ipc.Daemonup()
	if up {
		if resp, err := ipc.Send(ipc.Request{Cmd: ipc.Cmdping}); err == nil && resp.Version == ipc.Protocolversion && resp.Elevated {
			return nil
		}
	}
	if !conf.Elevated() {
		return errors.New("tun mode needs root, " + hint)
	}
	if up {
		Stop()
	}
	return spawn()
}

func spawn() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "ALEXANDRIA_DAEMON=1")
	cmd.SysProcAttr = detachattr()
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	for i := 0; i < 60; i++ {
		if ipc.Daemonup() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("the daemon did not start")
}
