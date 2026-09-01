package transport

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// FromURL builds a HandleConnect from a proxy URL.
// Schemes: socks5, socks5s, http, https.
func FromURL(r *url.URL) (HandleConnect, error) {
	if r == nil || r.Host == "" {
		return nil, fmt.Errorf("server required")
	}
	user := ""
	pass := ""
	if r.User != nil {
		user = r.User.Username()
		pass, _ = r.User.Password()
	}
	switch strings.ToLower(r.Scheme) {
	case "socks5":
		return Socks5(r.Host, user, pass), nil
	case "socks5s":
		return Socks5OverTLS(r.Host, user, pass, &tls.Config{}), nil
	case "http":
		return Http(r.Host, user, pass), nil
	case "https":
		return HttpOverTLS(r.Host, user, pass, &tls.Config{}), nil
	default:
		return nil, errors.New("cant find Scheme" + r.Scheme)
	}
}
