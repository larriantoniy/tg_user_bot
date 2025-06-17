package domain

import "time"

type Prediction struct {
	ID        string    // UUID,  часть ключа в Redis
	ChatID    int64     // откуда пришло сообщение
	RawText   string    // полный текст сообщения
	CreatedAt time.Time // время парсинга
}
