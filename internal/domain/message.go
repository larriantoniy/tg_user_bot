package domain

// Message описывает входящее сообщение из Telegram
type Message struct {
	ChatID int64
	Text   string
}
