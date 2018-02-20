package web

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

// UIRouter handle routes for dashboard
func UIRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getDashboard)
	return r
}

// GET /dashboard
func getDashboard(w http.ResponseWriter, r *http.Request) {
	render.PlainText(w, r, "OK")
	render.Status(r, http.StatusOK)
}
