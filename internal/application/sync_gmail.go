package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/repositories"
	"github.com/mostaql-notification/internal/domain/services"
)

// SyncGmailUseCase orchestrates the core Gmail monitoring loop.
// For each active Gmail account, it:
//  1. Ensures a valid access token (refreshes if needed)
//  2. Builds a search query from active sender rules
//  3. Fetches matching messages from Gmail
//  4. Checks for duplicates against the database
//  5. Persists new email messages and creates pending notifications (outbox)
type SyncGmailUseCase struct {
	accounts      repositories.GmailAccountRepository
	providers     repositories.ProviderRepository
	rules         repositories.SenderRuleRepository
	messages      repositories.EmailMessageRepository
	notifications repositories.NotificationRepository
	gmailClient   services.GmailClient
	matcher       *services.SenderMatcher
	formatter     *services.NotificationFormatter
	logger        *slog.Logger
}

// NewSyncGmailUseCase creates a new SyncGmailUseCase.
func NewSyncGmailUseCase(
	accounts repositories.GmailAccountRepository,
	providers repositories.ProviderRepository,
	rules repositories.SenderRuleRepository,
	messages repositories.EmailMessageRepository,
	notifications repositories.NotificationRepository,
	gmailClient services.GmailClient,
	matcher *services.SenderMatcher,
	formatter *services.NotificationFormatter,
	logger *slog.Logger,
) *SyncGmailUseCase {
	return &SyncGmailUseCase{
		accounts:      accounts,
		providers:     providers,
		rules:         rules,
		messages:      messages,
		notifications: notifications,
		gmailClient:   gmailClient,
		matcher:       matcher,
		formatter:     formatter,
		logger:        logger,
	}
}

// Execute runs one sync cycle across all active accounts.
func (uc *SyncGmailUseCase) Execute(ctx context.Context) error {
	// Load active sender rules once per cycle.
	activeRules, err := uc.rules.GetAllActive(ctx)
	if err != nil {
		return fmt.Errorf("loading sender rules: %w", err)
	}

	if len(activeRules) == 0 {
		uc.logger.Debug("no active sender rules configured, skipping sync")
		return nil
	}

	// Load active accounts.
	accounts, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("loading active accounts: %w", err)
	}

	if len(accounts) == 0 {
		uc.logger.Debug("no active gmail accounts, skipping sync")
		return nil
	}

	// Build a provider lookup map for formatting.
	providerMap, err := uc.buildProviderMap(ctx)
	if err != nil {
		return fmt.Errorf("building provider map: %w", err)
	}

	// Build the Gmail search query from sender rules.
	query := uc.buildSearchQuery(activeRules)

	uc.logger.Info("gmail sync started",
		"accounts", len(accounts),
		"rules", len(activeRules),
		"query", query,
	)

	var totalNew int
	var syncErrors []error

	for i := range accounts {
		account := &accounts[i]
		count, err := uc.syncAccount(ctx, account, query, activeRules, providerMap)
		if err != nil {
			uc.logger.Error("sync failed for account",
				"account", account.Email,
				"error", err,
			)
			syncErrors = append(syncErrors, fmt.Errorf("account %s: %w", account.Email, err))

			// If auth failed, mark the account.
			if isAuthError(err) {
				if statusErr := uc.accounts.UpdateStatus(ctx, account.ID, entities.AccountStatusAuthError); statusErr != nil {
					uc.logger.Error("failed to update account status", "account", account.Email, "error", statusErr)
				}
			}
			continue
		}
		totalNew += count
	}

	uc.logger.Info("gmail sync completed",
		"new_messages", totalNew,
		"errors", len(syncErrors),
	)

	if len(syncErrors) > 0 {
		return fmt.Errorf("%d account(s) had sync errors", len(syncErrors))
	}
	return nil
}

// syncAccount processes a single Gmail account.
func (uc *SyncGmailUseCase) syncAccount(
	ctx context.Context,
	account *entities.GmailAccount,
	query string,
	rules []entities.SenderRule,
	providerMap map[string]string,
) (int, error) {
	// Step 1: Ensure valid access token.
	if account.IsTokenExpired(5 * time.Minute) {
		newToken, expiry, err := uc.gmailClient.RefreshAccessToken(ctx, account)
		if err != nil {
			return 0, fmt.Errorf("refreshing token: %w", err)
		}
		account.AccessToken = newToken
		account.AccessTokenExpiry = expiry

		// Persist the new token.
		if err := uc.accounts.UpdateTokens(ctx, account.ID, newToken, expiry); err != nil {
			return 0, fmt.Errorf("persisting refreshed token: %w", err)
		}
	}

	// Step 2: Fetch matching messages (paginated).
	var newCount int
	pageToken := ""

	for {
		result, err := uc.gmailClient.FetchMessages(ctx, account, query, pageToken)
		if err != nil {
			return newCount, fmt.Errorf("fetching messages: %w", err)
		}

		// Step 3: Process each message.
		for _, gmailMsg := range result.Messages {
			count, err := uc.processMessage(ctx, account, &gmailMsg, rules, providerMap)
			if err != nil {
				uc.logger.Warn("failed to process message",
					"account", account.Email,
					"message_id", gmailMsg.MessageID,
					"error", err,
				)
				continue
			}
			newCount += count
		}

		// Continue to next page if available.
		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	// Step 4: Update sync state.
	now := time.Now()
	if err := uc.accounts.UpdateSyncState(ctx, account.ID, now, account.LastHistoryID); err != nil {
		uc.logger.Warn("failed to update sync state", "account", account.Email, "error", err)
	}

	if newCount > 0 {
		uc.logger.Info("new messages detected",
			"account", account.Email,
			"count", newCount,
		)
	}

	return newCount, nil
}

// processMessage checks a single Gmail message against sender rules, persists it,
// and creates a pending notification for each matching provider.
func (uc *SyncGmailUseCase) processMessage(
	ctx context.Context,
	account *entities.GmailAccount,
	gmailMsg *services.GmailMessage,
	rules []entities.SenderRule,
	providerMap map[string]string,
) (int, error) {
	// Extract the sender email address from the "From" header.
	senderEmail := extractEmailAddress(gmailMsg.From)

	// Match against sender rules.
	matches := uc.matcher.Match(senderEmail, rules)
	if len(matches) == 0 {
		return 0, nil
	}

	var newCount int

	for _, match := range matches {
		// Check for duplicates: (account_id, gmail_message_id, provider_id).
		exists, err := uc.messages.Exists(ctx, account.ID, gmailMsg.MessageID, match.ProviderID)
		if err != nil {
			return newCount, fmt.Errorf("checking duplicate: %w", err)
		}
		if exists {
			continue
		}

		// Create the email message record.
		emailMsg := entities.NewEmailMessage(
			account.ID,
			gmailMsg.MessageID,
			gmailMsg.ThreadID,
			match.ProviderID,
			gmailMsg.From,
			gmailMsg.Subject,
			gmailMsg.Snippet,
			gmailMsg.ReceivedAt,
			gmailMsg.IsUnread,
		)

		if err := uc.messages.Create(ctx, emailMsg); err != nil {
			return newCount, fmt.Errorf("persisting email message: %w", err)
		}

		// Resolve provider name for formatting.
		providerName := providerMap[match.ProviderID.String()]
		if providerName == "" {
			providerName = "Unknown"
		}

		// Format the notification content.
		content := uc.formatter.FormatNewEmail(emailMsg, account.Email, providerName)

		// Create a pending notification in the outbox.
		notification := entities.NewNotification(
			emailMsg.ID,
			match.ProviderID,
			account.ID,
			content,
		)

		if err := uc.notifications.Create(ctx, notification); err != nil {
			return newCount, fmt.Errorf("creating notification: %w", err)
		}

		uc.logger.Info("gmail message detected",
			"account", account.Email,
			"provider", providerName,
			"message_id", gmailMsg.MessageID,
			"subject", gmailMsg.Subject,
		)

		newCount++
	}

	return newCount, nil
}

// buildSearchQuery constructs a Gmail search query from active sender rules.
// Example output: "from:(support@heroku.com OR info@mostaql.com) is:unread"
func (uc *SyncGmailUseCase) buildSearchQuery(rules []entities.SenderRule) string {
	senders := make(map[string]bool)

	for _, rule := range rules {
		switch rule.MatchType {
		case entities.MatchTypeExact:
			if rule.SenderEmail != "" {
				senders[rule.SenderEmail] = true
			}
		case entities.MatchTypeDomain:
			if rule.SenderDomain != "" {
				// Gmail supports domain matching with from:@domain.com
				senders["@"+rule.SenderDomain] = true
			}
		}
	}

	if len(senders) == 0 {
		return ""
	}

	var parts []string
	for sender := range senders {
		parts = append(parts, sender)
	}

	// Build: from:(sender1 OR sender2 OR ...) is:unread
	query := "from:(" + strings.Join(parts, " OR ") + ") is:unread"
	return query
}

// buildProviderMap creates a lookup map from provider ID to provider name.
func (uc *SyncGmailUseCase) buildProviderMap(ctx context.Context) (map[string]string, error) {
	providers, err := uc.providers.GetActive(ctx)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string, len(providers))
	for _, p := range providers {
		m[p.ID.String()] = p.Name
	}
	return m, nil
}

// extractEmailAddress extracts the email address from a "From" header value.
// Handles formats like:
//   - "user@example.com"
//   - "Display Name <user@example.com>"
//   - "<user@example.com>"
func extractEmailAddress(from string) string {
	from = strings.TrimSpace(from)

	// Check for angle bracket format: "Name <email>"
	if start := strings.LastIndex(from, "<"); start != -1 {
		if end := strings.Index(from[start:], ">"); end != -1 {
			return strings.TrimSpace(from[start+1 : start+end])
		}
	}

	// Plain email address.
	return from
}

// isAuthError checks if an error is an authentication/authorization failure.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "token may be revoked") ||
		strings.Contains(errStr, "refreshing token")
}
