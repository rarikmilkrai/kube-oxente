package main

import (
	"embed"
	"net"
	"net/http"
	"path"
	"strings"
)

//go:embed ui
var uiFS embed.FS

func serveStatic(w http.ResponseWriter, r *http.Request) {
	name := path.Clean(strings.TrimPrefix(r.URL.Path, "/static/"))
	b, err := uiFS.ReadFile("ui/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch path.Ext(name) {
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(b)
}

func hostGuard(listen string, next http.Handler) http.Handler {
	open := listen != "127.0.0.1" && listen != "localhost" && listen != "::1"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !open {
			h := r.Host
			if hh, _, err := net.SplitHostPort(h); err == nil {
				h = hh
			}
			if h != "127.0.0.1" && h != "localhost" && h != "::1" && h != "[::1]" {
				http.Error(w, "host não permitido", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
