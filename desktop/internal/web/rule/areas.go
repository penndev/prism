package rule

import (
	"desktop/internal"
	"desktop/internal/storage"
	"encoding/json"
	"net/http"

	"github.com/penndev/gopkg/ipregion"
)

type areaNode struct {
	ID       uint32     `json:"id"`
	ParentID uint32     `json:"parent_id"`
	Name     string     `json:"name"`
	Children []areaNode `json:"children,omitempty"`
}

func HandleRuleAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path, err := storage.IpregionDBPath()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := internal.EnsureSearcher(path); err != nil {
		http.Error(w, "未找到 ipregion.db，请先下载或上传", http.StatusBadRequest)
		return
	}
	searcher := internal.AcquireSearcher()
	defer internal.ReleaseSearcher()
	if searcher == nil {
		http.Error(w, "未找到 ipregion.db，请先下载或上传", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(buildAreaTree(searcher, 0))
}

// searcher 由调用方持读锁后传入，避免递归里反复 RLock（有写者等待时会死锁）。
func buildAreaTree(searcher *ipregion.Searcher, parentID uint32) []areaNode {
	src := searcher.Areas(parentID)
	out := make([]areaNode, 0, len(src))
	for _, a := range src {
		out = append(out, areaNode{
			ID:       a.ID,
			ParentID: a.ParentID,
			Name:     a.Name,
			Children: buildAreaTree(searcher, a.ID),
		})
	}
	return out
}
