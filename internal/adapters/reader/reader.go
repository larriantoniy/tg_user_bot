package reader

import (
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/config"
	ocr "github.com/ranghetto/go_ocr_space"
	"log/slog"
	"sync"
)

type Reader struct {
	config *ocr.Config

	logger *slog.Logger

	// заготовленный http.Request
}

func NewReader(cfg *config.Config, logger *slog.Logger) (*Reader, error) {

	config := ocr.InitConfig(cfg.ReaderToken, "eng", ocr.OCREngine2)

	// 3) Собираем объект Reader
	return &Reader{
		config: &config,
		logger: logger,
	}, nil
}

func (r *Reader) Read(photoFile string, wg *sync.WaitGroup) (string, error) {
	defer wg.Done()
	result, err := r.config.ParseFromBase64(photoFile)
	if err != nil {
		return "", fmt.Errorf("error ocr read: %w", err)
	}
	//printing the just the parsed text
	fmt.Println(result.JustText())

	if len(result.JustText()) == 0 {
		return "", fmt.Errorf("empty ParsedResults")
	}

	r.logger.Info("After reader processing", "result", result.JustText())

	return result.JustText(), nil
}
