package subscribe

import (
	"desktop/internal/storage"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSubscriptionBytes = 2 << 20

func convertSubscriptionURL(subURL string, subType string) ([]storage.ServerEntry, error) {
	addr := strings.TrimSpace(subURL)
	parsed, err := url.Parse(addr)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("subscription url must be http or https")
	}
	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription failed: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(subType)) {
	case "shadowrocket":
		req.Header.Set("User-Agent", "shadowrocket/*")
	case "prism":
		version := strings.TrimSpace(appVersion())
		if version == "" {
			req.Header.Set("User-Agent", "prism/*")
		} else {
			req.Header.Set("User-Agent", "prism/"+version)
		}
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription response status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read subscription failed: %w", err)
	}
	if int64(len(body)) > maxSubscriptionBytes {
		return nil, fmt.Errorf("subscription content too large")
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil, fmt.Errorf("empty subscription content")
	}
	switch subType {
	case "prism":
		return subscriptionParsePrism(text)
	case "shadowrocket":
		return subscriptionParseShadowrocket(text)
	default:
		return nil, fmt.Errorf("unsupported subscription type: %s", subType)
	}
}
