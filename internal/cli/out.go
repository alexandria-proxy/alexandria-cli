package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	red    = "31"
	green  = "32"
	yellow = "33"
	dim    = "2"
	bold   = "1"
)

var (
	stdout    io.Writer = os.Stdout
	stderr    io.Writer = os.Stderr
	colorwhen           = "auto"
	failed    bool
)

type usageerr struct{ msg string }

func (e usageerr) Error() string { return e.msg }

type exiterr struct{ code int }

func (e exiterr) Error() string { return fmt.Sprintf("exit status %d", e.code) }

func usagef(format string, a ...any) error {
	return usageerr{fmt.Sprintf(format, a...)}
}

func isterm(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func colorfor(f *os.File) bool {
	switch colorwhen {
	case "always":
		return true
	case "never":
		return false
	}
	return os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && isterm(f)
}

func paint(code, s string) string {
	if s == "" || code == "" || !colorfor(os.Stdout) {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func tag(code, name string) string {
	if !colorfor(os.Stderr) {
		return name + ":"
	}
	return "\x1b[" + code + "m" + name + ":\x1b[0m"
}

func errf(format string, a ...any) {
	failed = true
	fmt.Fprintln(stderr, tag(red, "err"), fmt.Sprintf(format, a...))
}

func warnf(format string, a ...any) {
	fmt.Fprintln(stderr, tag(yellow, "warn"), fmt.Sprintf(format, a...))
}

func infof(format string, a ...any) {
	fmt.Fprintln(stderr, tag(dim, "info"), fmt.Sprintf(format, a...))
}

func say(format string, a ...any) {
	fmt.Fprintf(stdout, format+"\n", a...)
}

func printjson(v any) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func width(s string) int {
	return runewidth.StringWidth(s)
}

func pad(s string, w int) string {
	if d := w - width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func padpaint(code, s string, w int) string {
	return paint(code, s) + strings.Repeat(" ", max(0, w-width(s)))
}

func padleft(s string, w int) string {
	if d := w - width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}
