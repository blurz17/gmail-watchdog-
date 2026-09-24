package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/mostaql-notification/internal/config"
	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/infrastructure/postgres"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
	redirectURI    = "http://localhost:8090/callback"
	gmailScope     = "https://www.googleapis.com/auth/gmail.readonly"
	callbackPort   = "8090"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "auth":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gmail-monitor-cli auth <email>")
			fmt.Println("Example: gmail-monitor-cli auth account1@gmail.com")
			os.Exit(1)
		}
		email := os.Args[2]
		if err := runAuth(logger, email); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "add-provider":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gmail-monitor-cli add-provider <name> [sender1] [sender2] ...")
			fmt.Println("Example: gmail-monitor-cli add-provider Heroku support@heroku.com noreply@heroku.com")
			os.Exit(1)
		}
		name := os.Args[2]
		senders := os.Args[3:]
		if err := runAddProvider(logger, name, senders); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list-accounts":
		if err := runListAccounts(logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list-providers":
		if err := runListProviders(logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Gmail Notification Service — CLI Tool

Commands:
  auth <email>                              Authorize a Gmail account (opens browser)
  add-provider <name> [sender1] [sender2]   Add a provider with sender rules
  list-accounts                             List all Gmail accounts
  list-providers                            List all providers and their rules

Examples:
  gmail-monitor-cli auth account1@gmail.com
  gmail-monitor-cli add-provider Heroku support@heroku.com
  gmail-monitor-cli add-provider Mostaql info@mostaql.com notifications@mostaql.com
  gmail-monitor-cli list-accounts
  gmail-monitor-cli list-providers`)
}

// ─── Auth Command ────────────────────────────────────────────

func runAuth(logger *slog.Logger, email string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Gmail.ClientID == "" || cfg.Gmail.ClientSecret == "" {
		return fmt.Errorf("GMAIL_CLIENT_ID and GMAIL_CLIENT_SECRET must be set in .env")
	}

	// Build the authorization URL.
	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&login_hint=%s",
		googleAuthURL,
		url.QueryEscape(cfg.Gmail.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(gmailScope),
		url.QueryEscape(email),
	)

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           Gmail Account Authorization                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Authorizing: %s\n\n", email)
	fmt.Println("1. Open this URL in your browser:\n")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("2. Sign in with the Gmail account and click 'Continue'")
	fmt.Println("3. You'll be redirected back here automatically")
	fmt.Println()
	fmt.Println("Waiting for authorization callback...")

	// Start a local HTTP server to capture the OAuth callback.
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			http.Error(w, "Authorization failed: "+errMsg, http.StatusBadRequest)
			errChan <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h2>✅ Authorization successful!</h2><p>You can close this window.</p></body></html>`)
		codeChan <- code
	})

	server := &http.Server{Addr: ":" + callbackPort, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Wait for the callback or error.
	var authCode string
	select {
	case authCode = <-codeChan:
		fmt.Println("\n✅ Authorization code received!")
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authorization timed out after 5 minutes")
	}

	_ = server.Shutdown(context.Background())

	// Exchange the authorization code for tokens.
	fmt.Println("Exchanging authorization code for tokens...")

	accessToken, refreshToken, expiresIn, err := exchangeCode(authCode, cfg.Gmail.ClientID, cfg.Gmail.ClientSecret)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}

	fmt.Println("✅ Tokens received!")

	// Connect to database and store the account.
	ctx := context.Background()
	db, err := postgres.New(ctx, cfg.Database.URL, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Run migrations first.
	migrator := postgres.NewMigrator(db, logger)
	if err := migrator.Up(ctx, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	accountRepo := postgres.NewGmailAccountRepo(db)

	// Check if account already exists.
	existing, err := accountRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("checking existing account: %w", err)
	}

	if existing != nil {
		// Update the existing account with new tokens.
		existing.RefreshToken = refreshToken
		existing.AccessToken = accessToken
		existing.AccessTokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		existing.OAuthClientID = cfg.Gmail.ClientID
		existing.OAuthClientSecret = cfg.Gmail.ClientSecret
		existing.Status = entities.AccountStatusActive

		if err := accountRepo.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating account: %w", err)
		}
		fmt.Printf("\n✅ Account %s updated successfully!\n", email)
	} else {
		// Create new account.
		account := entities.NewGmailAccount(email, email, cfg.Gmail.ClientID, cfg.Gmail.ClientSecret, refreshToken)
		account.AccessToken = accessToken
		account.AccessTokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)

		if err := accountRepo.Create(ctx, account); err != nil {
			return fmt.Errorf("creating account: %w", err)
		}
		fmt.Printf("\n✅ Account %s added successfully!\n", email)
	}

	return nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func exchangeCode(code, clientID, clientSecret string) (accessToken, refreshToken string, expiresIn int, err error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.Post(googleTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", 0, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", 0, fmt.Errorf("parsing response: %w", err)
	}

	if tokenResp.RefreshToken == "" {
		return "", "", 0, fmt.Errorf("no refresh token received — try revoking access at https://myaccount.google.com/permissions and re-authorizing")
	}

	return tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn, nil
}

// ─── Add Provider Command ────────────────────────────────────

func runAddProvider(logger *slog.Logger, name string, senders []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()
	db, err := postgres.New(ctx, cfg.Database.URL, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	migrator := postgres.NewMigrator(db, logger)
	if err := migrator.Up(ctx, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	providerRepo := postgres.NewProviderRepo(db)
	senderRuleRepo := postgres.NewSenderRuleRepo(db)

	// Check if provider already exists.
	existing, err := providerRepo.GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("checking existing provider: %w", err)
	}

	var providerID uuid.UUID
	if existing != nil {
		providerID = existing.ID
		fmt.Printf("Provider '%s' already exists, adding sender rules...\n", name)
	} else {
		provider := entities.NewProvider(name, entities.DetectionModeUnread)
		if err := providerRepo.Create(ctx, provider); err != nil {
			return fmt.Errorf("creating provider: %w", err)
		}
		providerID = provider.ID
		fmt.Printf("✅ Provider '%s' created\n", name)
	}

	// Add sender rules.
	for _, sender := range senders {
		sender = strings.TrimSpace(sender)
		if sender == "" {
			continue
		}

		var rule *entities.SenderRule
		if strings.HasPrefix(sender, "@") {
			// Domain rule: @heroku.com
			rule = entities.NewDomainSenderRule(providerID, strings.TrimPrefix(sender, "@"))
			fmt.Printf("  ✅ Domain rule: *%s\n", sender)
		} else {
			// Exact email rule
			rule = entities.NewExactSenderRule(providerID, sender)
			fmt.Printf("  ✅ Sender rule: %s\n", sender)
		}

		if err := senderRuleRepo.Create(ctx, rule); err != nil {
			fmt.Printf("  ⚠️  Failed to add rule for %s: %v\n", sender, err)
		}
	}

	fmt.Println("\nDone!")
	return nil
}

// ─── List Commands ───────────────────────────────────────────

func runListAccounts(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()
	db, err := postgres.New(ctx, cfg.Database.URL, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	accountRepo := postgres.NewGmailAccountRepo(db)
	accounts, err := accountRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("fetching accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No Gmail accounts configured.")
		fmt.Println("Add one with: gmail-monitor-cli auth account@gmail.com")
		return nil
	}

	fmt.Println("Gmail Accounts:")
	fmt.Println()
	for i, a := range accounts {
		status := "🟢 Active"
		switch a.Status {
		case entities.AccountStatusAuthError:
			status = "🔴 Auth Error"
		case entities.AccountStatusDisabled:
			status = "⏸️  Disabled"
		}

		lastSync := "Never"
		if a.LastSyncAt != nil {
			lastSync = a.LastSyncAt.Format("02 Jan 2006 15:04:05")
		}

		fmt.Printf("  %d. %s  %s\n", i+1, a.Email, status)
		fmt.Printf("     Last sync: %s\n\n", lastSync)
	}

	return nil
}

func runListProviders(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()
	db, err := postgres.New(ctx, cfg.Database.URL, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	providerRepo := postgres.NewProviderRepo(db)
	senderRuleRepo := postgres.NewSenderRuleRepo(db)

	providers, err := providerRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("fetching providers: %w", err)
	}

	if len(providers) == 0 {
		fmt.Println("No providers configured.")
		fmt.Println("Add one with: gmail-monitor-cli add-provider Heroku support@heroku.com")
		return nil
	}

	fmt.Println("Providers:")
	fmt.Println()
	for i, p := range providers {
		status := "✅"
		if !p.IsActive {
			status = "⏸️"
		}

		fmt.Printf("  %d. %s %s (mode: %s)\n", i+1, status, p.Name, p.DetectionMode)

		rules, err := senderRuleRepo.GetByProviderID(ctx, p.ID)
		if err != nil {
			fmt.Printf("     ⚠️  Failed to load rules: %v\n", err)
			continue
		}

		for _, r := range rules {
			switch r.MatchType {
			case entities.MatchTypeExact:
				fmt.Printf("     → %s\n", r.SenderEmail)
			case entities.MatchTypeDomain:
				fmt.Printf("     → *@%s\n", r.SenderDomain)
			}
		}
		fmt.Println()
	}

	return nil
}
