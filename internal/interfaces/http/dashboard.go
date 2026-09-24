package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/repositories"
)

// Dashboard handles the web UI for managing accounts, providers, and rules.
type Dashboard struct {
	accountRepo    repositories.GmailAccountRepository
	providerRepo   repositories.ProviderRepository
	senderRuleRepo repositories.SenderRuleRepository
	gmailClientID  string
	gmailSecret    string
	apiKey         string
	logger         *slog.Logger
}

// NewDashboard creates a new Dashboard handler.
func NewDashboard(
	accountRepo repositories.GmailAccountRepository,
	providerRepo repositories.ProviderRepository,
	senderRuleRepo repositories.SenderRuleRepository,
	gmailClientID, gmailSecret, apiKey string,
	logger *slog.Logger,
) *Dashboard {
	return &Dashboard{
		accountRepo:    accountRepo,
		providerRepo:   providerRepo,
		senderRuleRepo: senderRuleRepo,
		gmailClientID:  gmailClientID,
		gmailSecret:    gmailSecret,
		apiKey:         apiKey,
		logger:         logger,
	}
}

// RegisterRoutes registers dashboard routes on the given mux.
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard", d.authMiddleware(d.handleDashboard))
	mux.HandleFunc("POST /dashboard/providers", d.authMiddleware(d.handleAddProvider))
	mux.HandleFunc("POST /dashboard/providers/delete", d.authMiddleware(d.handleDeleteProvider))
	mux.HandleFunc("POST /dashboard/rules", d.authMiddleware(d.handleAddRule))
	mux.HandleFunc("POST /dashboard/rules/delete", d.authMiddleware(d.handleDeleteRule))
	mux.HandleFunc("GET /dashboard/oauth/start", d.authMiddleware(d.handleOAuthStart))
	mux.HandleFunc("GET /dashboard/oauth/callback", d.handleOAuthCallback)
	mux.HandleFunc("POST /dashboard/accounts/delete", d.authMiddleware(d.handleDeleteAccount))
}

// authMiddleware checks for API key via query param or cookie.
func (d *Dashboard) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key != "" && key == d.apiKey {
			http.SetCookie(w, &http.Cookie{
				Name:     "dashboard_key",
				Value:    key,
				Path:     "/dashboard",
				HttpOnly: true,
				MaxAge:   86400 * 7,
			})
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		cookie, err := r.Cookie("dashboard_key")
		if err != nil || cookie.Value != d.apiKey {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, loginHTML)
			return
		}

		next(w, r)
	}
}

type dashboardData struct {
	Accounts  []entities.GmailAccount
	Providers []providerWithRules
	Flash     string
	FlashType string
}

type providerWithRules struct {
	Provider entities.Provider
	Rules    []entities.SenderRule
}

func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accounts, _ := d.accountRepo.GetAll(ctx)
	providers, _ := d.providerRepo.GetAll(ctx)

	var pwrs []providerWithRules
	for _, p := range providers {
		rules, _ := d.senderRuleRepo.GetByProviderID(ctx, p.ID)
		pwrs = append(pwrs, providerWithRules{Provider: p, Rules: rules})
	}

	data := dashboardData{
		Accounts:  accounts,
		Providers: pwrs,
		Flash:     r.URL.Query().Get("flash"),
		FlashType: r.URL.Query().Get("type"),
	}

	tmpl := template.Must(template.New("dashboard").Funcs(template.FuncMap{
		"statusEmoji": func(status string) string {
			switch status {
			case "active":
				return "🟢"
			case "auth_error":
				return "🔴"
			default:
				return "⏸️"
			}
		},
		"timeAgo": func(t *time.Time) string {
			if t == nil {
				return "Never"
			}
			return t.Format("02 Jan 15:04")
		},
		"ruleDisplay": func(r entities.SenderRule) string {
			if r.MatchType == entities.MatchTypeDomain {
				return "*@" + r.SenderDomain
			}
			return r.SenderEmail
		},
	}).Parse(dashboardHTML))

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		d.logger.Error("template render failed", "error", err)
	}
}

func (d *Dashboard) handleAddProvider(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/dashboard?flash=Provider+name+required&type=error", http.StatusSeeOther)
		return
	}

	provider := entities.NewProvider(name, entities.DetectionModeUnread)
	if err := d.providerRepo.Create(r.Context(), provider); err != nil {
		http.Redirect(w, r, "/dashboard?flash=Failed+to+create+provider&type=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard?flash=Provider+"+name+"+created&type=success", http.StatusSeeOther)
}

func (d *Dashboard) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/dashboard?flash=Invalid+ID&type=error", http.StatusSeeOther)
		return
	}
	_ = d.providerRepo.Delete(r.Context(), id)
	http.Redirect(w, r, "/dashboard?flash=Provider+deleted&type=success", http.StatusSeeOther)
}

func (d *Dashboard) handleAddRule(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	providerID, err := uuid.Parse(r.FormValue("provider_id"))
	if err != nil {
		http.Redirect(w, r, "/dashboard?flash=Invalid+provider&type=error", http.StatusSeeOther)
		return
	}

	value := strings.TrimSpace(r.FormValue("value"))
	if value == "" {
		http.Redirect(w, r, "/dashboard?flash=Sender+value+required&type=error", http.StatusSeeOther)
		return
	}

	var rule *entities.SenderRule
	if strings.HasPrefix(value, "@") {
		rule = entities.NewDomainSenderRule(providerID, strings.TrimPrefix(value, "@"))
	} else {
		rule = entities.NewExactSenderRule(providerID, value)
	}

	if err := d.senderRuleRepo.Create(r.Context(), rule); err != nil {
		http.Redirect(w, r, "/dashboard?flash=Failed+to+add+rule&type=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard?flash=Rule+added&type=success", http.StatusSeeOther)
}

func (d *Dashboard) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/dashboard?flash=Invalid+ID&type=error", http.StatusSeeOther)
		return
	}
	_ = d.senderRuleRepo.Delete(r.Context(), id)
	http.Redirect(w, r, "/dashboard?flash=Rule+deleted&type=success", http.StatusSeeOther)
}

func (d *Dashboard) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	if email == "" {
		http.Redirect(w, r, "/dashboard?flash=Email+required&type=error", http.StatusSeeOther)
		return
	}

	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/dashboard/oauth/callback", scheme, r.Host)

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=https://www.googleapis.com/auth/gmail.readonly&access_type=offline&prompt=consent&login_hint=%s&state=%s",
		d.gmailClientID, callbackURL, email, email,
	)

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (d *Dashboard) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	email := r.URL.Query().Get("state")

	if code == "" {
		errMsg := r.URL.Query().Get("error")
		http.Redirect(w, r, "/dashboard?flash=OAuth+failed:+"+errMsg+"&type=error", http.StatusSeeOther)
		return
	}

	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/dashboard/oauth/callback", scheme, r.Host)

	accessToken, refreshToken, expiresIn, err := exchangeOAuthCode(code, d.gmailClientID, d.gmailSecret, callbackURL)
	if err != nil {
		d.logger.Error("oauth exchange failed", "error", err)
		http.Redirect(w, r, "/dashboard?flash=Token+exchange+failed&type=error", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	existing, _ := d.accountRepo.GetByEmail(ctx, email)
	if existing != nil {
		existing.RefreshToken = refreshToken
		existing.AccessToken = accessToken
		existing.AccessTokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		existing.OAuthClientID = d.gmailClientID
		existing.OAuthClientSecret = d.gmailSecret
		existing.Status = entities.AccountStatusActive
		_ = d.accountRepo.Update(ctx, existing)
	} else {
		account := entities.NewGmailAccount(email, email, d.gmailClientID, d.gmailSecret, refreshToken)
		account.AccessToken = accessToken
		account.AccessTokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		_ = d.accountRepo.Create(ctx, account)
	}

	http.Redirect(w, r, "/dashboard?flash=Account+"+email+"+authorized!&type=success", http.StatusSeeOther)
}

func (d *Dashboard) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := uuid.Parse(r.FormValue("id"))
	if err != nil {
		http.Redirect(w, r, "/dashboard?flash=Invalid+ID&type=error", http.StatusSeeOther)
		return
	}
	_ = d.accountRepo.Delete(r.Context(), id)
	http.Redirect(w, r, "/dashboard?flash=Account+removed&type=success", http.StatusSeeOther)
}

func exchangeOAuthCode(code, clientID, clientSecret, redirectURI string) (string, string, int, error) {
	data := fmt.Sprintf("code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, clientID, clientSecret, redirectURI)

	resp, err := http.Post("https://oauth2.googleapis.com/token", "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", 0, err
	}

	if result.RefreshToken == "" {
		return "", "", 0, fmt.Errorf("no refresh token — revoke access at https://myaccount.google.com/permissions and retry")
	}

	return result.AccessToken, result.RefreshToken, result.ExpiresIn, nil
}

// Ignore unused import warning — context is used by the interface.
var _ context.Context
