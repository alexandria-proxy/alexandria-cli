package dmn

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/route"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

const (
	idletimeout  = 60 * time.Second
	waketick     = 5 * time.Second
	wakegap      = 25 * time.Second
	waketries    = 5
	wakedelay    = 2 * time.Second
	wakedelaycap = 15 * time.Second

	healthtick    = 10 * time.Second
	healthgrace   = 8 * time.Second
	healthstrikes = 2
	repaircool    = 45 * time.Second
	repaircap     = 5 * time.Minute

	bootwindow   = 3 * time.Minute
	bootretry    = 5 * time.Second
	fetchgrace   = 20 * time.Second
	pinggrace    = 10 * time.Second
	fastesttries = 3

	routetick = 2 * time.Second
)

type state struct {
	mu       sync.Mutex
	subs     []sub.Subscription
	conn     conn
	quit     chan struct{}
	quitonce sync.Once
	lastbeat time.Time
}

func usersettings() conf.Settings {
	c, err := conf.Load()
	if err != nil {
		return conf.Defaults().Settings
	}
	logs.configure(c.Settings.Logs)
	return c.Settings
}

func (s *state) shutdown() {
	time.Sleep(150 * time.Millisecond)
	s.quitonce.Do(func() { close(s.quit) })
}

func (s *state) touch() {
	s.mu.Lock()
	s.lastbeat = time.Now()
	s.mu.Unlock()
}

func (s *state) idlewatch() {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
			if s.conn.isconnected() {
				continue
			}
			s.mu.Lock()
			idle := time.Since(s.lastbeat)
			s.mu.Unlock()
			if idle > idletimeout {
				s.quitonce.Do(func() { close(s.quit) })
				return
			}
		}
	}
}

func (s *state) wakewatch() {
	last := time.Now().Round(0)
	t := time.NewTicker(waketick)
	defer t.Stop()
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
			now := time.Now().Round(0)
			gap := now.Sub(last)
			last = now
			if gap > wakegap {
				s.wake()
			}
		}
	}
}

func (s *state) wake() {
	st := s.conn.state()
	if !st.live {
		return
	}
	s.conn.setrestarting(true)
	defer s.conn.setrestarting(false)

	logevent("system resumed — restoring the tunnel")
	waituplink(s.quit, uplinkbudget)
	if !s.revive(st, waketries) {
		logs.push("daemon", "warn", "tunnel is still not passing traffic after resume")
	}
}

func (s *state) revive(st connstate, tries int) bool {
	srv, ok := s.findserver(st.url, st.idx)
	if !ok {
		return false
	}
	gen := st.gen
	delay := wakedelay
	for i := 0; i < tries; i++ {
		if i > 0 {
			select {
			case <-s.quit:
				return false
			case <-time.After(delay):
			}
			if delay *= 2; delay > wakedelaycap {
				delay = wakedelaycap
			}
		}
		if s.conn.superseded(gen) {
			return true
		}
		if resp := s.conn.connect(srv, st.url, st.idx, st.mode); resp.Error != "" {
			continue
		}
		now := s.conn.state()
		gen = now.gen
		if healthy(now.mode, now.socks) {
			return true
		}
	}
	return false
}

func (s *state) routewatch() {
	t := time.NewTicker(routetick)
	defer t.Stop()
	last := routestamp()
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
		}
		now := routestamp()
		if now == last {
			continue
		}
		last = now
		s.conn.reloadtun()
	}
}

func routestamp() string {
	var b strings.Builder
	for _, name := range []string{route.Name, route.Custom} {
		p, err := cfgfile(name)
		if err != nil {
			continue
		}
		if fi, err := os.Stat(p); err == nil {
			b.WriteString(strconv.FormatInt(fi.ModTime().UnixNano(), 10) + ":" + strconv.FormatInt(fi.Size(), 10) + ";")
		}
	}
	return b.String()
}

func Run(autoconnect, service bool) error {
	setprocname("alexad")

	path, err := ipc.Socketpath()
	if err != nil {
		return err
	}
	if _, err := conf.Mkdir(); err != nil {
		return err
	}
	_ = route.Migrate()

	subs, _ := sub.Load()
	s := &state{subs: subs, quit: make(chan struct{}), lastbeat: time.Now()}

	ln, err := ipc.Listen(path, s.handle)
	if err != nil {
		return err
	}
	defer ln.Close()

	logs.configure(usersettings().Logs)
	logevent("daemon started")

	if service {
		if cfg, err := conf.Load(); err == nil && cfg.Autostart {
			autoconnect = true
		}
	} else {
		go s.idlewatch()
	}
	go s.wakewatch()
	go s.healthwatch()
	go s.autoupdatewatch()
	go s.routewatch()

	if autoconnect {
		go s.autoconnect()
	}

	pid, _ := ipc.Pidpath()
	_ = os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0600)
	conf.Own(pid)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
	case <-s.quit:
	}

	logevent("daemon stopping")
	s.conn.stopop()
	cleanlogs()
	_ = os.Remove(pid)
	_ = os.Remove(path)
	return nil
}
