package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web
var webFS embed.FS

// staticFS é a subárvore web/ servida em /static/.
var staticFS = func() fs.FS {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	return sub
}()

// noCache desativa o cache do browser. Crítico: o self-update troca o
// binário inteiro, então o browser não pode servir um app.js/app.css antigo.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}
