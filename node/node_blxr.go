package node

import (
	"errors"
)

var ErrInvalidServersDestination = errors.New("invalid servers destination")

// Configure HTTP secured by IP
func configureHTTPSecured(n *Node, servers *[]*httpServer) error {
	if n.config.HTTPHost == "" || n.config.HTTPSecuredIPPort == 0 {
		return nil
	}

	if servers == nil {
		return ErrInvalidServersDestination
	}

	var err error

	if err = n.httpSecuredIP.setListenAddr(n.config.HTTPHost, n.config.HTTPSecuredIPPort); err != nil {
		return err
	}

	if err = n.httpSecuredIP.enableRPC(n.rpcAPIs, httpConfig{
		CorsAllowedOrigins: n.config.HTTPCors,
		AllowedIPs:         n.config.HTTPSecuredIPAllowedIPs,
		Modules:            n.config.HTTPSecuredIPModules,
		prefix:             n.config.HTTPPathPrefix,
	}); err != nil {
		return err
	}

	*servers = append(*servers, n.httpSecuredIP)

	return nil
}
