package server

import (
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
)

func BasicAuth(next http.Handler) http.Handler {
	user := os.Getenv("ADMIN_USER")
	pass := os.Getenv("ADMIN_PASSWORD")

	if user == "" || pass == "" {
		log.Println("WARNING: ADMIN_USER or ADMIN_PASSWORD not set — all requests will be rejected")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			unauthorized(w)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			unauthorized(w)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			unauthorized(w)
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			unauthorized(w)
			return
		}

		userMatch := subtle.ConstantTimeCompare([]byte(parts[0]), []byte(user))
		passMatch := subtle.ConstantTimeCompare([]byte(parts[1]), []byte(pass))

		if userMatch != 1 || passMatch != 1 {
			unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="wind-alert"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
