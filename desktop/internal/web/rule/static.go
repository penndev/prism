package rule

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:static
var staticRuleFS embed.FS

func HandleRuleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/rule/", http.StatusFound)
}

func HandleRuleFileServer() http.Handler {
	sub, err := fs.Sub(staticRuleFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
