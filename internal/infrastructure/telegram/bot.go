package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mostaql-notification/internal/application"
	"github.com/mostaql-notification/internal/domain/repositories"
)

// Bot handles incoming Telegram commands via long polling.
type Bot struct {
	sender         *Sender
	countUC        *application.CountEmailsUseCase
	latestUC       *application.GetLatestEmailsUseCase
	statusUC       *application.GetStatusUseCase
	providerRepo   repositories.ProviderRepository
	accountRepo    repositories.GmailAccountRepository
	authorizedChat string
	httpClient     *http.Client
	logger         *slog.Logger
	lastUpdateID   int
}

// NewBot creates a new Telegram bot handler.
func NewBot(
	sender *Sender,
	countUC *application.CountEmailsUseCase,
	latestUC *application.GetLatestEmailsUseCase,
	statusUC *application.GetStatusUseCase,
	providerRepo repositories.ProviderRepository,
	accountRepo repositories.GmailAccountRepository,
	authorizedChat string,
	logger *slog.Logger,
) *Bot {
	return &Bot{
		sender:         sender,
		countUC:        countUC,
		latestUC:       latestUC,
		statusUC:       statusUC,
		providerRepo:   providerRepo,
		accountRepo:    accountRepo,
		authorizedChat: authorizedChat,
		httpClient:     &http.Client{Timeout: 35 * time.Second},
		logger:         logger,
	}
}

// Start begins the long-polling loop to receive commands.
func (b *Bot) Start(ctx context.Context) {
	b.logger.Info("telegram bot started, listening for commands")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("telegram bot stopping")
			return
		default:
			b.pollUpdates(ctx)
		}
	}
}

// --- Telegram API types ---

type telegramUpdate struct {
	UpdateID int             `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int            `json:"message_id"`
	Chat      telegramChat   `json:"chat"`
	Text      string         `json:"text"`
	From      *telegramUser  `json:"from"`
}

type telegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type telegramUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

// --- Polling ---

func (b *Bot) pollUpdates(ctx context.Context) {
	url := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=30&allowed_updates=[\"message\"]",
		telegramAPIBase, b.sender.botToken, b.lastUpdateID+1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		b.logger.Error("creating poll request failed", "error", err)
		time.Sleep(5 * time.Second)
		return
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return // Context cancelled, shutting down.
		}
		b.logger.Error("polling updates failed", "error", err)
		time.Sleep(5 * time.Second)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger.Error("reading poll response failed", "error", err)
		return
	}

	var updatesResp telegramUpdatesResponse
	if err := json.Unmarshal(body, &updatesResp); err != nil {
		b.logger.Error("parsing updates failed", "error", err)
		return
	}

	for _, update := range updatesResp.Result {
		b.lastUpdateID = update.UpdateID

		if update.Message == nil || update.Message.Text == "" {
			continue
		}

		// Only respond to the authorized chat.
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		if chatID != b.authorizedChat {
			b.logger.Warn("ignoring message from unauthorized chat",
				"chat_id", chatID,
				"text", update.Message.Text,
			)
			continue
		}

		b.handleCommand(ctx, chatID, update.Message.Text)
	}
}

// --- Command handling ---

func (b *Bot) handleCommand(ctx context.Context, chatID string, text string) {
	text = strings.TrimSpace(text)
	parts := strings.SplitN(text, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	b.logger.Info("bot command received", "command", command, "args", args)

	var response string
	var err error

	switch command {
	case "/start", "/help":
		response = b.handleHelp()
	case "/status":
		response, err = b.handleStatus(ctx)
	case "/providers":
		response, err = b.handleProviders(ctx)
	case "/accounts":
		response, err = b.handleAccounts(ctx)
	case "/count":
		response, err = b.handleCount(ctx, args)
	case "/latest":
		response, err = b.handleLatest(ctx, args)
	default:
		response = "❓ Unknown command. Type /help for available commands."
	}

	if err != nil {
		b.logger.Error("command handler error", "command", command, "error", err)
		response = fmt.Sprintf("⚠️ Error: %s", err.Error())
	}

	if _, err := b.sender.SendToChat(ctx, chatID, response); err != nil {
		b.logger.Error("failed to send bot response", "error", err)
	}
}

func (b *Bot) handleHelp() string {
	return `📧 Gmail Notification Bot

Available commands:

/help — Show this help message
/status — Show monitoring status
/providers — List configured providers
/accounts — List monitored Gmail accounts
/count <provider> — Count matching emails
/latest <provider> — Show latest matching emails`
}

func (b *Bot) handleStatus(ctx context.Context) (string, error) {
	status, err := b.statusUC.Execute(ctx)
	if err != nil {
		return "", err
	}
	return status.FormatStatus(), nil
}

func (b *Bot) handleProviders(ctx context.Context) (string, error) {
	providers, err := b.providerRepo.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching providers: %w", err)
	}

	if len(providers) == 0 {
		return "📭 No providers configured.", nil
	}

	result := "📋 Configured Providers:\n\n"
	for i, p := range providers {
		status := "✅"
		if !p.IsActive {
			status = "⏸️"
		}
		result += fmt.Sprintf("%d. %s %s (mode: %s)\n", i+1, status, p.Name, p.DetectionMode)
	}
	return result, nil
}

func (b *Bot) handleAccounts(ctx context.Context) (string, error) {
	accounts, err := b.accountRepo.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching accounts: %w", err)
	}

	if len(accounts) == 0 {
		return "📭 No Gmail accounts configured.", nil
	}

	result := "📋 Monitored Gmail Accounts:\n\n"
	for i, a := range accounts {
		status := "🟢"
		switch a.Status {
		case "auth_error":
			status = "🔴"
		case "disabled":
			status = "⏸️"
		}

		lastSync := "Never"
		if a.LastSyncAt != nil {
			lastSync = a.LastSyncAt.Format("02 Jan 15:04")
		}

		result += fmt.Sprintf("%d. %s %s\n   Last sync: %s\n\n", i+1, status, a.Email, lastSync)
	}
	return result, nil
}

func (b *Bot) handleCount(ctx context.Context, providerName string) (string, error) {
	if providerName == "" {
		return "⚠️ Usage: /count <provider>\nExample: /count Heroku", nil
	}

	filter := repositories.CountFilter{UnreadOnly: true}
	return b.countUC.Execute(ctx, providerName, filter)
}

func (b *Bot) handleLatest(ctx context.Context, providerName string) (string, error) {
	if providerName == "" {
		return "⚠️ Usage: /latest <provider>\nExample: /latest Mostaql", nil
	}

	return b.latestUC.Execute(ctx, providerName, 5)
}
