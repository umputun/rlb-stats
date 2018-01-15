package rest

import (
	"fmt"
	"net/http"

	"log"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/umputun/rlb-stats/app/store"
)

// Server is a rest interface to storage
type Server struct {
	Engine store.Engine
	Port   int
}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate rest server on port %v", s.Port)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("there will be stats"))
		if err != nil {
			log.Printf("[ERROR] can't serve request '%v', %v", r.RequestURI, err)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", s.Port), r))
}
