package rule

import (
	"desktop/internal/storage"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/prism/fakeip"
)

var domainSet = map[string]struct{}{}
var domainMu sync.RWMutex

func setDomainMap(text string) {
	next := make(map[string]struct{})
	for _, line := range strings.Split(text, "\n") {
		d := strings.ToLower(strings.TrimSpace(line))
		d = strings.TrimPrefix(d, ".")
		if d == "" || strings.HasPrefix(d, "!") {
			continue
		}
		if _, ok := dns.IsDomainName(d); !ok {
			continue
		}
		next[d] = struct{}{}
	}
	domainMu.Lock()
	domainSet = next
	domainMu.Unlock()
}

// matchDomain 判断域名或它的任一父域是否在规则表里。
func matchDomain(name string) bool {
	domainMu.RLock()
	defer domainMu.RUnlock()
	if name == "" || len(domainSet) == 0 {
		return false
	}
	for name != "" {
		if _, ok := domainSet[name]; ok {
			return true
		}
		i := strings.IndexByte(name, '.')
		if i < 0 {
			return false
		}
		name = name[i+1:]
	}
	return false
}

func LoadDomains() {
	// 无条件注册：matchDomain 读的是 domainSet，规则页保存时只会更新 domainSet。
	// 放在下面的提前 return 之后的话，首次运行（还没存过规则配置）就永远不会注册，
	// 本次运行内加的域名要重启才生效。
	fakeip.SetNeedFake(matchDomain)

	st := storage.DefaultStorage
	if st == nil {
		return
	}
	cfg, err := st.GetRuleConfig()
	if err != nil || cfg == nil {
		setDomainMap("")
		return
	}
	setDomainMap(cfg.Domains)
}

func HandleRuleFetchText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	addr := strings.TrimSpace(payload.URL)
	u, err := url.Parse(addr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(addr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) > 2<<20 {
		http.Error(w, "too large", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(string(body))
	if resp.StatusCode < 200 || resp.StatusCode > 299 || text == "" {
		http.Error(w, "fetch failed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
}
