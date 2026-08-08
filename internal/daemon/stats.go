package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	statsinterval = time.Second
	statstimeout  = 900 * time.Millisecond
	statsmisses   = 3
)

var statsclient = &http.Client{
	Timeout: statstimeout,
	Transport: &http.Transport{
		Proxy:               nil,
		MaxIdleConnsPerHost: 1,
	},
}

var statsskip = map[string]bool{
	"direct":      true,
	"block":       true,
	"metrics_out": true,
	"api":         true,
	"dns-out":     true,
}

type stattrack struct {
	mu       sync.Mutex
	prevup   int64
	prevdown int64
	accup    int64
	accdown  int64
	up       int64
	down     int64
	uprate   int64
	downrate int64
	at       time.Time
	ok       bool
}

type statsnap struct {
	up       int64
	down     int64
	uprate   int64
	downrate int64
}

func (t *stattrack) reset() {
	t.mu.Lock()
	t.prevup, t.prevdown = 0, 0
	t.accup, t.accdown = 0, 0
	t.up, t.down = 0, 0
	t.uprate, t.downrate = 0, 0
	t.at = time.Time{}
	t.ok = false
	t.mu.Unlock()
}

func (t *stattrack) feed(rawup, rawdown int64, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if rawup < t.prevup || rawdown < t.prevdown {
		t.accup += t.prevup
		t.accdown += t.prevdown
		t.prevup, t.prevdown = 0, 0
	}

	up := t.accup + rawup
	down := t.accdown + rawdown

	if t.ok && now.After(t.at) {
		secs := now.Sub(t.at).Seconds()
		if secs > 0 {
			t.uprate = int64(float64(up-t.up) / secs)
			t.downrate = int64(float64(down-t.down) / secs)
		}
	}

	t.prevup, t.prevdown = rawup, rawdown
	t.up, t.down = up, down
	t.at = now
	t.ok = true
}

func (t *stattrack) stale() {
	t.mu.Lock()
	t.uprate, t.downrate = 0, 0
	t.mu.Unlock()
}

func (t *stattrack) snapshot() statsnap {
	t.mu.Lock()
	defer t.mu.Unlock()
	return statsnap{up: t.up, down: t.down, uprate: t.uprate, downrate: t.downrate}
}

func pickcounters(st map[string]map[string]map[string]int64) (int64, int64) {
	ob := st["outbound"]
	if ob == nil {
		return 0, 0
	}
	if c, ok := ob["proxy"]; ok {
		return c["uplink"], c["downlink"]
	}
	var up, down int64
	for tag, c := range ob {
		if statsskip[tag] {
			continue
		}
		up += c["uplink"]
		down += c["downlink"]
	}
	return up, down
}

func readvars(ctx context.Context, port int) (int64, int64, bool) {
	addr := "http://127.0.0.1:" + strconv.Itoa(port) + "/debug/vars"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return 0, 0, false
	}
	resp, err := statsclient.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}
	var v struct {
		Stats map[string]map[string]map[string]int64 `json:"stats"`
	}
	if json.NewDecoder(resp.Body).Decode(&v) != nil {
		return 0, 0, false
	}
	up, down := pickcounters(v.Stats)
	return up, down, true
}

func (c *conn) pollstats(port int, stop chan struct{}) {
	defer c.wg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stop
		cancel()
	}()

	t := time.NewTicker(statsinterval)
	defer t.Stop()

	misses := 0
	for {
		select {
		case <-stop:
			return
		case now := <-t.C:
			up, down, ok := readvars(ctx, port)
			if !ok {
				misses++
				if misses >= statsmisses {
					c.stats.stale()
				}
				continue
			}
			misses = 0
			c.stats.feed(up, down, now)
		}
	}
}
