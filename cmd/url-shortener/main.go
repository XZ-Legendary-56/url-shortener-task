package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/router"
	"url-shortener/internal/lib/logger/sl"
	appStorage "url-shortener/internal/storage"
	"url-shortener/internal/storage/memory"
	"url-shortener/internal/storage/postgres"
)

func main() {
	// TODO: init config: cleanenv
	cfg := config.MustLoad()

	fmt.Println(cfg)

	// TODO: init logger: slog
	log := setupLogger(cfg.Env)

	log.Info("starting url shortener", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	// TODO: init storage: postgres
	// Инициализация хранилища в зависимости от типа
	var store appStorage.Storage
	var err error

	switch cfg.StorageType {
	case "postgres":
		log.Info("using postgres storage")
		store, err = postgres.New(cfg.StoragePath)
	case "memory":
		log.Info("using in-memory storage")
		store = memory.New()
	default:
		log.Error("unknown storage type", slog.String("type", cfg.StorageType))
		os.Exit(1)
	}

	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	defer store.Close()

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router.New(log, store),
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// TODO: run server
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Error("server stopped")
}

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
