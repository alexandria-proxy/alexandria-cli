package daemon

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
)

type state struct {
	mu   sync.Mutex
	subs []subscription.Subscription
	conn conn
}

func (s *state) findserver(url string, idx int) (subscription.Server, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].URL == url && idx >= 0 && idx < len(s.subs[i].Servers) {
			return s.subs[i].Servers[idx], true
		}
	}
	return subscription.Server{}, false
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
	subscription.Sort(subs)
	s := &state{subs: subs}

	ln, err := ipc.Listen(path, s.handle)
	if err != nil {
		return err
	}
	defer ln.Close()

	pid := pidpath()
	_ = os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0600)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	s.conn.stopnow()
	_ = os.Remove(pid)
	_ = os.Remove(path)
	return nil
}

func pidpath() string {
	if p, err := ipc.SocketPath(); err == nil {
		return filepath.Join(filepath.Dir(p), "control.pid")
	}
	return ""
}

func stopdaemon() {
	if data, err := os.ReadFile(pidpath()); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && n > 0 {
			if p, err := os.FindProcess(n); err == nil {
				_ = terminate(p)
			}
		}
	}
	for i := 0; i < 40; i++ {
		if !ipc.DaemonUp() {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if p, err := ipc.SocketPath(); err == nil {
		_ = os.Remove(p)
	}
}

func (s *state) handle(req ipc.Request) ipc.Response {
	switch req.Cmd {
	case "ping":
		return ipc.Response{OK: true, Version: ipc.ProtocolVersion}

	case "list":
		s.mu.Lock()
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "connect":
		srv, ok := s.findserver(req.URL, req.SrvIdx)
		if !ok {
			return ipc.Response{Error: "server not found"}
		}
		return s.conn.connect(srv, req.URL, req.SrvIdx, req.Mode)

	case "disconnect":
		return s.conn.disconnect()

	case "status":
		return s.conn.status()

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
				s.subs[i] = subscription.Merge(s.subs[i], sub)
				replaced = true
				break
			}
		}
		if !replaced {
			s.subs = append(s.subs, sub)
		}
		subscription.Sort(s.subs)
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "remove_subscription":
		s.mu.Lock()
		out := s.subs[:0]
		for _, sub := range s.subs {
			if sub.URL != req.URL {
				out = append(out, sub)
			}
		}
		s.subs = out
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "toggle_pin":
		s.mu.Lock()
		for i := range s.subs {
			if s.subs[i].URL == req.URL {
				s.subs[i].Pinned = !s.subs[i].Pinned
				break
			}
		}
		subscription.Sort(s.subs)
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "ping_subscription":
		s.mu.Lock()
		var servers []subscription.Server
		for i := range s.subs {
			if s.subs[i].URL == req.URL {
				servers = append([]subscription.Server(nil), s.subs[i].Servers...)
				break
			}
		}
		s.mu.Unlock()
		if servers == nil {
			return ipc.Response{Error: "subscription not found"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		probe := subscription.Subscription{Servers: servers}
		subscription.Ping(ctx, &probe)
		cancel()

		s.mu.Lock()
		for i := range s.subs {
			if s.subs[i].URL == req.URL {
				s.subs[i].Servers = probe.Servers
				break
			}
		}
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}

	case "update_server":
		s.mu.Lock()
		found := false
		for i := range s.subs {
			if s.subs[i].URL != req.URL {
				continue
			}
			if req.SrvIdx >= 0 && req.SrvIdx < len(s.subs[i].Servers) {
				s.subs[i].Servers[req.SrvIdx].Raw = req.Raw
				found = true
			}
			break
		}
		snapshot := append([]subscription.Subscription(nil), s.subs...)
		s.mu.Unlock()

		if !found {
			return ipc.Response{Error: "server not found"}
		}
		_ = subscription.SaveAll(snapshot)
		return ipc.Response{OK: true, Subscriptions: snapshot}
	}
	return ipc.Response{Error: "unknown command"}
}

func Ensure() error {
	if ipc.DaemonUp() {
		if resp, err := ipc.Send(ipc.Request{Cmd: "ping"}); err == nil && resp.Version == ipc.ProtocolVersion {
			return nil
		}
		stopdaemon()
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--daemon")
	cmd.SysProcAttr = detachattr()
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
