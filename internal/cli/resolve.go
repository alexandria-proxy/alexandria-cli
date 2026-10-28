package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

type entry struct {
	num int
	sub int
	idx int
	srv sub.Server
}

func number(list []sub.Subscription) []entry {
	var out []entry
	n := 0
	for si, s := range list {
		for i, srv := range s.Servers {
			n++
			out = append(out, entry{num: n, sub: si, idx: i, srv: srv})
		}
	}
	return out
}

func ambiguous(arg string, names []string) error {
	return fmt.Errorf("'%s' is ambiguous:\n  %s", arg, strings.Join(names, "\n  "))
}

func findserver(list []sub.Subscription, arg string) (entry, error) {
	all := number(list)
	if len(all) == 0 {
		return entry{}, fmt.Errorf("no servers")
	}
	if n, err := strconv.Atoi(strings.Trim(arg, "{}")); err == nil {
		if n < 1 || n > len(all) {
			return entry{}, fmt.Errorf("no server {%d}", n)
		}
		return all[n-1], nil
	}
	needle := strings.ToLower(arg)
	var hits, exact []entry
	for _, e := range all {
		name := strings.ToLower(e.srv.Name)
		if name == needle || strings.TrimSpace(strings.TrimLeftFunc(name, isflagrune)) == needle {
			exact = append(exact, e)
		}
		if strings.Contains(name, needle) {
			hits = append(hits, e)
		}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	switch len(hits) {
	case 0:
		return entry{}, fmt.Errorf("no server matches '%s'", arg)
	case 1:
		return hits[0], nil
	}
	names := make([]string, len(hits))
	for i, e := range hits {
		names[i] = fmt.Sprintf("{%d} %s", e.num, e.srv.Name)
	}
	return entry{}, ambiguous(arg, names)
}

func isflagrune(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func findsub(list []sub.Subscription, arg string) (int, error) {
	if len(list) == 0 {
		return 0, fmt.Errorf("no subscriptions")
	}
	if n, err := strconv.Atoi(strings.Trim(arg, "{}")); err == nil {
		if n < 1 || n > len(list) {
			return 0, fmt.Errorf("no subscription {%d}", n)
		}
		return n - 1, nil
	}
	needle := strings.ToLower(arg)
	var hits []int
	for i, s := range list {
		name := strings.ToLower(s.Name)
		if name == needle || s.Url == arg {
			return i, nil
		}
		if strings.Contains(name, needle) {
			hits = append(hits, i)
		}
	}
	switch len(hits) {
	case 0:
		return 0, fmt.Errorf("no subscription matches '%s'", arg)
	case 1:
		return hits[0], nil
	}
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = fmt.Sprintf("{%d} %s", h+1, list[h].Name)
	}
	return 0, ambiguous(arg, names)
}

func findsubs(list []sub.Subscription, args []string) []int {
	if len(args) == 0 {
		out := make([]int, len(list))
		for i := range list {
			out[i] = i
		}
		return out
	}
	var out []int
	seen := map[int]bool{}
	for _, a := range args {
		i, err := findsub(list, a)
		if err != nil {
			errf("%v", err)
			continue
		}
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	return out
}
