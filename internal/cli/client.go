package cli

import (
	"errors"
	"os/exec"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
)

const besttimeout = 90 * time.Second

func elevated(err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return exiterr{ee.ExitCode()}
	}
	return err
}

func installsvc() error {
	if !conf.Elevated() {
		infof("installing the service")
	}
	return elevated(ctl.Install())
}

func ensurefor(mode string) error {
	if ctl.Needinstall() {
		if err := installsvc(); err != nil {
			return err
		}
	}
	return ctl.Ensurefor(mode)
}

func status() (ipc.Response, bool) {
	if ipc.Rootowned() {
		warnf("the daemon socket belongs to root, use sudo once")
		return ipc.Response{}, false
	}
	return ctl.Status()
}

func loadcfg() (conf.Config, error) {
	cfg, err := conf.Load()
	if err != nil {
		return cfg, errors.New("config.json is broken: " + err.Error())
	}
	return cfg, nil
}
