// Package api sadrži HTTP handlere i middleware za Open Data servis.
package api

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware je HTTP middleware koji loguje svaki zahtev:
// metod, putanju, status kod i vreme obrade.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Omotavamo ResponseWriter da bismo uhvatili status kod
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s | status=%d | trajanje=%s | ip=%s",
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// responseWriter je wrapper oko http.ResponseWriter koji pamti status kod.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader presreće status kod pre prosleđivanja originalnom writeru.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
