package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/mostaql-notification/internal/domain/services"
)

const telegramAPIBase = "https://api.telegram.org/bot"

// Sender implements services.NotificationSender for Telegram.
type Sender struct {
	botToken   string
	chatID     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewSender creates a new Telegram notification sender.
func NewSender(botToken, chatID string, logger *slog.Logger) *Sender {
	return &Sender{
		botToken:   botToken,
		chatID:     chatID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// Compile-time check that Sender implements NotificationSender.
var _ services.NotificationSender = (*Sender)(nil)

// Send delivers a text message to the configured Telegram chat.
func (s *Sender) Send(ctx context.Context, content string) (string, error) {
	return s.sendMessage(ctx, s.chatID, content)
}

// Channel returns the channel type.
func (s *Sender) Channel() string {
	return "telegram"
}

// SendToChat delivers a text message to a specific chat (used by the bot handler).
func (s *Sender) SendToChat(ctx context.Context, chatID string, content string) (string, error) {
	return s.sendMessage(ctx, chatID, content)
}

// sendMessage sends a message via the Telegram Bot API.
func (s *Sender) sendMessage(ctx context.Context, chatID string, text string) (string, error) {
	url := fmt.Sprintf("%s%s/sendMessage", telegramAPIBase, s.botToken)

	// Telegram has a 4096 character limit per message.
	// If the content exceeds this, truncate with an indicator.
	if len(text) > 4000 {
		text = text[:4000] + "\n\n... (truncated)"
	}

	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading telegram response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("telegram rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse the response to get the message ID.
	var telegramResp struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &telegramResp); err != nil {
		return "", fmt.Errorf("parsing telegram response: %w", err)
	}

	if !telegramResp.OK {
		return "", fmt.Errorf("telegram returned ok=false: %s", string(respBody))
	}

	return fmt.Sprintf("%d", telegramResp.Result.MessageID), nil
}
