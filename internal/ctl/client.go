package ctl

import (
	"errors"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

func Call(req ipc.Request) (ipc.Response, error) {
	return Calltimeout(req, 0)
}

func Calltimeout(req ipc.Request, d time.Duration) (ipc.Response, error) {
	var resp ipc.Response
	var err error
	if d > 0 {
		resp, err = ipc.Sendtimeout(req, d)
	} else {
		resp, err = ipc.Send(req)
	}
	if err != nil {
		return resp, errors.New("can't reach the daemon: " + err.Error())
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

func Up() bool {
	return ipc.Daemonup()
}

func Status() (ipc.Response, bool) {
	if !Up() {
		return ipc.Response{}, false
	}
	resp, err := ipc.Send(ipc.Request{Cmd: ipc.Cmdstatus})
	return resp, err == nil
}

func Subs() ([]sub.Subscription, error) {
	if Up() {
		if resp, err := Call(ipc.Request{Cmd: ipc.Cmdlist}); err == nil {
			return resp.Subscriptions, nil
		}
	}
	return sub.Load()
}

func Livesubs() ([]sub.Subscription, error) {
	if err := Ensure(); err != nil {
		return nil, err
	}
	resp, err := Call(ipc.Request{Cmd: ipc.Cmdlist})
	return resp.Subscriptions, err
}

func Current() (string, int) {
	if st, ok := Status(); ok && st.Connected {
		return st.Activeurl, st.Activesrv
	}
	cfg, _ := conf.Load()
	return cfg.Lasturl, cfg.Lastsrv
}
