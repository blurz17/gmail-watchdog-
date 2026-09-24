-- Rollback initial schema
DROP TABLE IF EXISTS system_events CASCADE;
DROP TABLE IF EXISTS notification_deliveries CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS email_messages CASCADE;
DROP TABLE IF EXISTS sender_rules CASCADE;
DROP TABLE IF EXISTS providers CASCADE;
DROP TABLE IF EXISTS gmail_accounts CASCADE;
