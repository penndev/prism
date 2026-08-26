package dns

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"desktop/internal/storage"
	"github.com/miekg/dns"
)

func localProxyURL() *url.URL {
	u := &url.URL{Scheme: "http", Host: defaultProxy}
	if storage.DefaultStorage == nil {
		return u
	}
	s, err := storage.DefaultStorage.GetSettings()
	if err != nil || s == nil {
		return u
	}
	if s.Proxy.Port > 0 {
		u.Host = net.JoinHostPort("127.0.0.1", strconv.Itoa(s.Proxy.Port))
	}
	if s.Proxy.Username != "" || s.Proxy.Password != "" {
		u.User = url.UserPassword(s.Proxy.Username, s.Proxy.Password)
	}
	return u
}

// queryDoH 经本地代理向 DoH 上游查询。
func queryDoH(req *dns.Msg) (*dns.Msg, error) {
	data, err := req.Pack()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, dohUpstream, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	httpResp, err := (&http.Client{
		Timeout:   dohTimeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(localProxyURL())},
	}).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH: %s", httpResp.Status)
	}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp := new(dns.Msg)
	return resp, resp.Unpack(body)
}
