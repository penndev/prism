package subscribe

import (
	"desktop/internal/storage"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func subscriptionParseShadowrocket(text string) ([]storage.ServerEntry, error) {
	encoded := strings.TrimSpace(text)
	if encoded == "" {
		return nil, fmt.Errorf("empty prism subscription")
	}
	if m := len(encoded) % 4; m != 0 {
		encoded += strings.Repeat("=", 4-m)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid prism base64: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(decoded), "\r\n", "\n"), "\n")
	servers := make([]storage.ServerEntry, 0, len(lines))
	for _, line := range lines {
		u, err := subscriptionParseNodeURL(line)
		if err != nil {
			continue
		}
		scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
		protocol := ""
		switch scheme {
		case "https":
			protocol = "httpovertls"
		case "socks5":
			protocol = "socks5overtls"
		default:
			continue
		}
		host := strings.TrimSpace(u.Host)
		if host == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(host); err != nil {
			continue
		}
		username, password := "", ""
		if u.User != nil {
			username = u.User.Username()
			password, _ = u.User.Password()
		}
		remark := strings.TrimSpace(u.Fragment)
		if remark == "" {
			remark = host
		}
		servers = append(servers, storage.ServerEntry{
			Host:     host,
			Remark:   remark,
			Username: username,
			Password: password,
			Protocol: protocol,
		})
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no https/socks5 nodes found in subscription")
	}
	return servers, nil
}

func subscriptionParseNodeURL(line string) (*url.URL, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty subscription line")
	}

	parts := strings.SplitN(line, "://", 2)
	if len(parts) != 2 {
		return url.Parse(line)
	}

	scheme := strings.TrimSpace(parts[0])
	payload := strings.TrimSpace(parts[1])
	if scheme == "" || payload == "" {
		return url.Parse(line)
	}

	encoded := payload
	if m := len(encoded) % 4; m != 0 {
		encoded += strings.Repeat("=", 4-m)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return url.Parse(line)
	}

	rebuilt := scheme + "://" + strings.TrimSpace(string(decoded))
	return url.Parse(rebuilt)
}
