//go:build unix

package svc

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

func Elevate(args ...string) error {
	exe, err := exepath()
	if err != nil {
		return err
	}
	dir, err := conf.Dir()
	if err != nil {
		return err
	}
	full := append(append([]string{exe}, args...), "--dir", dir)
	for _, tool := range []string{"sudo", "doas", "pkexec"} {
		bin, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, full...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}
	return errors.New("no sudo, doas or pkexec here, run as root: " + strings.Join(full, " "))
}
