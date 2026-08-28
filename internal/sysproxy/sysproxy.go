package sysproxy

type Opts struct {
	Host  string
	Socks int
	HTTP  int
}

func Enable(o Opts) error {
	if o.Host == "" {
		o.Host = "127.0.0.1"
	}
	return enable(o)
}

func Disable() error { return disable() }
