//go:build windows

package svc

import (
	"os"
	"os/exec"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

func configdir() (string, error) {
	return conf.Dir()
}

func psquote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func Elevate(args ...string) error {
	exe, err := exepath()
	if err != nil {
		return err
	}
	dir, err := conf.Dir()
	if err != nil {
		return err
	}
	var line []string
	for _, a := range append(args, "--dir", dir) {
		line = append(line, "\""+strings.ReplaceAll(a, "\"", "\\\"")+"\"")
	}
	script := "$p = Start-Process -FilePath " + psquote(exe) +
		" -ArgumentList " + psquote(strings.Join(line, " ")) +
		" -Verb RunAs -Wait -PassThru -WindowStyle Hidden; exit $p.ExitCode"
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
