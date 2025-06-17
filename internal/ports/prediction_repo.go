package ports

import "github.com/larriantoniy/tg_user_bot/internal/domain"

type PredictionRepo interface {
	Save(pred *domain.Prediction) error
	GetAll() ([]domain.Prediction, error)
	FindByText(query string) ([]domain.Prediction, error)
}
