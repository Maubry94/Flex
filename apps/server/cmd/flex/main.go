package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flex.local/server/internal/config"
	"flex.local/server/internal/database"
	"flex.local/server/internal/httpserver"
	"flex.local/server/internal/library"
	"flex.local/server/internal/media"
	"flex.local/server/internal/playback"
	"flex.local/server/internal/scanmanager"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.FromEnv()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	db, err := database.Open(cfg.ConfigPath)
	if err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	libraryService := library.NewService(library.NewSQLRepository(db), cfg.MediaPath)
	mediaRepository := media.NewSQLRepository(db)
	thumbnailGenerator := media.NewThumbnailGenerator(cfg.CachePath, "")
	transcoder := media.NewTranscoder(cfg.CachePath, "")
	mediaScanner := media.NewScanner(libraryService, mediaRepository, media.FFprobe{}, thumbnailGenerator, transcoder, logger)
	scanCoordinator := scanmanager.New(cfg.MediaPath, libraryService, mediaScanner, logger)
	if err := scanCoordinator.Start(context.Background()); err != nil {
		logger.Error("automatic scanning initialization failed", "error", err)
		os.Exit(1)
	}
	defer scanCoordinator.Close()
	playbackService := playback.NewService(playback.NewSQLRepository(db))

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           httpserver.New(cfg, logger, libraryService, mediaScanner, scanCoordinator, playbackService),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("Flex server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	logger.Info("shutting down Flex server")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
