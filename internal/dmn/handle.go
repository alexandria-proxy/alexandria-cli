package dmn

import (
	"context"
	"os"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/gen"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/route"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

func (s *state) reset(kind string) ipc.Response {
	switch kind {
	case "tun":
		s.conn.stopop()
		tuncleanup(gen.Tunname)
		if p, err := cfgfile("tun.json"); err == nil {
			_ = os.Remove(p)
		}
		logevent("tunnel configuration reset")
		return ipc.Response{Ok: true}

	case "settings":
		c, _ := conf.Load()
		d := conf.Defaults()
		d.Mode, d.Lasturl, d.Lastsrv = c.Mode, c.Lasturl, c.Lastsrv
		if err := conf.Save(d); err != nil {
			return ipc.Response{Error: err.Error()}
		}
		logevent("settings reset to defaults")
		return ipc.Response{Ok: true}

	case "user":
		s.conn.stopop()
		s.mu.Lock()
		s.subs = nil
		s.mu.Unlock()
		_ = sub.Saveall(nil)
		for _, name := range []string{route.Name, route.Custom} {
			if p, err := cfgfile(name); err == nil {
				_ = os.Remove(p)
			}
		}
		if err := conf.Save(conf.Defaults()); err != nil {
			return ipc.Response{Error: err.Error()}
		}
		logs.reset()
		logevent("user data reset")
		return ipc.Response{Ok: true, Subscriptions: nil}
	}
	return ipc.Response{Error: "unknown reset kind"}
}

func (s *state) handle(req ipc.Request) ipc.Response {
	s.touch()
	switch req.Cmd {
	case ipc.Cmdping:
		return ipc.Response{Ok: true, Version: ipc.Protocolversion, Build: ipc.Appversion, Elevated: conf.Elevated()}

	case ipc.Cmdlist:
		s.mu.Lock()
		snapshot := append([]sub.Subscription(nil), s.subs...)
		s.mu.Unlock()
		return ipc.Response{Ok: true, Subscriptions: snapshot}

	case ipc.Cmdconnect:
		srv, ok := s.findserver(req.Url, req.Srvidx)
		if !ok {
			return ipc.Response{Error: "server not found"}
		}
		return s.conn.connect(srv, req.Url, req.Srvidx, req.Mode)

	case ipc.Cmddisconnect:
		logevent("disconnect requested")
		return s.conn.disconnect()

	case ipc.Cmdstatus:
		return s.conn.status()

	case ipc.Cmdshutdown:
		if s.conn.isconnected() {
			return ipc.Response{Ok: true, Connected: true}
		}
		go s.shutdown()
		return ipc.Response{Ok: true}

	case ipc.Cmdlogs:
		logs.configure(usersettings().Logs)
		lines, seq := logs.since(req.Since)
		return ipc.Response{Ok: true, Logs: lines, Seq: seq}

	case ipc.Cmdreset:
		return s.reset(req.Kind)

	case ipc.Cmdadd:
		set := usersettings()
		ctx, cancel := context.WithTimeout(context.Background(), fetchwindow(set))
		defer cancel()

		got, err := sub.Fetch(ctx, req.Url, fetchopts(set))
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}

		s.mu.Lock()
		replaced := false
		for i := range s.subs {
			if s.subs[i].Url == got.Url {
				s.subs[i] = sub.Merge(s.subs[i], got)
				replaced = true
				break
			}
		}
		if !replaced {
			s.subs = append(s.subs, got)
		}
		snapshot := append([]sub.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = sub.Saveall(snapshot)
		return ipc.Response{Ok: true, Subscriptions: snapshot}

	case ipc.Cmdremove:
		s.mu.Lock()
		out := s.subs[:0]
		for _, x := range s.subs {
			if x.Url != req.Url {
				out = append(out, x)
			}
		}
		s.subs = out
		snapshot := append([]sub.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = sub.Saveall(snapshot)
		return ipc.Response{Ok: true, Subscriptions: snapshot}

	case ipc.Cmdprobe:
		s.mu.Lock()
		var servers []sub.Server
		for i := range s.subs {
			if s.subs[i].Url == req.Url {
				servers = append([]sub.Server(nil), s.subs[i].Servers...)
				break
			}
		}
		s.mu.Unlock()
		if servers == nil {
			return ipc.Response{Error: "subscription not found"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		probe := sub.Subscription{Servers: servers}
		sub.Ping(ctx, &probe, usersettings().Pingproto)
		cancel()

		s.mu.Lock()
		for i := range s.subs {
			if s.subs[i].Url == req.Url {
				s.subs[i].Servers = probe.Servers
				break
			}
		}
		snapshot := append([]sub.Subscription(nil), s.subs...)
		s.mu.Unlock()

		_ = sub.Saveall(snapshot)
		return ipc.Response{Ok: true, Subscriptions: snapshot}

	case ipc.Cmdbest:
		return s.connectbest(req.Url, req.Mode)
	}
	return ipc.Response{Error: "unknown command"}
}
