package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"time"
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
	if err := r.client.Set(r.ctx, key, data, 48*time.Hour).Err(); err != nil {
		r.logger.Error("Redis set failed", key, "Redis set err", err)
		return err
	}
	r.logger.Info("Redis set succeeded", key)
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
	r.logger.Info("Redis GET ALL predictions", "res", fmt.Sprintf("%s", res))
	return parseSearchResult(res)
}

func parseSearchResult(res interface{}) ([]domain.Prediction, error) {
	arr, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", res)
	}

	preds := make([]domain.Prediction, 0, len(arr)/2)
	// начинаем с 1, т.к. arr[0] = totalCount
	for i := 1; i+1 < len(arr); i += 2 {
		rawDoc := arr[i+1]
		var data []byte

		switch v := rawDoc.(type) {
		case []byte:
			data = v
		case string:
			data = []byte(v)
		default:
			return preds, fmt.Errorf("unexpected document type: %T", rawDoc)
		}

		var p domain.Prediction
		if err := json.Unmarshal(data, &p); err != nil {
			return preds, fmt.Errorf("failed to unmarshal prediction JSON: %w", err)
		}

		preds = append(preds, p)
	}
	fmt.Println("Redis GET ALL predictions", "res", fmt.Sprintf("%s", res))
	return preds, nil
}
