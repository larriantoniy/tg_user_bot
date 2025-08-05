package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

type PredictionRepo struct {
	client *redis.Client
	ctx    context.Context
	logger *slog.Logger
}

func NewPredictionRepo(addr, password string, db int, logger *slog.Logger) ports.PredictionRepo {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &PredictionRepo{
		client: rdb,
		ctx:    context.Background(),
		logger: logger,
	}
}

func (r *PredictionRepo) Save(pred *domain.Prediction) error {
	key := fmt.Sprintf("prediction:%s", pred.ID)
	data, err := json.Marshal(pred)
	if err != nil {
		r.logger.Error("Redis", key, "marshaling error", err)
		return err
	}
	// TTL = 48 часов
	res, err := r.client.Do(r.ctx, "JSON.SET", key, "$", string(data)).Result()
	if err != nil {
		r.logger.Error("Redis JSON.SET failed", "key", key, "err", err)
		return err
	}
	r.logger.Info("Redis set succeeded", key, "res", res)
	return nil
}

func (r *PredictionRepo) FindByText(query string) ([]domain.Prediction, error) {
	// FT.SEARCH idx:predictions "манчестер"
	args := []interface{}{"FT.SEARCH", "idx:predictions", query, "LIMIT", "0", "100"}
	res, err := r.client.Do(r.ctx, args...).Result()
	if err != nil {
		r.logger.Error("Redis find %query failed", query, "err", err)
		return nil, err
	}
	// разбираем ответ: [total, key1, [json1], key2, [json2], ...]
	return parseSearchResult(res)
}

func (r *PredictionRepo) GetAll() ([]domain.Prediction, error) {
	// Выполняем полнотекстовый запрос на получение всех записей
	args := []interface{}{"FT.SEARCH", "idx:predictions", "*", "LIMIT", "0", "1000"}
	res, err := r.client.Do(r.ctx, args...).Result()
	if err != nil {
		r.logger.Error("Redis GET ALL predictions failed", "err", err)
		return nil, err
	}
	switch resTyped := res.(type) {
	case []interface{}:
		r.logger.Info("FT.SEARCH returned []interface{}", "len", len(resTyped))
		return parseSearchResult(resTyped)
	default:
		r.logger.Error("FT.SEARCH returned unexpected type", "type", fmt.Sprintf("%T", res))
		b, _ := json.MarshalIndent(res, "", "  ")
		r.logger.Error("Raw Redis result", "json", string(b))
		return nil, fmt.Errorf("unexpected result type: %T", res)
	}

}

func parseSearchResult(res interface{}) ([]domain.Prediction, error) {
	arr, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", res)
	}

	preds := make([]domain.Prediction, 0, len(arr)/2)

	// начинаем с 1, потому что arr[0] — это общее число найденных документов
	for i := 1; i+1 < len(arr); i += 2 {
		docFields, ok := arr[i+1].([]interface{})
		if !ok || len(docFields) != 2 {
			return preds, fmt.Errorf("unexpected document format at index %d: %T", i+1, arr[i+1])
		}

		// docFields[1] — это строка JSON
		jsonStr, ok := docFields[1].(string)
		if !ok {
			return preds, fmt.Errorf("unexpected json payload type: %T", docFields[1])
		}

		var p domain.Prediction
		if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
			return preds, fmt.Errorf("failed to unmarshal prediction JSON: %w", err)
		}

		preds = append(preds, p)
	}

	return preds, nil
}
