package subscribe

import (
	"desktop/internal/storage"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func subscriptionParsePrism(text string) ([]storage.ServerEntry, error) {
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
	var servers []storage.ServerEntry
	if err := json.Unmarshal(decoded, &servers); err != nil {
		return nil, fmt.Errorf("invalid prism json: %w", err)
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("empty prism nodes")
	}
	for i := range servers {
		servers[i].Protocol = storage.NormalizeProtocol(servers[i].Protocol)
	}
	return servers, nil
}
