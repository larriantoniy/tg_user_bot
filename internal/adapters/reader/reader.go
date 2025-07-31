package reader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/config"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Reader struct {
	client      *http.Client
	ctx         *context.Context
	logger      *slog.Logger
	baseURL     string                    // https://openrouter.ai/api/v1
	apiKey      string                    // TOKEN neuro
	defaultBody *domain.DefaultReaderBody // закодированное JSON-тело
	// заготовленный http.Request
}

func NewReader(cfg *config.Config, logger *slog.Logger) (*Reader, error) {
	// 1) Кодируем заранее JSON-тело
	body := domain.DefaultReaderBody{
		Language: "rus",
	}

	// 3) Собираем объект Neuro
	return &Reader{
		client:      &http.Client{},
		logger:      logger,
		baseURL:     cfg.ReaderAddr,
		apiKey:      cfg.ReaderToken,
		defaultBody: &body,
	}, nil
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}

func (r *Reader) Read(ctx context.Context, photoFile string, wg *sync.WaitGroup) (string, error) {
	defer wg.Done()
	// Подготовка тела
	body := r.defaultBody
	body.Base64Image = photoFile
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	// Создание запроса
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("apikey", r.apiKey)

	// Логируем URL, метод и заголовки — безопасно
	r.logger.Info("Request to neuro",
		"url", req.URL.String(),
		"method", req.Method,
		"headers", req.Header,
	)

	// Логируем тело запроса (если нужно для отладки)
	r.logger.Debug("Request body", "body", string(bodyBytes))

	// Ответ нейросети
	var rr domain.ReaderResponse

	err = retry(3, time.Second, func() error {
		resp, err := r.client.Do(req)
		if err != nil {
			r.logger.Error("HTTP request to reader failed", "err", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			r.logger.Error("reader API returned error",
				"status", resp.StatusCode,
				"body", string(data),
			)
			return fmt.Errorf("status %d: %s", resp.StatusCode, string(data))
		}

		return json.NewDecoder(resp.Body).Decode(&rr)
	})

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if len(rr.ParsedResults) == 0 {
		return "", fmt.Errorf("empty ParsedResults")
	}

	r.logger.Info("After reader processing", "result", *rr.ParsedResults[0].ParsedText)

	return *rr.ParsedResults[0].ParsedText, nil
}
