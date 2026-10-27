//go:build darwin

package svc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

const (
	plabel    = "org." + label + ".daemon"
	plistpath = "/Library/LaunchDaemons/" + plabel + ".plist"
	Starthint = "try: sudo launchctl kickstart -k system/" + plabel
)

func Supported() bool { return true }

func Installed() bool {
	b, err := os.ReadFile(plistpath)
	return err == nil && strings.Contains(string(b), "--service")
}

func escape(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

func plistdoc(exe, dir, home string) string {
	var args strings.Builder
	for _, a := range []string{exe, "--daemon", "--service", "--dir", dir} {
		args.WriteString("\t\t<string>" + escape(a) + "</string>\n")
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + plabel + `</string>
	<key>ProgramArguments</key>
	<array>
` + args.String() + `	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>HOME</key>
		<string>` + escape(home) + `</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`
}

func install(exe, dir, home string) error {
	if err := os.WriteFile(plistpath, []byte(plistdoc(exe, dir, home)), 0644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "bootout", "system/"+plabel).Run()
	if out, err := exec.Command("launchctl", "bootstrap", "system", plistpath).CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

func uninstall() error {
	_ = exec.Command("launchctl", "bootout", "system/"+plabel).Run()
	if err := os.Remove(plistpath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Start() error { return nil }

func legacy(dir, home string) {
	agent := filepath.Join(home, "Library", "LaunchAgents", plabel+".plist")
	if _, err := os.Stat(agent); err != nil {
		return
	}
	if uid, _, ok := conf.Owner(); ok {
		_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(uid)+"/"+plabel).Run()
	}
	_ = os.Remove(agent)
}
