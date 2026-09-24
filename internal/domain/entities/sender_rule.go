package entities

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// MatchType defines how a sender rule matches against email addresses.
type MatchType string

const (
	// MatchTypeExact matches the full sender email address.
	MatchTypeExact MatchType = "exact"
	// MatchTypeDomain matches the domain part of the sender email.
	MatchTypeDomain MatchType = "domain"
)

// SenderRule defines a rule for matching email senders to a provider.
type SenderRule struct {
	ID           uuid.UUID
	ProviderID   uuid.UUID
	SenderEmail  string // Used when MatchType is "exact"
	SenderDomain string // Used when MatchType is "domain"
	MatchType    MatchType
	IsActive     bool
	CreatedAt    time.Time
}

// Matches returns true if the given sender email address matches this rule.
func (r *SenderRule) Matches(senderEmail string) bool {
	if !r.IsActive {
		return false
	}

	senderEmail = strings.ToLower(strings.TrimSpace(senderEmail))

	switch r.MatchType {
	case MatchTypeExact:
		return senderEmail == strings.ToLower(r.SenderEmail)
	case MatchTypeDomain:
		domain := strings.ToLower(r.SenderDomain)
		parts := strings.SplitN(senderEmail, "@", 2)
		if len(parts) != 2 {
			return false
		}
		return parts[1] == domain
	default:
		return false
	}
}

// NewExactSenderRule creates a sender rule that matches an exact email address.
func NewExactSenderRule(providerID uuid.UUID, senderEmail string) *SenderRule {
	return &SenderRule{
		ID:          uuid.New(),
		ProviderID:  providerID,
		SenderEmail: strings.ToLower(strings.TrimSpace(senderEmail)),
		MatchType:   MatchTypeExact,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
}

// NewDomainSenderRule creates a sender rule that matches all emails from a domain.
func NewDomainSenderRule(providerID uuid.UUID, domain string) *SenderRule {
	return &SenderRule{
		ID:           uuid.New(),
		ProviderID:   providerID,
		SenderDomain: strings.ToLower(strings.TrimSpace(domain)),
		MatchType:    MatchTypeDomain,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}
}
