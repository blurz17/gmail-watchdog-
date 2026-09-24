package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/mostaql-notification/internal/application"
	"github.com/mostaql-notification/internal/config"
	"github.com/mostaql-notification/internal/domain/services"
	"github.com/mostaql-notification/internal/infrastructure/gmail"
	"github.com/mostaql-notification/internal/infrastructure/postgres"
	"github.com/mostaql-notification/internal/infrastructure/scheduler"
	"github.com/mostaql-notification/internal/infrastructure/telegram"
	httpapi "github.com/mostaql-notification/internal/interfaces/http"
)

func main() {
	// Load .env file if present (development convenience).
	_ = godotenv.Load()

	// Set up structured logger.
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Configuration ──────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger.Info("configuration loaded",
		"env", cfg.App.Env,
		"timezone", cfg.App.Timezone.String(),
		"poll_interval", cfg.Monitor.PollInterval.String(),
	)

	// ── Database ───────────────────────────────────────────────
	db, err := postgres.New(ctx, cfg.Database.URL, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	migrator := postgres.NewMigrator(db, logger)
	if err := migrator.Up(ctx, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// ── Repositories ───────────────────────────────────────────
	accountRepo := postgres.NewGmailAccountRepo(db)
	providerRepo := postgres.NewProviderRepo(db)
	senderRuleRepo := postgres.NewSenderRuleRepo(db)
	emailMessageRepo := postgres.NewEmailMessageRepo(db)
	notificationRepo := postgres.NewNotificationRepo(db)

	// ── Domain Services ────────────────────────────────────────
	matcher := services.NewSenderMatcher()
	formatter := services.NewNotificationFormatter(cfg.App.Timezone)

	// ── Infrastructure Adapters ────────────────────────────────
	oauthMgr := gmail.NewOAuthManager(logger)
	gmailClient := gmail.NewClient(oauthMgr, logger)
	telegramSender := telegram.NewSender(cfg.Telegram.BotToken, cfg.Telegram.ChatID, logger)

	// ── Application Use Cases ──────────────────────────────────
	syncGmailUC := application.NewSyncGmailUseCase(
		accountRepo, providerRepo, senderRuleRepo,
		emailMessageRepo, notificationRepo,
		gmailClient, matcher, formatter, logger,
	)

	retryPolicy := application.RetryPolicy{
		MaxAttempts:    cfg.Retry.MaxAttempts,
		InitialBackoff: cfg.Retry.InitialBackoff,
		MaxBackoff:     cfg.Retry.MaxBackoff,
		BackoffFactor:  2.0,
		JitterFactor:   0.1,
	}

	deliverUC := application.NewDeliverNotificationsUseCase(
		notificationRepo, emailMessageRepo,
		telegramSender, retryPolicy, logger,
	)

	countUC := application.NewCountEmailsUseCase(
		providerRepo, emailMessageRepo, formatter, logger,
	)

	latestUC := application.NewGetLatestEmailsUseCase(
		providerRepo, emailMessageRepo, formatter, logger,
	)

	statusUC := application.NewGetStatusUseCase(
		accountRepo, providerRepo, notificationRepo, logger,
	)

	// ── HTTP Server ────────────────────────────────────────────
	httpServer := httpapi.NewServer(
		cfg.HTTP.Port, db, statusUC, cfg.HTTP.APIKey, logger,
	)

	// ── Dashboard (Web UI) ─────────────────────────────────────
	dashboard := httpapi.NewDashboard(
		accountRepo, providerRepo, senderRuleRepo,
		cfg.Gmail.ClientID, cfg.Gmail.ClientSecret,
		cfg.HTTP.APIKey, logger,
	)
	dashboard.RegisterRoutes(httpServer.Mux())

	httpServer.Start()

	// ── Telegram Bot ───────────────────────────────────────────
	bot := telegram.NewBot(
		telegramSender,
		countUC, latestUC, statusUC,
		providerRepo, accountRepo,
		cfg.Telegram.ChatID,
		logger,
	)
	go bot.Start(ctx)

	// ── Scheduler ──────────────────────────────────────────────
	sched := scheduler.New(logger)

	sched.Add(scheduler.Task{
		Name:     "gmail_sync",
		Interval: cfg.Monitor.PollInterval,
		Fn:       syncGmailUC.Execute,
	})

	sched.Add(scheduler.Task{
		Name:     "notification_delivery",
		Interval: cfg.Monitor.NotificationPollInterval,
		Fn:       deliverUC.Execute,
	})

	sched.Start(ctx)

	// ── Ready ──────────────────────────────────────────────────
	logger.Info("gmail notification service started",
		"env", cfg.App.Env,
		"poll_interval", cfg.Monitor.PollInterval.String(),
		"http_port", cfg.HTTP.Port,
	)

	// ── Graceful Shutdown ──────────────────────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("shutdown signal received", "signal", sig.String())

	// Stop in reverse order.
	sched.Stop()
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*1e9)
	defer shutdownCancel()
	_ = httpServer.Stop(shutdownCtx)

	logger.Info("gmail notification service stopped")
	return nil
}
