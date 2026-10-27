package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

const Protocolversion = 12

var Appversion = "dev"

const (
	Cmdping       = "ping"
	Cmdlist       = "list"
	Cmdconnect    = "connect"
	Cmdbest       = "connect_best"
	Cmddisconnect = "disconnect"
	Cmdstatus     = "status"
	Cmdshutdown   = "shutdown"
	Cmdlogs       = "logs"
	Cmdreset      = "reset"
	Cmdadd        = "add_subscription"
	Cmdremove     = "remove_subscription"
	Cmdprobe      = "ping_subscription"
)

const (
	sendtimeout  = 30 * time.Second
	servetimeout = 2 * time.Minute
)

type Request struct {
	Cmd    string `json:"cmd"`
	Url    string `json:"url,omitempty"`
	Srvidx int    `json:"srv_idx,omitempty"`
	Mode   string `json:"mode,omitempty"`
	Since  int64  `json:"since,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

type Logline struct {
	At   int64  `json:"at"`
	Seq  int64  `json:"seq"`
	Src  string `json:"src"`
	Lvl  string `json:"lvl"`
	Text string `json:"text"`
}

type Response struct {
	Ok            bool               `json:"ok"`
	Error         string             `json:"error,omitempty"`
	Version       int                `json:"version,omitempty"`
	Build         string             `json:"build,omitempty"`
	Elevated      bool               `json:"elevated,omitempty"`
	Connected     bool               `json:"connected,omitempty"`
	Mode          string             `json:"mode,omitempty"`
	Activeurl     string             `json:"active_url,omitempty"`
	Activesrv     int                `json:"active_srv,omitempty"`
	Since         int64              `json:"since,omitempty"`
	Code          string             `json:"code,omitempty"`
	Hasstats      bool               `json:"has_stats,omitempty"`
	Uptotal       int64              `json:"up_total,omitempty"`
	Downtotal     int64              `json:"down_total,omitempty"`
	Uprate        int64              `json:"up_rate,omitempty"`
	Downrate      int64              `json:"down_rate,omitempty"`
	Subscriptions []sub.Subscription `json:"subscriptions,omitempty"`
	Logs          []Logline          `json:"logs,omitempty"`
	Seq           int64              `json:"seq,omitempty"`
}

type Handler func(Request) Response

func Socketpath() (string, error) {
	return conf.File("control.sock")
}

func Pidpath() (string, error) {
	return conf.File("control.pid")
}

func Listen(path string, h Handler) (net.Listener, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0600)
	conf.Own(path)
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
	_ = conn.SetDeadline(time.Now().Add(servetimeout))
	var req Request
	if json.NewDecoder(conn).Decode(&req) != nil {
		return
	}
	_ = json.NewEncoder(conn).Encode(h(req))
}

func Send(req Request) (Response, error) {
	return Sendtimeout(req, sendtimeout)
}

func Sendtimeout(req Request, d time.Duration) (Response, error) {
	path, err := Socketpath()
	if err != nil {
		return Response{}, err
	}
	conn, err := net.DialTimeout("unix", path, 3*time.Second)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(d))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func Reach() error {
	path, err := Socketpath()
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}

func Daemonup() bool {
	return Reach() == nil
}

func Rootowned() bool {
	return errors.Is(Reach(), os.ErrPermission)
}
