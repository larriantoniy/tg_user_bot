package main

import (
	"errors"
	"fmt"
	redisrepo "github.com/larriantoniy/tg_user_bot/internal/adapters/redis"
	"github.com/larriantoniy/tg_user_bot/internal/adapters/tdlib"
	"github.com/larriantoniy/tg_user_bot/internal/config"
	delivery "github.com/larriantoniy/tg_user_bot/internal/delivery/http"
	"github.com/larriantoniy/tg_user_bot/internal/useCases"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// либо log.Fatalf, либо panic с читаемым сообщением
		fmt.Fprintf(os.Stderr, "ошибка загрузки конфига: %v\n", err)
		os.Exit(1)
	}
	logger := setupLogger(cfg.Env)

	rdb := redisrepo.NewPredictionRepo(cfg.RedisAddr, "", cfg.RedisDB, logger)
	ps := useCases.NewPredictionService(rdb, logger)
	handler := delivery.NewHandler(ps)
	router := delivery.NewRouter(handler)
	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	// Запускаем HTTP сервер в горутине:
	go func() {
		logger.Info("HTTP server starting", "addr", cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Error("HTTP server error", "err", err)
				os.Exit(1)
			}
			logger.Info("HTTP server closed")
		}
	}()

	tdClient, err := tdlib.NewClient(cfg.APIID, cfg.APIHash, logger)
	if err != nil {
		logger.Error("TDLib init failed", "error", err)
		os.Exit(1)
	}
	tdClient.JoinChannels(cfg.Channels)

	for {
		updates, err := tdClient.Listen()
		if err != nil {
			logger.Error("Listen failed, retrying", "error", err)
			time.Sleep(time.Second) // можно увеличить backoff по желанию
			continue
		}

		for msg := range updates {
			logger.Info("New message", "chat_id", msg.ChatID, "text", msg.Text)
			ps.Save(msg)
		}

		logger.Warn("Listen exited — вероятно упало соединение, пробуем снова")
	}
}

func setupLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case envDev:
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return logger
}
