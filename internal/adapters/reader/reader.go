package reader

import (
	"context"
	"github.com/larriantoniy/tg_user_bot/internal/config"
	ocr "github.com/ranghetto/go_ocr_space"
	"log/slog"
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

func (r *Reader) Read(ctx context.Context, photoFile string) (string, error) {

	type out struct {
		text string
		err  error
	}
	ch := make(chan out, 1)

	go func() {
		// блокирующий вызов сторонней либы
		res, err := r.config.ParseFromBase64(photoFile)
		if err != nil {
			ch <- out{"", err}
			return
		}
		ch <- out{res.JustText(), nil}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err() // DeadlineExceeded/Cancelled
	case o := <-ch:
		return o.text, o.err
	}

}
