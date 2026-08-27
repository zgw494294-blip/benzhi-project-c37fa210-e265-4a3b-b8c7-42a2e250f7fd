package web

import (
	"embed"
	"net/http"
)

// staticAssets keeps the browser workbench inside the Go binary while leaving
// the HTML, CSS and JavaScript as independently inspectable frontend sources.
//
//go:embed assets/*
var staticAssets embed.FS

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	var name, contentType string
	switch r.URL.Path {
	case "/workbench", "/":
		name = "assets/index.html"
		contentType = "text/html; charset=utf-8"
	case "/static/app.css":
		name = "assets/app.css"
		contentType = "text/css; charset=utf-8"
	case "/static/app.js":
		name = "assets/app.js"
		contentType = "application/javascript; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}

	content, err := staticAssets.ReadFile(name)
	if err != nil {
		http.Error(w, "工作台资源不可用", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(content)
}
