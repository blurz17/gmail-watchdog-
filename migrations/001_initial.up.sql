-- Initial database schema for Gmail Notification Service
-- Creates all core tables with proper constraints and indexes.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Gmail accounts table
CREATE TABLE gmail_accounts (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email               VARCHAR(255) NOT NULL UNIQUE,
    display_name        VARCHAR(255) NOT NULL DEFAULT '',
    oauth_client_id     TEXT NOT NULL,
    oauth_client_secret TEXT NOT NULL,
    refresh_token       TEXT NOT NULL,
    access_token        TEXT NOT NULL DEFAULT '',
    access_token_expiry TIMESTAMP WITH TIME ZONE,
    status              VARCHAR(50) NOT NULL DEFAULT 'active',
    last_sync_at        TIMESTAMP WITH TIME ZONE,
    last_history_id     BIGINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_gmail_accounts_status CHECK (status IN ('active', 'auth_error', 'disabled'))
);

-- Providers table
CREATE TABLE providers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL UNIQUE,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    detection_mode  VARCHAR(50) NOT NULL DEFAULT 'unread',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_providers_detection_mode CHECK (detection_mode IN ('unread', 'new'))
);

-- Sender rules table
CREATE TABLE sender_rules (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id     UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    sender_email    VARCHAR(255),
    sender_domain   VARCHAR(255),
    match_type      VARCHAR(50) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sender_rules_match_type CHECK (match_type IN ('exact', 'domain')),
    CONSTRAINT chk_sender_rules_value CHECK (
        (match_type = 'exact' AND sender_email IS NOT NULL AND sender_email != '') OR
        (match_type = 'domain' AND sender_domain IS NOT NULL AND sender_domain != '')
    )
);

-- Email messages table
CREATE TABLE email_messages (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id          UUID NOT NULL REFERENCES gmail_accounts(id) ON DELETE CASCADE,
    gmail_message_id    VARCHAR(255) NOT NULL,
    thread_id           VARCHAR(255) NOT NULL DEFAULT '',
    provider_id         UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    sender_from         VARCHAR(500) NOT NULL,
    subject             TEXT NOT NULL DEFAULT '',
    snippet             TEXT NOT NULL DEFAULT '',
    received_at         TIMESTAMP WITH TIME ZONE NOT NULL,
    is_unread           BOOLEAN NOT NULL DEFAULT TRUE,
    processing_status   VARCHAR(50) NOT NULL DEFAULT 'detected',
    detected_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_email_messages_status CHECK (processing_status IN ('detected', 'notified', 'skipped'))
);

-- Prevent duplicate message processing: the core duplicate-prevention mechanism.
-- A message from a specific account matched to a specific provider is recorded exactly once.
CREATE UNIQUE INDEX idx_email_messages_account_gmail_provider
    ON email_messages(account_id, gmail_message_id, provider_id);

-- Notifications outbox table
CREATE TABLE notifications (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email_message_id    UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    provider_id         UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    account_id          UUID NOT NULL REFERENCES gmail_accounts(id) ON DELETE CASCADE,
    formatted_content   TEXT NOT NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count         INTEGER NOT NULL DEFAULT 0,
    next_retry_at       TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_notifications_status CHECK (status IN ('pending', 'sending', 'delivered', 'failed'))
);

-- Notification deliveries table
CREATE TABLE notification_deliveries (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id     UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel             VARCHAR(50) NOT NULL,
    status              VARCHAR(50) NOT NULL,
    external_id         VARCHAR(255) NOT NULL DEFAULT '',
    error_message       TEXT NOT NULL DEFAULT '',
    delivered_at        TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_deliveries_channel CHECK (channel IN ('telegram', 'whatsapp')),
    CONSTRAINT chk_deliveries_status CHECK (status IN ('success', 'failed'))
);

-- System events table
CREATE TABLE system_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type  VARCHAR(100) NOT NULL,
    details     TEXT NOT NULL DEFAULT '',
    severity    VARCHAR(50) NOT NULL DEFAULT 'info',
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_events_severity CHECK (severity IN ('info', 'warning', 'error'))
);

-- Performance indexes

-- Fast lookup for pending notifications (outbox polling)
CREATE INDEX idx_notifications_pending
    ON notifications(status, next_retry_at)
    WHERE status IN ('pending', 'failed');

-- Fast sender matching during sync
CREATE INDEX idx_sender_rules_email
    ON sender_rules(sender_email)
    WHERE is_active = TRUE AND sender_email IS NOT NULL;

CREATE INDEX idx_sender_rules_domain
    ON sender_rules(sender_domain)
    WHERE is_active = TRUE AND sender_domain IS NOT NULL;

-- Count/latest queries by provider
CREATE INDEX idx_email_messages_provider_received
    ON email_messages(provider_id, received_at DESC);

-- Unread count queries
CREATE INDEX idx_email_messages_unread
    ON email_messages(account_id, provider_id)
    WHERE is_unread = TRUE;

-- System events by time (for recent events query)
CREATE INDEX idx_system_events_created
    ON system_events(created_at DESC);

-- Active accounts for sync
CREATE INDEX idx_gmail_accounts_active
    ON gmail_accounts(status)
    WHERE status = 'active';
