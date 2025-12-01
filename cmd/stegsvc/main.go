package main

import (
	"log"
	"net/http"
	"time"

	"github.com/aswearingen91/Steganography-Service/internal/handlers"
)

func main() {
	h := handlers.NewHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("/encode", h.EncodeHandler)
	mux.HandleFunc("/decode", h.DecodeHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("stegsvc: service up\n"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("stegsvc starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// Simple request logger
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
