package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
)

// Storage — то, что нужно роутеру от хранилища.
type Storage interface {
	save.URLSaver
	redirect.URLGetter
}

// New собирает HTTP-роутер сервиса. Используется и в main, и в
// функциональных тестах, чтобы тесты проверяли ровно ту же конфигурацию.
func New(log *slog.Logger, store Storage) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)

	r.Post("/url", save.New(log, store))
	r.Get("/{alias}", redirect.New(log, store))

	return r
}
