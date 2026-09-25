package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
	"github.com/mostaql-notification/internal/domain/repositories"
	"github.com/mostaql-notification/internal/infrastructure/postgres"
)

// Pages handles the new dashboard pages: health, analytics, activity, emails.
type Pages struct {
	db              *postgres.DB
	accountRepo     repositories.GmailAccountRepository
	providerRepo    repositories.ProviderRepository
	emailRepo       repositories.EmailMessageRepository
	notifRepo       repositories.NotificationRepository
	eventRepo       repositories.SystemEventRepository
	senderRuleRepo  repositories.SenderRuleRepository
	logger          *slog.Logger
}

// NewPages creates the pages handler.
func NewPages(
	db *postgres.DB,
	accountRepo repositories.GmailAccountRepository,
	providerRepo repositories.ProviderRepository,
	emailRepo repositories.EmailMessageRepository,
	notifRepo repositories.NotificationRepository,
	eventRepo repositories.SystemEventRepository,
	senderRuleRepo repositories.SenderRuleRepository,
	logger *slog.Logger,
) *Pages {
	return &Pages{
		db: db, accountRepo: accountRepo, providerRepo: providerRepo,
		emailRepo: emailRepo, notifRepo: notifRepo, eventRepo: eventRepo,
		senderRuleRepo: senderRuleRepo, logger: logger,
	}
}

// RegisterRoutes registers the new page routes. authFn wraps them with auth.
func (p *Pages) RegisterRoutes(mux *http.ServeMux, authFn func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /dashboard/health", authFn(p.handleHealth))
	mux.HandleFunc("GET /dashboard/analytics", authFn(p.handleAnalytics))
	mux.HandleFunc("GET /dashboard/activity", authFn(p.handleActivity))
	mux.HandleFunc("GET /dashboard/emails", authFn(p.handleEmails))
	mux.HandleFunc("GET /dashboard/docs", authFn(p.handleDocs))
}

// ─── Documentation ────────────────────────────────────────────

func (p *Pages) handleDocs(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "Documentation", docsPageHTML, nil, p.logger)
}

// ─── System Health ────────────────────────────────────────────

type healthData struct {
	DBHealthy       bool
	DBLatencyMs     float64
	Accounts        []entities.GmailAccount
	ProviderCount   int
	PendingNotifs   int
	DeliveredNotifs int
	TotalEmails     int
	Uptime          string
}

var startTime = time.Now()

func (p *Pages) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// DB health check with timing.
	dbStart := time.Now()
	dbHealthy := p.db.HealthCheck(ctx) == nil
	dbLatency := float64(time.Since(dbStart).Microseconds()) / 1000.0

	accounts, _ := p.accountRepo.GetAll(ctx)
	providers, _ := p.providerRepo.GetAll(ctx)
	pending, _ := p.notifRepo.GetPendingCount(ctx)
	delivered, _ := p.notifRepo.GetDeliveredCount(ctx)
	totalEmails, _ := p.emailRepo.CountTotal(ctx)

	uptime := time.Since(startTime)
	uptimeStr := fmt.Sprintf("%dd %dh %dm", int(uptime.Hours())/24, int(uptime.Hours())%24, int(uptime.Minutes())%60)

	data := healthData{
		DBHealthy:       dbHealthy,
		DBLatencyMs:     math.Round(dbLatency*100) / 100,
		Accounts:        accounts,
		ProviderCount:   len(providers),
		PendingNotifs:   pending,
		DeliveredNotifs: delivered,
		TotalEmails:     totalEmails,
		Uptime:          uptimeStr,
	}

	renderPage(w, "System Health", healthPageHTML, data, p.logger)
}

// ─── Analytics ────────────────────────────────────────────────

type analyticsData struct {
	TotalEmails     int
	DeliveredNotifs int
	ProviderCount   int
	AccountCount    int
	DailyJSON       template.JS
	ProviderJSON    template.JS
	HourJSON        template.JS
}

func (p *Pages) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	totalEmails, _ := p.emailRepo.CountTotal(ctx)
	delivered, _ := p.notifRepo.GetDeliveredCount(ctx)
	providers, _ := p.providerRepo.GetAll(ctx)
	accounts, _ := p.accountRepo.GetAll(ctx)

	dailyCounts, _ := p.emailRepo.CountByDay(ctx, 30)
	providerCounts, _ := p.emailRepo.CountByProviderGrouped(ctx)
	hourCounts, _ := p.emailRepo.CountByHour(ctx, 30)

	dailyJSON, _ := json.Marshal(dailyCounts)
	providerJSON, _ := json.Marshal(providerCounts)
	hourJSON, _ := json.Marshal(hourCounts)

	data := analyticsData{
		TotalEmails:     totalEmails,
		DeliveredNotifs: delivered,
		ProviderCount:   len(providers),
		AccountCount:    len(accounts),
		DailyJSON:       template.JS(dailyJSON),
		ProviderJSON:    template.JS(providerJSON),
		HourJSON:        template.JS(hourJSON),
	}

	renderPage(w, "Analytics", analyticsPageHTML, data, p.logger)
}

// ─── Activity Feed ────────────────────────────────────────────

type activityData struct {
	Events []entities.SystemEvent
}

func (p *Pages) handleActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	events, _ := p.eventRepo.GetRecent(ctx, 100)

	data := activityData{Events: events}
	renderPage(w, "Activity Feed", activityPageHTML, data, p.logger)
}

// ─── Email Browser ────────────────────────────────────────────

type emailBrowserData struct {
	Emails     []entities.EmailMessage
	Providers  []entities.Provider
	Accounts   []entities.GmailAccount
	Total      int
	Page       int
	TotalPages int
	Search     string
	ProviderID string
	AccountID  string
	HasPrev    bool
	HasNext    bool
}

func (p *Pages) handleEmails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 20
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	providerIDStr := r.URL.Query().Get("provider")
	accountIDStr := r.URL.Query().Get("account")

	filter := repositories.EmailListFilter{
		Search: search,
		Offset: (page - 1) * perPage,
		Limit:  perPage,
	}

	if providerIDStr != "" {
		if id, err := uuid.Parse(providerIDStr); err == nil {
			filter.ProviderID = &id
		}
	}
	if accountIDStr != "" {
		if id, err := uuid.Parse(accountIDStr); err == nil {
			filter.AccountID = &id
		}
	}

	emails, total, _ := p.emailRepo.GetAll(ctx, filter)
	providers, _ := p.providerRepo.GetAll(ctx)
	accounts, _ := p.accountRepo.GetAll(ctx)
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	data := emailBrowserData{
		Emails:     emails,
		Providers:  providers,
		Accounts:   accounts,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
		Search:     search,
		ProviderID: providerIDStr,
		AccountID:  accountIDStr,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	renderPage(w, "Email Browser", emailBrowserPageHTML, data, p.logger)
}

// ─── Helpers ──────────────────────────────────────────────────

func renderPage(w http.ResponseWriter, title, bodyHTML string, data interface{}, logger *slog.Logger) {
	fullHTML := strings.Replace(pageLayoutHTML, "{{PAGE_TITLE}}", title, 1)
	fullHTML = strings.Replace(fullHTML, "{{PAGE_BODY}}", bodyHTML, 1)

	funcMap := template.FuncMap{
		"statusEmoji": func(status entities.AccountStatus) string {
			switch status {
			case entities.AccountStatusActive:
				return "🟢"
			case entities.AccountStatusAuthError:
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
		"fmtTime": func(t time.Time) string {
			return t.Format("02 Jan 15:04")
		},
		"severityColor": func(s entities.EventSeverity) string {
			switch s {
			case entities.EventSeverityError:
				return "#f87171"
			case entities.EventSeverityWarning:
				return "#fbbf24"
			default:
				return "#34d399"
			}
		},
		"severityIcon": func(s entities.EventSeverity) string {
			switch s {
			case entities.EventSeverityError:
				return "🔴"
			case entities.EventSeverityWarning:
				return "🟡"
			default:
				return "🟢"
			}
		},
		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"gmailURL": func(msgID string) string {
			return "https://mail.google.com/mail/#inbox/" + msgID
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "…"
		},
	}

	tmpl, err := template.New("page").Funcs(funcMap).Parse(fullHTML)
	if err != nil {
		logger.Error("template parse error", "error", err)
		http.Error(w, "Template error", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		logger.Error("template render error", "error", err)
	}
}
