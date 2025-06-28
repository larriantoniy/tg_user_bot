package neuro

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
	"strings"
	"time"
)

const prompt = "Вот скриншот. " +
	"Распознай информацию , определи и дай ответ вида :\n" +
	"Вид спорта: [Вид спорта на английском языке если это ставка на спортивное событие , и n/a если это не ставка на спортивное событие или ставка на сыгранное спортивное событие]\n" +
	"Ставка: [true если это ставка на спортивное событие , и false если это не ставка на спортивное событие или ставка на сыгранное спортивное событие]\n" +
	"Без домыслов, анализа или пояснений — только данные со скриншота."

type Neuro struct {
	client      *http.Client
	ctx         *context.Context
	logger      *slog.Logger
	baseURL     string                   // https://api.deepseek.com/chat/completions
	apiKey      string                   // TOKEN neuro
	defaultBody *domain.DefaultNeuroBody // закодированное JSON-тело
	// заготовленный http.Request
}

func NewNeuro(cfg *config.Config, logger *slog.Logger) (*Neuro, error) {
	// 1) Кодируем заранее JSON-тело
	body := domain.DefaultNeuroBody{
		ContentType:   "application/json",
		Authorization: "Bearer " + cfg.NeuroToken, // ваш ключ опенроутер
		Model:         domain.MistralModel,        // например "mistral-small-2506"
		Messages: []domain.NeuroMessage{
			{
				Role: domain.RoleUser,
				Content: []domain.MessageContent{
					{
						Type: "text", // "text" для промпта
						Text: "prompt",
					},
					{
						Type:     "image_url", // "image_url" , шлем фото в б64
						ImageUrl: domain.ImageUrl{Url: ""},
					},
				},
			},
		},
		Stream: false,
	}

	// 3) Собираем объект Neuro
	return &Neuro{
		client:      &http.Client{},
		logger:      logger,
		baseURL:     cfg.NeuroAddr,
		apiKey:      cfg.NeuroToken,
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

func (n *Neuro) GetCompletion(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	// Подготовка тела
	body := n.defaultBody
	body.Messages[0].Content[1].ImageUrl.Url = msg.PhotoFile
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	url := n.baseURL + "/" + domain.MistralModel

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	n.logger.Info("Request to neuro", "url", req.URL.String())

	var nr domain.NeuroResponse

	err = retry(3, time.Second, func() error {
		resp, err := n.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("status %d: %s", resp.StatusCode, string(data))
		}
		return json.NewDecoder(resp.Body).Decode(&nr)
	})
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if len(nr.Choices) == 0 {
		return nil, fmt.Errorf("empty choices")
	}
	n.logger.Info("Response from neuro", "content", nr.Choices[0].Message.Content)

	var sb strings.Builder
	sb.WriteString(msg.Text)
	sb.WriteString("\n")
	sb.WriteString(nr.Choices[0].Message.Content)
	n.logger.Info("After neuro processing string", "result", sb.String())

	msg.Text = sb.String()

	return msg, nil
}
