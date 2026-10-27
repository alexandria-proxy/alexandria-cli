package dmn

import (
	"strings"
	"sync"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
)

const logring = 2000

var logs = newring()

func newring() *ringbuf {
	return &ringbuf{set: conf.Defaults().Settings.Logs}
}

type ringbuf struct {
	mu    sync.Mutex
	lines []ipc.Logline
	seq   int64
	set   conf.Logs
}

func (r *ringbuf) configure(set conf.Logs) {
	r.mu.Lock()
	r.set = set
	r.mu.Unlock()
}

func (r *ringbuf) settings() conf.Logs {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.set
}

func (r *ringbuf) wants(src string) bool {
	s := r.settings()
	if !s.On {
		return false
	}
	switch src {
	case "xray":
		return s.Xray
	case "sing-box":
		return s.Singbox
	}
	return s.Daemon
}

func (r *ringbuf) push(src, lvl, text string) {
	text = strings.TrimRight(text, "\r\n\t ")
	if text == "" || !r.wants(src) {
		return
	}
	r.mu.Lock()
	r.seq++
	r.lines = append(r.lines, ipc.Logline{
		At:   time.Now().Unix(),
		Seq:  r.seq,
		Src:  src,
		Lvl:  lvl,
		Text: text,
	})
	if n := len(r.lines) - logring; n > 0 {
		r.lines = append(r.lines[:0], r.lines[n:]...)
	}
	r.mu.Unlock()
}

func (r *ringbuf) since(seq int64) ([]ipc.Logline, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ipc.Logline, 0, len(r.lines))
	for _, l := range r.lines {
		if l.Seq > seq {
			out = append(out, l)
		}
	}
	return out, r.seq
}

func (r *ringbuf) reset() {
	r.mu.Lock()
	r.lines = nil
	r.mu.Unlock()
}

var coretags = []struct{ tag, lvl string }{
	{"[error]", "error"}, {"[warning]", "warn"}, {"[info]", "info"}, {"[debug]", "info"},
	{" fatal[", "error"}, {" error[", "error"}, {" warn[", "warn"}, {" info[", "info"}, {" debug[", "info"},
}

func level(text string) string {
	low := strings.ToLower(text)
	for _, t := range coretags {
		if strings.Contains(low, t.tag) {
			return t.lvl
		}
	}
	switch {
	case strings.Contains(low, "error") || strings.Contains(low, "failed") || strings.Contains(low, "fatal"):
		return "error"
	case strings.Contains(low, "warn"):
		return "warn"
	}
	return "info"
}

func logevent(text string) {
	logs.push("daemon", "info", text)
}

func logerr(text string) {
	logs.push("daemon", "error", text)
}

func logcore(src string, chunk []byte) {
	for _, line := range strings.Split(string(chunk), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		logs.push(src, level(line), line)
	}
}
