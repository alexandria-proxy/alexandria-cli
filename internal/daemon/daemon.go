package daemon

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
	"github.com/alexandria-proxy/alexandria-cli/internal/xray"
)

type state struct {
	mu   sync.Mutex
	subs []subscription.Subscription
}

func Run() error {
	path, err := ipc.SocketPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	subs, _ := subscription.Load()
	s := &state{subs: subs}

	ln, err := ipc.Listen(path, s.handle)
	if err != nil {
		return err
	}
	defer ln.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	_ = os.Remove(path)
	return nil
}

func (s *state) handle(req ipc.Request) ipc.Response {
	switch req.Cmd {
	case "ping":
		return ipc.Response{OK: true}

	case "list":
		s.mu.Lock()
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "ensure_core":
		path, err := xray.Ensure()
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		return ipc.Response{OK: true, Path: path}

	case "add_subscription":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sub, err := subscription.Fetch(ctx, req.URL)
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}

		s.mu.Lock()
		replaced := false
		for i := range s.subs {
			if s.subs[i].URL == sub.URL {
				s.subs[i] = sub
				replaced = true
				break
			}
		}
		if !replaced {
			s.subs = append(s.subs, sub)
		}
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}
	}
	return ipc.Response{Error: "unknown command"}
}

func Ensure() error {
	if ipc.DaemonUp() {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--daemon")
	cmd.SysProcAttr = detachAttr()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()

	for i := 0; i < 60; i++ {
		if ipc.DaemonUp() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("daemon did not start")
}
