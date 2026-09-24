package services

import (
	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

// MatchResult represents a successful match between a sender and a provider.
type MatchResult struct {
	ProviderID uuid.UUID
	RuleID     uuid.UUID
}

// SenderMatcher matches incoming email sender addresses against configured sender rules.
// This is a pure domain service with no external dependencies.
type SenderMatcher struct{}

// NewSenderMatcher creates a new SenderMatcher.
func NewSenderMatcher() *SenderMatcher {
	return &SenderMatcher{}
}

// Match finds all providers whose sender rules match the given email address.
// Returns a list of matching provider IDs. A single email can match multiple providers
// if the same sender address is configured for multiple providers.
func (m *SenderMatcher) Match(senderEmail string, rules []entities.SenderRule) []MatchResult {
	var results []MatchResult
	seen := make(map[uuid.UUID]bool)

	for _, rule := range rules {
		if rule.Matches(senderEmail) {
			// Deduplicate: one result per provider even if multiple rules match.
			if !seen[rule.ProviderID] {
				seen[rule.ProviderID] = true
				results = append(results, MatchResult{
					ProviderID: rule.ProviderID,
					RuleID:     rule.ID,
				})
			}
		}
	}

	return results
}

// MatchAll finds all provider matches across a list of sender emails.
// This is useful when processing emails with multiple From headers or aliases.
func (m *SenderMatcher) MatchAll(senderEmails []string, rules []entities.SenderRule) []MatchResult {
	var allResults []MatchResult
	seen := make(map[uuid.UUID]bool)

	for _, email := range senderEmails {
		results := m.Match(email, rules)
		for _, r := range results {
			if !seen[r.ProviderID] {
				seen[r.ProviderID] = true
				allResults = append(allResults, r)
			}
		}
	}

	return allResults
}
