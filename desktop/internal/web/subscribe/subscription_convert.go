package subscribe

import (
	"desktop/internal/storage"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func convertSubscriptionURL(subURL string, subType string) ([]storage.ServerEntry, error) {
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(subURL)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription response status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read subscription failed: %w", err)
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
