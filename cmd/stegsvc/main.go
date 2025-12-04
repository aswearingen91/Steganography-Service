package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aswearingen91/Steganography-Service/internal/handlers"
)

// Set LOG_LEVEL=DEBUG to see verbose debug logs
var logLevel = getLogLevel()

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	h := handlers.NewHandler()

	mux := http.NewServeMux()

	// Wrap each handler with cors
	mux.Handle("/encode", cors(http.HandlerFunc(h.EncodeHandler)))
	mux.Handle("/decode", cors(http.HandlerFunc(h.DecodeHandler)))

	// Health check
	mux.Handle("/", cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("stegsvc: service up\n"))
	})))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	infof("stegsvc starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		errorf("server failed: %v", err)
	}
}

// -------------------- Logging Middleware --------------------

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		infof("➡ Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if logLevel == "DEBUG" {
			for k, v := range r.Header {
				debugf("   Header: %s=%v", k, v)
			}
		}

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		infof("⬅ Completed %s %s with status %d in %v", r.Method, r.URL.Path, lrw.statusCode, duration)

		if logLevel == "DEBUG" {
			debugf("   RemoteAddr=%s Duration=%v", r.RemoteAddr, duration)
		}
	})
}

// -------------------- Response Writer Wrapper --------------------

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// -------------------- CORS Middleware --------------------

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			debugf("OPTIONS preflight handled for %s", r.URL.Path)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// -------------------- Logging Helpers --------------------

func getLogLevel() string {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		return "INFO"
	}
	return level
}

func debugf(format string, v ...interface{}) {
	if logLevel == "DEBUG" {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func infof(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func warnf(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

func errorf(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}
