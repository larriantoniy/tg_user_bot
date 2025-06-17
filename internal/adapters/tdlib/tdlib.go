package tdlib

import (
	"log/slog"

	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
	"github.com/zelenin/go-tdlib/client"
)

// TDLibClient реализует ports.TelegramClient через go-tdlib
type TDLibClient struct {
	client *client.Client
	logger *slog.Logger
	selfId int64
}

// NewClient создаёт и авторизует TDLib клиента
func NewClient(apiID int32, apiHash string, logger *slog.Logger) (ports.TelegramClient, error) {
	// Параметры TDLib
	tdParams := &client.SetTdlibParametersRequest{
		ApiId:              apiID,
		ApiHash:            apiHash,
		SystemLanguageCode: "en",
		DeviceModel:        "GoUserBot",
		ApplicationVersion: "0.1",
		UseMessageDatabase: true,
		UseFileDatabase:    false,
		DatabaseDirectory:  "./tdlib-db",
		FilesDirectory:     "./tdlib-files",
	}

	// Авторизатор и CLI-интерактор
	authorizer := client.ClientAuthorizer(tdParams)
	go client.CliInteractor(authorizer)

	// Создаём клиента
	tdClient, err := client.NewClient(authorizer)
	if err != nil {
		logger.Error("TDLib NewClient error", "error", err)
		return nil, err
	}
	// Получаем информацию о себе (боте) — понадобится для GetChatMember
	me, err := tdClient.GetMe()
	if err != nil {
		logger.Error("GetMe failed", "error", err)
		return nil, err
	}
	logger.Info("TDLib client initialized and authorized", "self_id", me.Id)
	return &TDLibClient{client: tdClient, logger: logger, selfId: me.Id}, nil
}

// JoinChannel подписывается на публичный канал по его username, если ещё не подписан
func (t *TDLibClient) JoinChannel(username string) error {
	// Ищем чат по username
	chat, err := t.client.SearchPublicChat(&client.SearchPublicChatRequest{
		Username: username,
	})
	if err != nil {
		t.logger.Error("SearchPublicChat failed", "username", username, "error", err)
		return err
	}

	// Пытаемся подписаться
	_, err = t.client.JoinChat(&client.JoinChatRequest{
		ChatId: chat.Id,
	})
	if err != nil {
		// Telegram вернёт ошибку, если уже в канале — можно логировать как инфо
		t.logger.Error("JoinChat failed", "chat_id", chat.Id, "error", err)
		return err
	}

	t.logger.Info("Joined channel", "channel", username)
	return nil
}
func (t *TDLibClient) JoinChannels(chs []string) {
	for _, ch := range chs {
		isMember, err := t.IsChannelMember(ch)
		if err != nil {
			t.logger.Error("Failed to check membership", "channel", ch, "error", err)
			continue
		}
		if isMember {
			t.logger.Info("Already a member, skipping join", "channel", ch)
			continue
		}

		if err := t.JoinChannel(ch); err != nil {
			// Если ошибка содержит USER_ALREADY_PARTICIPANT — просто логируем и продолжаем
			t.logger.Error("Failed to join channel", "channel", ch, "error", err)
		} else {
			t.logger.Info("Successfully joined channel", "channel", ch)
		}
	}

}

// Listen возвращает канал доменных сообщений из TDLib и запускает обработку обновлений
func (t *TDLibClient) Listen() (<-chan domain.Message, error) {
	out := make(chan domain.Message)

	// Получаем слушатель обновлений
	listener := t.client.GetListener()
	go func() {
		defer close(out)
		for update := range listener.Updates {
			// Фильтруем только новые сообщения
			if upd, ok := update.(*client.UpdateNewMessage); ok {
				if textMsg, ok := upd.Message.Content.(*client.MessageText); ok {
					out <- domain.Message{
						ChatID: upd.Message.ChatId,
						Text:   textMsg.Text.Text,
					}
					t.logger.Debug("Dispatched message", "chat_id", upd.Message.ChatId)
				}
			}
		}
	}()

	return out, nil
}

func (t *TDLibClient) IsChannelMember(username string) (bool, error) {
	//  Нахождение чата
	chat, err := t.client.SearchPublicChat(&client.SearchPublicChatRequest{
		Username: username,
	})
	if err != nil {
		t.logger.Error("SearchPublicChat failed", "username", username, "error", err)
		return false, err
	}

	//  Получаем информацию об участнике

	member, err := t.client.GetChatMember(&client.GetChatMemberRequest{
		ChatId:   chat.Id,
		MemberId: &client.MessageSenderUser{UserId: t.selfId},
	})
	if err != nil {
		t.logger.Debug("GetChatMember failed, assuming not a member", "chat_id", chat.Id, "error", err)
		return false, nil
	}

	//  Определяем статус через type assertion
	switch member.Status.(type) {
	case *client.ChatMemberStatusMember, *client.ChatMemberStatusAdministrator, *client.ChatMemberStatusCreator:
		t.logger.Debug("Bot is channel member", "chat_id", chat.Id)
		return true, nil
	default:
		t.logger.Debug("Bot not member", "chat_id", chat.Id)
		return false, nil
	}
}
