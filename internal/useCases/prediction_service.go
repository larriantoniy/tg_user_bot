package useCases

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
)

type PredictionService struct {
	repo          ports.PredictionRepo
	decimalRegexp *regexp.Regexp
	logger        *slog.Logger
	re            *regexp.Regexp
}

func NewPredictionService(repo ports.PredictionRepo, logger *slog.Logger) *PredictionService {
	return &PredictionService{
		repo:          repo,
		decimalRegexp: regexp.MustCompile(`\b\d+\.\d{2}\b`),
		logger:        logger,
		re: regexp.MustCompile(
			`•\s*Вид спорта:\s*([A-Za-z]+|n/a)\s*,?\s*•\s*Ставка:\s*(true|false)`,
		),
	}
}

func (s *PredictionService) Save(msg *domain.Message) error {
	s.logger.Info("Received message from chat:", msg.ChatName, "processing ...")
	pred := &domain.Prediction{
		ID:        uuid.New().String(),
		ChatName:  msg.ChatName,
		ChatID:    msg.ChatID,
		RawText:   msg.Text,
		Sport:     s.extractSport(msg.Text),
		CreatedAt: time.Now()}
	s.logger.Info("Saving", "prediction", pred)
	return s.repo.Save(pred)
}

func (s *PredictionService) GetAll() ([]domain.Prediction, error) {
	return s.repo.GetAll()
}

func (s *PredictionService) IsPrediction(input string) bool {

	match := s.re.FindStringSubmatch(input)
	// match[0] — вся строка, match[1] — вид спорта, match[2] — true/false :contentReference[oaicite:4]{index=4}

	if len(match) != 3 {
		// Нет совпадений или неправильный формат
		s.logger.Info("Message is not prediction", input)
		return false
	}
	if match[2] != "true" {
		// Если ставка false — ничего не возвращаем
		s.logger.Info("Message is not prediction", input)
		return false
	}
	s.logger.Info("Message is prediction", input)
	return true
}

func (s *PredictionService) extractSport(input string) string {
	match := s.re.FindStringSubmatch(input)
	return match[1]
}
