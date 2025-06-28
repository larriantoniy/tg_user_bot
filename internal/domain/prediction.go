package domain

import "time"

type Prediction struct {
	ID        string    `json:"id"`        // UUID,  часть ключа в Redis
	ChatID    int64     `json:"chatId"`    // откуда пришло сообщение
	RawText   string    `json:"text"`      // полный текст сообщения
	ChatName  string    `json:"chatName"`  // ссылка или имя чата
	CreatedAt time.Time `json:"createdAt"` // время парсинга
	EventDate time.Time `json:"eventDate"` // дата спортивного события
	Sport     string    `json:"sport"`     // вид спорта
}
