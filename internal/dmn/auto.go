package dmn

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

func fetchopts(s conf.Settings) sub.Fetchopts {
	return sub.Fetchopts{
		Useragent: s.Subs.Useragent,
		Timeout:   time.Duration(s.Subs.Timeoutsec) * time.Second,
		Sendhwid:  s.Subs.Sendhwid,
	}
}

func fetchwindow(s conf.Settings) time.Duration {
	d := time.Duration(s.Subs.Timeoutsec)*time.Second + 3*time.Second
	if d < 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (s *state) findserver(url string, idx int) (sub.Server, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].Url == url && idx >= 0 && idx < len(s.subs[i].Servers) {
			return s.subs[i].Servers[idx], true
		}
	}
	return sub.Server{}, false
}

func (s *state) autoconnect() {
	cfg, err := conf.Load()
	if err != nil {
		return
	}
	mode := cfg.Mode
	if mode != "tun" {
		mode = "proxy"
	}
	url := cfg.Lasturl
	if url == "" {
		url = s.firstsub()
	}
	if url == "" {
		return
	}

	mine := int64(0)
	deadline := time.Now().Add(bootwindow)
	for {
		s.touch()
		if st := s.conn.state(); st.live && st.gen != mine {
			return
		}
		s.refreshsub(url)
		ok, gen := s.connectfastest(url, mode, mine)
		mine = gen
		if ok {
			return
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-s.quit:
			return
		case <-time.After(bootretry):
		}
	}

	if st := s.conn.state(); st.live && st.gen != mine {
		return
	}
	if srv, ok := s.findserver(url, cfg.Lastsrv); ok {
		s.conn.connect(srv, url, cfg.Lastsrv, mode)
	}
}

func (s *state) firstsub() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.subs) == 0 {
		return ""
	}
	return s.subs[0].Url
}

func (s *state) refreshsub(url string) {
	set := usersettings()
	ctx, cancel := context.WithTimeout(context.Background(), fetchgrace)
	defer cancel()

	got, err := sub.Fetch(ctx, url, fetchopts(set))
	if err != nil {
		return
	}

	s.mu.Lock()
	for i := range s.subs {
		if s.subs[i].Url == got.Url {
			s.subs[i] = sub.Merge(s.subs[i], got)
			break
		}
	}
	snapshot := append([]sub.Subscription(nil), s.subs...)
	s.mu.Unlock()

	_ = sub.Saveall(snapshot)
}

func (s *state) connectfastest(url, mode string, mine int64) (bool, int64) {
	order := s.pingorder(url)
	tries := 0
	for _, idx := range order {
		if st := s.conn.state(); st.live && st.gen != mine {
			return true, mine
		}
		srv, ok := s.findserver(url, idx)
		if !ok {
			continue
		}
		if resp := s.conn.connect(srv, url, idx, mode); resp.Error == "" {
			st := s.conn.state()
			mine = st.gen
			if healthy(st.mode, st.socks) {
				savelast(url, idx)
				return true, mine
			}
			logs.push("daemon", "warn", srv.Name+" came up but passed no traffic — trying the next one")
		}
		s.touch()
		if tries++; tries >= fastesttries {
			break
		}
	}
	return false, mine
}

func (s *state) pingorder(url string) []int {
	s.mu.Lock()
	var servers []sub.Server
	for i := range s.subs {
		if s.subs[i].Url == url {
			servers = append([]sub.Server(nil), s.subs[i].Servers...)
			break
		}
	}
	s.mu.Unlock()
	if len(servers) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), pinggrace)
	probe := sub.Subscription{Servers: servers}
	sub.Ping(ctx, &probe, usersettings().Pingproto)
	cancel()

	s.mu.Lock()
	for i := range s.subs {
		if s.subs[i].Url == url && len(s.subs[i].Servers) == len(probe.Servers) {
			s.subs[i].Servers = probe.Servers
			break
		}
	}
	snapshot := append([]sub.Subscription(nil), s.subs...)
	s.mu.Unlock()

	_ = sub.Saveall(snapshot)

	var order []int
	for i := range probe.Servers {
		if probe.Servers[i].Pingms > 0 {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool {
		return probe.Servers[order[a]].Pingms < probe.Servers[order[b]].Pingms
	})
	return order
}

func (s *state) connectbest(url, mode string) ipc.Response {
	if !s.hassub(url) {
		return ipc.Response{Error: "subscription not found"}
	}
	if ok, _ := s.connectfastest(url, mode, s.conn.state().gen); !ok {
		resp := s.conn.status()
		msg := "no working server found in this subscription"
		if resp.Error != "" {
			msg += " (" + resp.Error + ")"
		}
		return ipc.Response{Error: msg, Code: resp.Code}
	}
	return s.conn.status()
}

func (s *state) hassub(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].Url == url {
			return true
		}
	}
	return false
}

func savelast(url string, idx int) {
	cfg, err := conf.Load()
	if err != nil || cfg.Lasturl == "" {
		return
	}
	if cfg.Lasturl == url && cfg.Lastsrv == idx {
		return
	}
	cfg.Lasturl, cfg.Lastsrv = url, idx
	_ = conf.Save(cfg)
}

func (s *state) autoupdatewatch() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	last := time.Time{}
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
		}
		set := usersettings()
		if !set.Subs.Auto {
			continue
		}
		s.touch()
		every := time.Duration(set.Subs.Intervalh) * time.Hour
		if time.Since(last) < every {
			continue
		}
		last = time.Now()
		s.updateall()
	}
}

func (s *state) updateall() {
	s.mu.Lock()
	urls := make([]string, 0, len(s.subs))
	for i := range s.subs {
		urls = append(urls, s.subs[i].Url)
	}
	s.mu.Unlock()

	for _, u := range urls {
		s.refreshsub(u)
	}
	if len(urls) > 0 {
		logevent("auto update finished · " + strconv.Itoa(len(urls)) + " subscription(s)")
	}
}
