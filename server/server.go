package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func NewServer(rep *Repository) *Server {
	return &Server{
		rep: rep,
	}
}

func (s *Server) Start(port string) error {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api", func(r chi.Router) {
		r.Post("/check", s.checkLinksHandler)
		r.Post("/report", s.reportHandler)
	})

	return http.ListenAndServe(":"+port, router)
}
