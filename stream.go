package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

func handleLogStream(w http.ResponseWriter, r *http.Request, tailLines int) {
	q := r.URL.Query()
	ns := q.Get("ns")
	name := q.Get("name")

	if ns == "" || name == "" {
		http.Error(w, "ns e name obrigatórios", 400)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	args := []string{"logs", "-f", "-n", ns, name, "--all-containers", "--tail", itoa(tailLines)}
	if c := getActiveContext(); c != "" {
		args = append([]string{"--context", c}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		b, _ := json.Marshal(fmt.Sprintf("erro ao iniciar logs: %s", err))
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		return
	}

	ctx := r.Context()
	done := make(chan struct{})

	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			b, _ := json.Marshal(line)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}

	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	pw.Close()
	<-done
}
