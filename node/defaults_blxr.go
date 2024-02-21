package node

const DefaultHTTPSecuredIPPort = 8548 // Default TCP port for the HTTP RPC server secured by IP

func WithDefaultHTTPSecuredIP(cfg *Config) *Config {
	cfg.HTTPSecuredIPPort = DefaultHTTPSecuredIPPort
	cfg.HTTPSecuredIPAllowedIPs = []string{}
	cfg.HTTPSecuredIPModules = []string{}

	return cfg
}
