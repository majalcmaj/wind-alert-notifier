// Vendored assets:
//   htmx 2.0.3  https://unpkg.com/htmx.org@2.0.3/dist/htmx.min.js
//   Pico 2.0.6  https://cdn.jsdelivr.net/npm/@picocss/pico@2.0.6/css/pico.classless.min.css

package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return withCacheControl(http.FileServerFS(sub))
}

func withCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
