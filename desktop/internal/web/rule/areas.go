package rule

import (
	"desktop/internal/ipregion"
	"desktop/internal/lang"
	"net/http"

	"github.com/penndev/prism/engine"
)

func HandleRuleAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := ipregion.Ensure(); err != nil {
		http.Error(w, lang.DefaultLang.T("rules.db.missing"), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(engine.AreaTree()))
}
