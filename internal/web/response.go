package web

import (
	"net/http"
	"strings"
)

func allowJSON(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func methodAllowed(w http.ResponseWriter, methods ...string) bool {
	for _, method := range methods {
		if method == http.MethodGet {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	w.WriteHeader(http.StatusMethodNotAllowed)
	return false
}
func pathID(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "projects" {
		return parts[2]
	}
	return ""
}
func writeText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(text))
}
