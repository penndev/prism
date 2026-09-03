package rule

import (
	"desktop/internal/storage"
	"strings"
	"sync"

	"github.com/penndev/prism/fakeip"
)

var domainSet = map[string]struct{}{}
var domainMu sync.RWMutex

func setDomainMap(list []string) {
	next := make(map[string]struct{}, len(list))
	for _, d := range list {
		if d != "" {
			next[d] = struct{}{}
		}
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
		setDomainMap(nil)
		return
	}
	setDomainMap(normalizeDomains(cfg.Domains))
}

func normalizeDomains(list []string) []string {
	if list == nil {
		return []string{}
	}
	out := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		d := strings.ToLower(strings.TrimSpace(item))
		d = strings.TrimPrefix(d, ".")
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}
