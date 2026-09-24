package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/services"
)

const (
	gmailAPIBase = "https://gmail.googleapis.com/gmail/v1"
)

// Client implements services.GmailClient using the Gmail REST API.
type Client struct {
	oauth      *OAuthManager
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Gmail API client.
func NewClient(oauth *OAuthManager, logger *slog.Logger) *Client {
	return &Client{
		oauth:      oauth,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// Compile-time check that Client implements GmailClient.
var _ services.GmailClient = (*Client)(nil)

// --- Gmail API response types ---

type gmailMessageListResponse struct {
	Messages           []gmailMessageRef `json:"messages"`
	NextPageToken      string            `json:"nextPageToken"`
	ResultSizeEstimate int               `json:"resultSizeEstimate"`
}

type gmailMessageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

type gmailMessageResponse struct {
	ID        string              `json:"id"`
	ThreadID  string              `json:"threadId"`
	LabelIDs  []string            `json:"labelIds"`
	Snippet   string              `json:"snippet"`
	Payload   gmailMessagePayload `json:"payload"`
	InternalDate string           `json:"internalDate"` // milliseconds since epoch
}

type gmailMessagePayload struct {
	Headers []gmailHeader `json:"headers"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// --- Interface implementation ---

func (c *Client) FetchMessages(ctx context.Context, account *entities.GmailAccount, query string, pageToken string) (*services.MessageListResult, error) {
	url := fmt.Sprintf("%s/users/me/messages?q=%s&maxResults=50", gmailAPIBase, urlEncode(query))
	if pageToken != "" {
		url += "&pageToken=" + urlEncode(pageToken)
	}

	body, err := c.doAuthorizedRequest(ctx, account, url)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}

	var listResp gmailMessageListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("parsing message list: %w", err)
	}

	// Fetch full details for each message.
	var messages []services.GmailMessage
	for _, ref := range listResp.Messages {
		msg, err := c.GetMessage(ctx, account, ref.ID)
		if err != nil {
			c.logger.Warn("failed to fetch message details, skipping",
				"account", account.Email,
				"message_id", ref.ID,
				"error", err,
			)
			continue
		}
		messages = append(messages, *msg)
	}

	return &services.MessageListResult{
		Messages:           messages,
		NextPageToken:      listResp.NextPageToken,
		ResultSizeEstimate: listResp.ResultSizeEstimate,
	}, nil
}

func (c *Client) GetMessage(ctx context.Context, account *entities.GmailAccount, messageID string) (*services.GmailMessage, error) {
	// Request only metadata + snippet to minimize API usage.
	url := fmt.Sprintf("%s/users/me/messages/%s?format=metadata&metadataHeaders=From&metadataHeaders=Subject&metadataHeaders=Date",
		gmailAPIBase, messageID)

	body, err := c.doAuthorizedRequest(ctx, account, url)
	if err != nil {
		return nil, fmt.Errorf("getting message %s: %w", messageID, err)
	}

	var msgResp gmailMessageResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return nil, fmt.Errorf("parsing message %s: %w", messageID, err)
	}

	return c.parseMessage(&msgResp), nil
}

func (c *Client) RefreshAccessToken(ctx context.Context, account *entities.GmailAccount) (string, time.Time, error) {
	return c.oauth.RefreshAccessToken(ctx, account)
}

// --- Internal helpers ---

// doAuthorizedRequest performs an authenticated GET request to the Gmail API.
// It automatically refreshes the access token if expired.
func (c *Client) doAuthorizedRequest(ctx context.Context, account *entities.GmailAccount, url string) ([]byte, error) {
	// Ensure we have a valid access token.
	if c.oauth.NeedsRefresh(account) {
		c.logger.Debug("access token expired, refreshing", "account", account.Email)
		newToken, expiry, err := c.oauth.RefreshAccessToken(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("refreshing token for %s: %w", account.Email, err)
		}
		// Update the in-memory account object. The caller is responsible for
		// persisting this to the database.
		account.AccessToken = newToken
		account.AccessTokenExpiry = expiry
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Token might have been revoked or expired between check and use.
		return nil, fmt.Errorf("gmail API returned 401 for %s: token may be revoked", account.Email)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("gmail API rate limited for %s (429)", account.Email)
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("gmail API server error for %s: %d", account.Email, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gmail API error for %s: status %d, body: %s", account.Email, resp.StatusCode, string(body))
	}

	return body, nil
}

// parseMessage converts a Gmail API message response into our domain GmailMessage.
func (c *Client) parseMessage(resp *gmailMessageResponse) *services.GmailMessage {
	msg := &services.GmailMessage{
		MessageID: resp.ID,
		ThreadID:  resp.ThreadID,
		Snippet:   resp.Snippet,
		IsUnread:  false,
	}

	// Extract headers.
	for _, h := range resp.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			msg.From = h.Value
		case "subject":
			msg.Subject = h.Value
		case "date":
			msg.ReceivedAt = parseEmailDate(h.Value)
		}
	}

	// Check if UNREAD label is present.
	for _, label := range resp.LabelIDs {
		if label == "UNREAD" {
			msg.IsUnread = true
			break
		}
	}

	// Fallback: use internalDate if Date header parsing failed.
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = parseInternalDate(resp.InternalDate)
	}

	return msg
}

// parseEmailDate tries multiple RFC formats used in email Date headers.
func parseEmailDate(dateStr string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseInternalDate converts Gmail's internalDate (ms since epoch) to time.Time.
func parseInternalDate(msStr string) time.Time {
	var ms int64
	if _, err := fmt.Sscanf(msStr, "%d", &ms); err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// urlEncode encodes a string for use in a URL query parameter.
func urlEncode(s string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(s, " ", "+"),
			"(", "%28"),
		")", "%29")
}
