package subscribe

import (
	"desktop/build"
	"embed"
	"io/fs"
	"net/http"
)

func HandleSubscribeRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/subscribe/", http.StatusFound)
}

func HandleSubscribeLogo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(build.Icon) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(build.Icon)
}

//go:embed all:static
var staticSubscribeFS embed.FS

func HandleSubscribeFileServer() http.Handler {
	sub, err := fs.Sub(staticSubscribeFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
