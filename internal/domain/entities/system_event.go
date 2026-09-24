package entities

import (
	"time"

	"github.com/google/uuid"
)

// EventSeverity represents the severity of a system event.
type EventSeverity string

const (
	EventSeverityInfo    EventSeverity = "info"
	EventSeverityWarning EventSeverity = "warning"
	EventSeverityError   EventSeverity = "error"
)

// EventType categorizes system events.
type EventType string

const (
	EventTypeAccountAuthError  EventType = "account_auth_error"
	EventTypeSyncError         EventType = "sync_error"
	EventTypeNotificationError EventType = "notification_error"
	EventTypeSystemStart       EventType = "system_start"
	EventTypeSystemStop        EventType = "system_stop"
	EventTypeConfigChange      EventType = "config_change"
)

// SystemEvent records a notable event in the system's operation.
type SystemEvent struct {
	ID        uuid.UUID
	Type      EventType
	Details   string
	Severity  EventSeverity
	CreatedAt time.Time
}

// NewSystemEvent creates a new system event.
func NewSystemEvent(eventType EventType, details string, severity EventSeverity) *SystemEvent {
	return &SystemEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Details:   details,
		Severity:  severity,
		CreatedAt: time.Now(),
	}
}
