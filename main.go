// main.go — servidor do painel kdash (read-only).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func main() {
	port := flag.Int("port", 8200, "porta do painel web")
	ctx := flag.String("context", "", "contexto kubeconfig inicial (vazio = o atual)")
	logLines := flag.Int("logs", 40, "linhas de log no detalhe do pod")
	flag.Parse()

	if *ctx != "" {
		setActiveContext(*ctx)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	mux.HandleFunc("/api/contexts", func(w http.ResponseWriter, r *http.Request) {
		cur, all := listContexts()
		writeJSON(w, map[string]any{"current": cur, "active": effectiveContext(), "all": all})
	})
	mux.HandleFunc("/api/setcontext", func(w http.ResponseWriter, r *http.Request) {
		setActiveContext(r.URL.Query().Get("name"))
		writeJSON(w, map[string]any{"ok": true, "active": effectiveContext()})
	})

	mux.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectOverview()) })
	mux.HandleFunc("/api/problems", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectProblems()) })
	mux.HandleFunc("/api/namespaces", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectNamespaces()) })
	mux.HandleFunc("/api/pods", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectPods()) })
	mux.HandleFunc("/api/workloads", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectWorkloads()) })
	mux.HandleFunc("/api/services", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectServices()) })
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectEvents()) })
	mux.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, collectNodes()) })
	mux.HandleFunc("/api/detail", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		writeJSON(w, collectDetail(q.Get("ns"), q.Get("name"), q.Get("reason"), *logLines))
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("\n  kdash em  http://%s\n", addr)
	fmt.Printf("  contexto: %s\n", effectiveContext())
	fmt.Printf("  (só leitura · Ctrl+C para sair)\n\n")
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}
