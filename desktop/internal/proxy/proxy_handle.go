package proxy

import (
	"crypto/tls"
	"errors"
	"net/url"

	"github.com/penndev/prism/transport"
)

func HandleConnect(r *url.URL) (transport.HandleConnect, error) {
	user := r.User.Username()
	pass, _ := r.User.Password()
	switch r.Scheme {
	case "socks5":
		return transport.Socks5(r.Host, user, pass), nil
	case "socks5overtls":
		return transport.Socks5OverTLS(r.Host, user, pass, &tls.Config{}), nil
	case "http":
		return transport.Http(r.Host, user, pass), nil
	case "httpovertls":
		return transport.HttpOverTLS(r.Host, user, pass, &tls.Config{}), nil
	default:
		return nil, errors.New("cant find Scheme" + r.Scheme)
	}
}
