package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mostaql-notification/internal/domain/repositories"
	"github.com/mostaql-notification/internal/domain/services"
)

// CountEmailsUseCase handles the /count command.
type CountEmailsUseCase struct {
	providers repositories.ProviderRepository
	messages  repositories.EmailMessageRepository
	formatter *services.NotificationFormatter
	logger    *slog.Logger
}

// NewCountEmailsUseCase creates a new CountEmailsUseCase.
func NewCountEmailsUseCase(
	providers repositories.ProviderRepository,
	messages repositories.EmailMessageRepository,
	formatter *services.NotificationFormatter,
	logger *slog.Logger,
) *CountEmailsUseCase {
	return &CountEmailsUseCase{
		providers: providers,
		messages:  messages,
		formatter: formatter,
		logger:    logger,
	}
}

// Execute counts matching emails for a provider and returns formatted output.
func (uc *CountEmailsUseCase) Execute(ctx context.Context, providerName string, filter repositories.CountFilter) (string, error) {
	provider, err := uc.providers.GetByName(ctx, providerName)
	if err != nil {
		return "", fmt.Errorf("looking up provider: %w", err)
	}
	if provider == nil {
		return fmt.Sprintf("❌ Provider '%s' not found.", providerName), nil
	}

	counts, err := uc.messages.CountByProvider(ctx, provider.ID, filter)
	if err != nil {
		return "", fmt.Errorf("counting emails: %w", err)
	}

	if len(counts) == 0 {
		return fmt.Sprintf("📭 %s\n\nNo matching emails found.", providerName), nil
	}

	// Calculate total.
	var total int
	var formatterCounts []services.CountResult
	for _, c := range counts {
		total += c.Count
		formatterCounts = append(formatterCounts, services.CountResult{
			AccountEmail: c.AccountEmail,
			Count:        c.Count,
			LatestAt:     c.LatestAt,
		})
	}

	return uc.formatter.FormatCount(provider.Name, formatterCounts, total), nil
}

// GetLatestEmailsUseCase handles the /latest command.
type GetLatestEmailsUseCase struct {
	providers repositories.ProviderRepository
	messages  repositories.EmailMessageRepository
	formatter *services.NotificationFormatter
	logger    *slog.Logger
}

// NewGetLatestEmailsUseCase creates a new GetLatestEmailsUseCase.
func NewGetLatestEmailsUseCase(
	providers repositories.ProviderRepository,
	messages repositories.EmailMessageRepository,
	formatter *services.NotificationFormatter,
	logger *slog.Logger,
) *GetLatestEmailsUseCase {
	return &GetLatestEmailsUseCase{
		providers: providers,
		messages:  messages,
		formatter: formatter,
		logger:    logger,
	}
}

// Execute retrieves the latest emails for a provider and returns formatted output.
func (uc *GetLatestEmailsUseCase) Execute(ctx context.Context, providerName string, limit int) (string, error) {
	provider, err := uc.providers.GetByName(ctx, providerName)
	if err != nil {
		return "", fmt.Errorf("looking up provider: %w", err)
	}
	if provider == nil {
		return fmt.Sprintf("❌ Provider '%s' not found.", providerName), nil
	}

	if limit <= 0 {
		limit = 5
	}

	messages, err := uc.messages.GetLatestByProvider(ctx, provider.ID, limit)
	if err != nil {
		return "", fmt.Errorf("getting latest emails: %w", err)
	}

	return uc.formatter.FormatLatest(provider.Name, messages), nil
}

// GetStatusUseCase handles the /status command.
type GetStatusUseCase struct {
	accounts      repositories.GmailAccountRepository
	providers     repositories.ProviderRepository
	notifications repositories.NotificationRepository
	logger        *slog.Logger
}

// NewGetStatusUseCase creates a new GetStatusUseCase.
func NewGetStatusUseCase(
	accounts repositories.GmailAccountRepository,
	providers repositories.ProviderRepository,
	notifications repositories.NotificationRepository,
	logger *slog.Logger,
) *GetStatusUseCase {
	return &GetStatusUseCase{
		accounts:      accounts,
		providers:     providers,
		notifications: notifications,
		logger:        logger,
	}
}

// SystemStatus holds the current status of the monitoring system.
type SystemStatus struct {
	AccountCount         int
	ActiveAccountCount   int
	ErrorAccountCount    int
	ProviderCount        int
	PendingNotifications int
	LastSyncAt           string
}

// Execute gathers the current system status.
func (uc *GetStatusUseCase) Execute(ctx context.Context) (*SystemStatus, error) {
	allAccounts, err := uc.accounts.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching accounts: %w", err)
	}

	activeAccounts, err := uc.accounts.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching active accounts: %w", err)
	}

	providers, err := uc.providers.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching providers: %w", err)
	}

	pendingCount, err := uc.notifications.GetPendingCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting pending notifications: %w", err)
	}

	// Find the most recent sync time.
	var errorCount int
	var lastSync string
	for _, a := range allAccounts {
		if a.Status == "auth_error" {
			errorCount++
		}
		if a.LastSyncAt != nil {
			syncStr := a.LastSyncAt.Format("02 Jan 2006 15:04:05")
			if syncStr > lastSync {
				lastSync = syncStr
			}
		}
	}
	if lastSync == "" {
		lastSync = "Never"
	}

	return &SystemStatus{
		AccountCount:         len(allAccounts),
		ActiveAccountCount:   len(activeAccounts),
		ErrorAccountCount:    errorCount,
		ProviderCount:        len(providers),
		PendingNotifications: pendingCount,
		LastSyncAt:           lastSync,
	}, nil
}

// FormatStatus formats the system status for display.
func (s *SystemStatus) FormatStatus() string {
	statusEmoji := "🟢"
	if s.ErrorAccountCount > 0 {
		statusEmoji = "🟡"
	}
	if s.ActiveAccountCount == 0 {
		statusEmoji = "🔴"
	}

	return fmt.Sprintf(`%s Gmail Monitor

Accounts: %d (active: %d)
Providers: %d

Last successful synchronization:
%s

Pending notifications: %d
Account errors: %d`,
		statusEmoji,
		s.AccountCount, s.ActiveAccountCount,
		s.ProviderCount,
		s.LastSyncAt,
		s.PendingNotifications,
		s.ErrorAccountCount,
	)
}
