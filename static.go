package main

import "net/http"

func static(prefix string, w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, prefix+r.URL.Path)
}

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/frontend/index.html")
}

func ServeAdminIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/admin/index.html")
}

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	var head string
	head, r.URL.Path = shiftPath(r.URL.Path)
	switch head {
	case "frontend":
		static("./static/frontend", w, r)
	case "admin":
		static("./static/admin", w, r)
	default:
		http.NotFound(w, r)
	}
}
