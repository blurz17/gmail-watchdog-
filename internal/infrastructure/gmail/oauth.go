package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
)

const (
	googleTokenURL = "https://oauth2.googleapis.com/token"
	// Refresh tokens 5 minutes before they expire to avoid mid-request failures.
	tokenExpiryMargin = 5 * time.Minute
)

// tokenResponse represents the JSON response from Google's OAuth token endpoint.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// OAuthManager handles OAuth 2.0 token lifecycle for Gmail accounts.
type OAuthManager struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewOAuthManager creates a new OAuthManager.
func NewOAuthManager(logger *slog.Logger) *OAuthManager {
	return &OAuthManager{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// RefreshAccessToken exchanges a refresh token for a new access token.
// This implements the OAuth 2.0 refresh token grant type.
func (m *OAuthManager) RefreshAccessToken(ctx context.Context, account *entities.GmailAccount) (string, time.Time, error) {
	data := url.Values{
		"client_id":     {account.OAuthClientID},
		"client_secret": {account.OAuthClientSecret},
		"refresh_token": {account.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sending token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("oauth token refresh failed",
			"account", account.Email,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return "", time.Time{}, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("parsing token response: %w", err)
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	m.logger.Info("oauth token refreshed",
		"account", account.Email,
		"expires_in", tokenResp.ExpiresIn,
	)

	return tokenResp.AccessToken, expiry, nil
}

// NeedsRefresh checks whether an account's access token needs refreshing.
func (m *OAuthManager) NeedsRefresh(account *entities.GmailAccount) bool {
	return account.IsTokenExpired(tokenExpiryMargin)
}
