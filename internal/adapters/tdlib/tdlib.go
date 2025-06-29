package tdlib

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/larriantoniy/tg_user_bot/internal/domain"
	"github.com/larriantoniy/tg_user_bot/internal/ports"
	"github.com/zelenin/go-tdlib/client"
	"image"
	"io"
	"log/slog"
	"net/http"
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
	// 1) Логируем входные данные
	t.logger.Info("JoinChannels called", "channels", chs)

	// 2) Получаем уже присоединённые
	joinedChs, err := t.GetJoinedChannelIdentifiers()
	if err != nil {
		t.logger.Error("Failed to fetch joined channels, aborting", "error", err)
		return
	}
	t.logger.Info("Already joined channels", "channels", joinedChs)

	// 3) Если ни одного канала нет — сразу выходим
	if len(chs) == 0 {
		t.logger.Info("No channels to join, exiting")
		return
	}

	// 4) Пробегаем по каждому каналу и логируем, что сейчас обрабатываем
	for _, ch := range chs {
		t.logger.Info("Processing channel", "channel", ch)

		// 4.1) Уже в канале?
		if _, isJoined := joinedChs[ch]; isJoined {
			t.logger.Info("Already a member, skipping", "channel", ch)
			continue
		}

		// 4.2) Username-канал
		if strings.HasPrefix(ch, "@") {
			t.logger.Info("Attempting JoinChannel by username", "channel", ch)
			if err := t.JoinChannel(ch); err != nil {
				t.logger.Error("Failed to join by username", "channel", ch, "error", err)
			} else {
				t.logger.Info("Successfully joined by username", "channel", ch)
			}
			continue
		}

		// 4.3) Invite-link
		t.logger.Info("Attempting JoinChatByInviteLink", "link", ch)
		if _, err := t.client.JoinChatByInviteLink(&client.JoinChatByInviteLinkRequest{
			InviteLink: ch,
		}); err != nil {
			t.logger.Error("Failed to join by invite link", "link", ch, "error", err)
		} else {
			t.logger.Info("Successfully joined by invite link", "link", ch)
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
					t.logger.Error("Error process UpdateNewMessage msg content type", "upd MessageContentType", upd.Message.Content.MessageContentType())
				}
			} else {
				t.logger.Debug("Skipping update: not UpdateNewMessage", "upd", upd)
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

func (t *TDLibClient) GetJoinedChannelIdentifiers() (map[string]bool, error) {
	const limit = 100
	identifiers := make(map[string]bool, limit)

	// 1. Получаем первые `limit` чатов из основного списка
	chatsResp, err := t.client.GetChats(&client.GetChatsRequest{
		ChatList: &client.ChatListMain{},
		Limit:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("GetChats failed: %w", err)
	}

	// 2. Обходим все chatID
	for _, chatID := range chatsResp.ChatIds {
		chat, err := t.client.GetChat(&client.GetChatRequest{ChatId: chatID})
		if err != nil {
			t.logger.Error("GetChat failed", "chat_id", chatID, "error", err)
			continue
		}

		switch ct := chat.Type.(type) {
		// канал или супергруппа
		case *client.ChatTypeSupergroup:
			// получение публичного @username
			sup, err := t.client.GetSupergroup(&client.GetSupergroupRequest{
				SupergroupId: ct.SupergroupId,
			})
			if err == nil && sup != nil && sup.Usernames != nil && sup.Usernames.ActiveUsernames != nil && len(sup.Usernames.ActiveUsernames) > 0 {
				identifiers["@"+sup.Usernames.ActiveUsernames[0]] = true
			}
		case *client.ChatTypePrivate:
			usr, err := t.client.GetUser(&client.GetUserRequest{
				UserId: ct.UserId,
			})
			if err != nil {
				t.logger.Error("GetUser failed", "user_id", ct.UserId, "error", err)
				continue
			}
			if usr != nil && usr.Usernames != nil && usr.Usernames.ActiveUsernames != nil && len(usr.Usernames.ActiveUsernames) > 0 {
				identifiers["@"+usr.Usernames.ActiveUsernames[0]] = true
			}

		default:
			// ничего не делаем
		}
	}

	return identifiers, nil
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
		t.logger.Debug("cant switch type update", "upd message MessageContentType()", upd.Message.Content.MessageContentType())
		return out, nil
	}
}
func (t *TDLibClient) processMessagePhoto(out chan domain.Message, msg *client.MessagePhoto, msgChatId int64, ChatName string) (<-chan domain.Message, error) {
	var text string

	var photoFileId string

	var best *client.PhotoSize
	for i, size := range msg.Photo.Sizes {
		if i == 0 || size.Width*size.Height > best.Width*best.Height {
			best = size
			photoFileId = best.Photo.Remote.Id
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no photo sizes available")
	}
	if msg.Caption != nil {
		text = msg.Caption.Text
	}
	photoFile, err := t.GetPhotoBase64ById(photoFileId)
	if err != nil {
		t.logger.Info("GetPhotoBase64ById", "err", err)
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
	return BuildDataURI(bytes.NewReader(data))

}

// BuildDataURI читает первые 512 байт для детектирования MIME,
// затем определяет формат через DecodeConfig и формирует Data URI.
func BuildDataURI(r io.Reader) (string, error) {
	// Читаем все байты (можно оптимизировать потоково)
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}

	// 1) Sniff MIME
	mimeType := http.DetectContentType(data[:min(512, len(data))]) // :contentReference[oaicite:9]{index=9}

	// 2) DecodeConfig для более точного формата

	if _, format, err := image.DecodeConfig(r); err == nil {
		mimeType = "image/" + format // :contentReference[oaicite:10]{index=10}
	}

	// 3) Base64 encode
	b64 := base64.StdEncoding.EncodeToString(data) // :contentReference[oaicite:11]{index=11}

	// 4) Собираем Data URI согласно RFC 2397
	return fmt.Sprintf("data:%s;base64,%s", mimeType, b64), nil // :contentReference[oaicite:12]{index=12}
}
