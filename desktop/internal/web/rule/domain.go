package rule

import (
	"desktop/internal/storage"
	"strings"
	"sync"

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
		if d == "" {
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
