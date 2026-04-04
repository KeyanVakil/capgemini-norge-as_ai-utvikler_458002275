package api

import (
	"net/http"
	"strings"
)

func NewRouter(h *Handler, staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/reviews", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateReview(w, r)
		case http.MethodGet:
			h.ListReviews(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/reviews/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stream") {
			h.StreamReview(w, r)
		} else {
			h.GetReview(w, r)
		}
	})

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("/review/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, staticDir+"/../templates/review.html")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/static/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, staticDir+"/../templates/index.html")
	})

	handler := RecoveryMiddleware(CORSMiddleware(LoggingMiddleware(mux)))
	return handler
}
