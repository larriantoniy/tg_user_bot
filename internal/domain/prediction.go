package domain

import "time"

type Prediction struct {
	ID        string    // UUID,  часть ключа в Redis
	ChatID    int64     // откуда пришло сообщение
	RawText   string    // полный текст сообщения
	ChatName  string    // ссылка или имя чата
	CreatedAt time.Time // время парсинга
	Sport     string    // вид спорта
}
