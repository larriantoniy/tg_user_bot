package tdlib

import (
	"encoding/base64"
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
	"github.com/zelenin/go-tdlib/client"
	"log/slog"
	"os"
	"strings"
	"time"
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
	if _, err := client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	}); err != nil {
		logger.Error("TDLib SetLogVerbosity level", "error", err)
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
	t.logger.Info("Join Channels:", chs)

	joinedChs, err := t.GetJoinedChannels()
	if err != nil {
		t.logger.Error("Failed to fetch joined channels, aborting", "error", err)
		return
	}
	t.logger.Info("Already joined channels:", joinedChs)

	for _, ch := range chs {
		// 1) Пропускаем, если уже присоединились
		if _, isJoined := joinedChs[ch]; isJoined {
			t.logger.Debug("Already a member, skipping", "channel", ch)
			continue
		}

		// 2) Если username (@name) — подписываемся через JoinChannel
		if strings.HasPrefix(ch, "@") {
			if err := t.JoinChannel(ch); err != nil {
				t.logger.Error("Failed to join by username", "channel", ch, "error", err)
			} else {
				t.logger.Info("Joined channel by username", "channel", ch)
			}
			continue
		}

		// 3) Иначе — это, скорее всего, invite link
		t.logger.Info("Joining by invite link", "link", ch)
		_, err := t.client.JoinChatByInviteLink(&client.JoinChatByInviteLinkRequest{
			InviteLink: ch,
		})
		if err != nil {
			t.logger.Error("Failed to join by invite link", "link", ch, "error", err)
		} else {
			t.logger.Info("Joined channel by invite link", "link", ch)
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
			t.logger.Debug("Received new message")
			if upd, ok := update.(*client.UpdateNewMessage); ok {
				_, err := t.processUpdateNewMessage(out, upd)
				if err != nil {
					t.logger.Error("Error process UpdateNewMessage msg content type", upd.Message.Content.MessageContentType())
				}
			}
			t.logger.Debug("Skipping new message is not UpdateNewMessage Type")
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

func (t *TDLibClient) GetJoinedChannels() (map[string]bool, error) {
	var (
		// для первой страницы используем максимально возможный order
		//@todo увеличить лимит через pagination
		limit int32 = 200 // TDLib рекомендует не запрашивать >100–200 за раз
	)
	channels := make(map[string]bool)

	for {
		// 1) Получаем страницу чатов из основного списка (ChatListMain)
		resp, err := t.client.GetChats(&client.GetChatsRequest{
			ChatList: &client.ChatListMain{},
			Limit:    limit,
		})
		if err != nil {
			t.logger.Error("GetChats failed", "error", err)
			return nil, err
		}
		if len(resp.ChatIds) == 0 {
			// больше страниц нет
			break
		}

		// 2) Для каждого chatID запрашиваем полные данные и отбираем только каналы
		for _, chatID := range resp.ChatIds {
			chat, err := t.client.GetChat(&client.GetChatRequest{ChatId: chatID})
			if err != nil {
				t.logger.Warn("GetChat failed, skipping", "chat_id", chatID, "chat title:", chat.Title, "error", err)
				continue
			}
			//Супергруппа (тип ChatTypeSupergroup с IsChannel=false) по умолчанию приватная, но может быть сделана публичной путём назначения username
			//medium.com
			//core.telegram.org
			//Проверка приватности сводится к получению объекта Supergroup и проверке поля Username: если строка пуста — группа приватная, иначе — публичная
			if sg, ok := chat.Type.(*client.ChatTypeSupergroup); ok && sg.IsChannel {
				channels[chat.Title] = true

			}
		}
	}

	return channels, nil
}

func (t *TDLibClient) getChatTitle(chatID int64) (string, error) {

	chat, err := t.client.GetChat(&client.GetChatRequest{
		ChatId: chatID,
	})
	if err != nil {
		return "", err
	}

	return chat.Title, nil
}

func (t *TDLibClient) processUpdateNewMessage(out chan domain.Message, upd *client.UpdateNewMessage) (<-chan domain.Message, error) {
	chatName, err := t.getChatTitle(upd.Message.ChatId)
	if err != nil {
		t.logger.Info("Error getting chat title", err)
		chatName = ""
	}
	switch content := upd.Message.Content.(type) {
	case *client.MessageText:
		return t.processMessageText(out, content, upd.Message.ChatId, chatName)
	case *client.MessagePhoto:
		return t.processMessagePhoto(out, content, upd.Message.ChatId, chatName)
	default:
		t.logger.Debug("cant switch type update")
		return out, nil
	}
}
func (t *TDLibClient) processMessagePhoto(out chan domain.Message, msg *client.MessagePhoto, msgChatId int64, ChatName string) (<-chan domain.Message, error) {
	var text string

	var photoFile string

	var best *client.PhotoSize
	for i, size := range msg.Photo.Sizes {
		if i == 0 || size.Width*size.Height > best.Width*best.Height {
			best = size
			photoFile = best.Photo.Remote.Id
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no photo sizes available")
	}
	if msg.Caption != nil {
		text = msg.Caption.Text
	}

	out <- domain.Message{
		ChatID:    msgChatId,
		Text:      text,
		ChatName:  ChatName,
		PhotoFile: photoFile,
	}
	return out, nil
}
func (t *TDLibClient) processMessageText(out chan domain.Message, msg *client.MessageText, msgChatId int64, ChatName string) (<-chan domain.Message, error) {
	out <- domain.Message{
		ChatID:   msgChatId,
		Text:     msg.Text.Text,
		ChatName: ChatName,
	}
	return out, nil
}

func (t *TDLibClient) GetPhotoBase64ById(photoId string) (string, error) {
	// 1. Регистрируем remote ID и получаем локальный file ID
	remoteFile, err := t.client.GetRemoteFile(&client.GetRemoteFileRequest{
		RemoteFileId: photoId,
	})
	if err != nil {
		return "", fmt.Errorf("GetRemoteFile failed: %w", err)
	}

	// 2. Начинаем опрашивать статус загрузки
	var fileInfo *client.File
	for {
		fileInfo, err = t.client.GetFile(&client.GetFileRequest{
			FileId: remoteFile.Id,
		})
		if err != nil {
			return "", fmt.Errorf("GetFile polling failed: %w", err)
		}
		if fileInfo.Local.IsDownloadingCompleted {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 3. Читаем файл из кеша TDLib
	data, err := os.ReadFile(fileInfo.Local.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fileInfo.Local.Path, err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
