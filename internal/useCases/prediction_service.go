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
		re:            regexp.MustCompile(`(?i)Вид спорта:\s*([a-z]+|n/a)\s*,?\s*Ставка:\s*(true|false)\s*,?\s*Дата:\s*(\d{2}\.\d{2}\.\d{2})`),
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
		CreatedAt: time.Now(),
		EventDate: s.extractEventDate(msg.Text),
	}
	s.logger.Info("Saving", "prediction", pred)
	return s.repo.Save(pred)
}

func (s *PredictionService) GetAll() ([]domain.Prediction, error) {
	res, err := s.repo.GetAll()
	if err != nil {
		return res, err
	}
	var filterd []domain.Prediction
	now := time.Now()
	// возвращаем только те прогнозы которые еще не наступили по времени
	for _, pred := range res {
		if pred.EventDate.Before(now) {
			filterd = append(filterd, pred)
		}
	}
	return filterd, nil

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
func (s *PredictionService) extractEventDate(input string) time.Time {
	match := s.re.FindStringSubmatch(input)
	// Layout: день-месяц-год часы:минуты
	layout := "02-01-2006 15:04"
	t, err := time.Parse(layout, match[3])
	if err != nil {
		return time.Time{}
	}
	return t
}
