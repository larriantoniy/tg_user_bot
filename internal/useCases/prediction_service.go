package useCases

import (
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
)

type PredictionService struct {
	repo          ports.PredictionRepo
	decimalRegexp *regexp.Regexp
	logger        *slog.Logger
}

func NewPredictionService(repo ports.PredictionRepo, logger *slog.Logger) *PredictionService {
	return &PredictionService{
		repo:          repo,
		decimalRegexp: regexp.MustCompile(`\b\d+\.\d{2}\b`),
		logger:        logger,
	}
}

func (s *PredictionService) Save(msg domain.Message) error {
	s.logger.Info("Received message:", msg, "processing")
	if !s.isPrediction(msg) {
		return nil
	}
	pred := &domain.Prediction{
		ID:        uuid.New().String(),
		ChatID:    msg.ChatID,
		RawText:   msg.Text,
		CreatedAt: time.Now()}
	s.logger.Info("Saving prediction", pred)
	return s.repo.Save(pred)
}

func (s *PredictionService) GetAll() ([]domain.Prediction, error) {
	s.logger.Info("Get All prediction")
	return s.repo.GetAll()
}

func (s *PredictionService) isPrediction(msg domain.Message) bool {
	low := strings.ToLower(msg.Text)
	if !strings.Contains(low, "прогноз") &&
		!strings.Contains(low, "коэффициент") &&
		!strings.Contains(low, "кф") &&
		!s.decimalRegexp.MatchString(msg.Text) {
		s.logger.Info("Message is not prediction")
		return false
	}
	s.logger.Info("Message is prediction")
	return true
}
