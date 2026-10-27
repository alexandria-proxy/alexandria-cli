package ctl

import (
	"errors"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/svc"
)

func Needinstall() bool {
	return svc.Supported() && !svc.Installed()
}

func Install() error {
	if conf.Elevated() {
		Stop()
		if err := svc.Install(); err != nil {
			return err
		}
		return Ready()
	}
	if _, err := conf.Mkdir(); err != nil {
		return err
	}
	if err := svc.Elevate("svc", "on"); err != nil {
		return err
	}
	if !svc.Installed() {
		return errors.New("the service didn't get installed")
	}
	return nil
}

func Uninstall() error {
	if !svc.Installed() {
		return nil
	}
	if !conf.Elevated() {
		return svc.Elevate("svc", "off")
	}
	_, _ = ipc.Send(ipc.Request{Cmd: ipc.Cmddisconnect})
	_, _ = ipc.Send(ipc.Request{Cmd: ipc.Cmdshutdown})
	return svc.Uninstall()
}

func current() bool {
	resp, err := ipc.Send(ipc.Request{Cmd: ipc.Cmdping})
	return err == nil && resp.Version == ipc.Protocolversion && resp.Build == ipc.Appversion
}

func Ready() error {
	if ipc.Daemonup() {
		if current() {
			return nil
		}
		_, _ = ipc.Send(ipc.Request{Cmd: ipc.Cmddisconnect})
		_, _ = ipc.Send(ipc.Request{Cmd: ipc.Cmdshutdown})
		for i := 0; i < 50 && ipc.Daemonup(); i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	_ = svc.Start()
	for i := 0; i < 150; i++ {
		if ipc.Daemonup() && current() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("the alexandria service isn't answering, " + svc.Starthint)
}

func Ensure() error {
	if svc.Installed() {
		return Ready()
	}
	if err := ensureuser(); err != nil {
		return errors.New("can't start the daemon: " + err.Error())
	}
	return nil
}

func Ensurefor(mode string) error {
	if svc.Supported() {
		return Ready()
	}
	if mode == "tun" {
		return ensureroot()
	}
	return ensureuser()
}
