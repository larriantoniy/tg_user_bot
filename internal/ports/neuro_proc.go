package ports

import (
	"context"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
)

type NeuroProccesor interface {
	GetCompletion(ctx context.Context, msg *domain.Message) (*domain.Message, error)
}
