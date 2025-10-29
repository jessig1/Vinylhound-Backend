package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that sets common CORS headers for the configured origin.
// If allowedOrigin is empty CORS headers are not applied. The special value "*"
// allows any origin.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	origin := strings.TrimSpace(allowedOrigin)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case origin == "":
				// no cors headers
			case origin == "*":
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Credentials", "false")
				setCommonHeaders(w)
			default:
				requestOrigin := r.Header.Get("Origin")
				if requestOrigin != "" && strings.EqualFold(requestOrigin, origin) {
					w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					setCommonHeaders(w)
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
	w.Header().Set("Access-Control-Max-Age", "3600")
}
