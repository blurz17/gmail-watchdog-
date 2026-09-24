package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mostaql-notification/internal/domain/entities"
)

func TestSenderMatcher_Match_ExactMatch(t *testing.T) {
	matcher := NewSenderMatcher()
	providerID := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			SenderEmail: "support@heroku.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
	}

	results := matcher.Match("support@heroku.com", rules)
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
	if results[0].ProviderID != providerID {
		t.Errorf("expected provider %s, got %s", providerID, results[0].ProviderID)
	}
}

func TestSenderMatcher_Match_CaseInsensitive(t *testing.T) {
	matcher := NewSenderMatcher()
	providerID := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			SenderEmail: "Support@Heroku.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
	}

	results := matcher.Match("support@heroku.com", rules)
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSenderMatcher_Match_DomainMatch(t *testing.T) {
	matcher := NewSenderMatcher()
	providerID := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:           uuid.New(),
			ProviderID:   providerID,
			SenderDomain: "heroku.com",
			MatchType:    entities.MatchTypeDomain,
			IsActive:     true,
		},
	}

	results := matcher.Match("anything@heroku.com", rules)
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSenderMatcher_Match_NoMatch(t *testing.T) {
	matcher := NewSenderMatcher()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  uuid.New(),
			SenderEmail: "support@heroku.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
	}

	results := matcher.Match("other@example.com", rules)
	if len(results) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(results))
	}
}

func TestSenderMatcher_Match_InactiveRuleSkipped(t *testing.T) {
	matcher := NewSenderMatcher()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  uuid.New(),
			SenderEmail: "support@heroku.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    false, // Inactive rule
		},
	}

	results := matcher.Match("support@heroku.com", rules)
	if len(results) != 0 {
		t.Fatalf("expected 0 matches for inactive rule, got %d", len(results))
	}
}

func TestSenderMatcher_Match_MultipleProviders(t *testing.T) {
	matcher := NewSenderMatcher()
	provider1 := uuid.New()
	provider2 := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  provider1,
			SenderEmail: "notifications@github.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
		{
			ID:           uuid.New(),
			ProviderID:   provider2,
			SenderDomain: "github.com",
			MatchType:    entities.MatchTypeDomain,
			IsActive:     true,
		},
	}

	// Same email matches both an exact rule and a domain rule for different providers.
	results := matcher.Match("notifications@github.com", rules)
	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}
}

func TestSenderMatcher_Match_DeduplicatesSameProvider(t *testing.T) {
	matcher := NewSenderMatcher()
	providerID := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			SenderEmail: "info@mostaql.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
		{
			ID:           uuid.New(),
			ProviderID:   providerID,
			SenderDomain: "mostaql.com",
			MatchType:    entities.MatchTypeDomain,
			IsActive:     true,
		},
	}

	// Both rules match, but they belong to the same provider. Should deduplicate.
	results := matcher.Match("info@mostaql.com", rules)
	if len(results) != 1 {
		t.Fatalf("expected 1 match (deduplicated), got %d", len(results))
	}
}

func TestSenderMatcher_Match_WhitespaceHandling(t *testing.T) {
	matcher := NewSenderMatcher()
	providerID := uuid.New()

	rules := []entities.SenderRule{
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			SenderEmail: "support@heroku.com",
			MatchType:   entities.MatchTypeExact,
			IsActive:    true,
		},
	}

	results := matcher.Match("  support@heroku.com  ", rules)
	if len(results) != 1 {
		t.Fatalf("expected 1 match with whitespace, got %d", len(results))
	}
}
