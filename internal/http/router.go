package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"automessaging/internal/http/handler"
)

// NewRouter wires HTTP routes.
func NewRouter(control *handler.ControlHandler, message *handler.MessageHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	api := chi.NewRouter()
	api.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	api.Route("/control", func(r chi.Router) {
		r.Post("/start", control.Start)
		r.Post("/stop", control.Stop)
	})

	api.Route("/messages", func(r chi.Router) {
		r.Get("/sent", message.ListSent)
	})

	fileServer := http.StripPrefix("/api/v1/docs/", http.FileServer(http.Dir("./api")))
	api.Handle("/docs/*", fileServer)

	r.Mount("/api/v1", api)

	return r
}
