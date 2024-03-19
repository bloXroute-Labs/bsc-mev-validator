package node

import (
	"net"
	"net/http"
	"strings"
)

// SecuredIPHandler is a handler which validates the IP of incoming requests.
type SecuredIPHandler struct {
	ip   map[string]struct{}
	next http.Handler
}

func newIPHandler(ips []string, next http.Handler) http.Handler {
	if len(ips) == 0 {
		return next
	}

	vIPsMap := make(map[string]struct{})
	for _, allowedIP := range ips {
		vIPsMap[strings.ToLower(allowedIP)] = struct{}{}
	}

	return &SecuredIPHandler{ip: vIPsMap, next: next}
}

// ServeHTTP serves JSON-RPC requests over HTTP, implements http.Handler
func (h *SecuredIPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if r.Host is not set, we can continue serving since a browser would set the Host header
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Either invalid (too many colons) or no port specified
		ip = r.RemoteAddr
	}

	// Not an IP address, but a hostname. Need to validate
	if _, exist := h.ip["*"]; exist {
		h.next.ServeHTTP(w, r)
		return
	}

	if _, exist := h.ip[ip]; exist {
		h.next.ServeHTTP(w, r)
		return
	}

	http.Error(w, "IP is not allowed", http.StatusForbidden)
}
