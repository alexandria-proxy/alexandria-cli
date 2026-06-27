package ipc

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/subscription"
)

const ProtocolVersion = 7

type Request struct {
	Cmd    string `json:"cmd"`
	URL    string `json:"url,omitempty"`
	SrvIdx int    `json:"srv_idx,omitempty"`
	Raw    string `json:"raw,omitempty"`
	Mode   string `json:"mode,omitempty"`
}

type Response struct {
	OK            bool                        `json:"ok"`
	Error         string                      `json:"error,omitempty"`
	Version       int                         `json:"version,omitempty"`
	Connected     bool                        `json:"connected,omitempty"`
	Mode          string                      `json:"mode,omitempty"`
	ActiveURL     string                      `json:"active_url,omitempty"`
	ActiveSrv     int                         `json:"active_srv,omitempty"`
	Subscriptions []subscription.Subscription `json:"subscriptions,omitempty"`
}

type Handler func(Request) Response

func SocketPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria", "control.sock"), nil
}

func Listen(path string, h Handler) (net.Listener, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0600)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveconn(conn, h)
		}
	}()
	return ln, nil
}

func serveconn(conn net.Conn, h Handler) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	var req Request
	if json.NewDecoder(conn).Decode(&req) != nil {
		return
	}
	_ = json.NewEncoder(conn).Encode(h(req))
}

func Send(req Request) (Response, error) {
	path, err := SocketPath()
	if err != nil {
		return Response{}, err
	}
	conn, err := net.DialTimeout("unix", path, 3*time.Second)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func DaemonUp() bool {
	path, err := SocketPath()
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
