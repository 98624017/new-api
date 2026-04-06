package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
)

func main() {
	port := flag.Int("port", 39090, "listen port")
	taskID := flag.String("task-id", "upstream-video-failure-001", "upstream task id to mock")
	flag.Parse()

	var hitCount atomic.Int64
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"task_fetch_hits":%d}`, hitCount.Load())
	})

	mux.HandleFunc("/v1/videos/"+*taskID, func(w http.ResponseWriter, _ *http.Request) {
		hitCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":"%s","object":"video","status":"failed","progress":100,"error":{"message":"mock upstream failure","code":"mock_failure"}}`, *taskID)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "mock video failure server failed: %v\n", err)
		os.Exit(1)
	}
}
