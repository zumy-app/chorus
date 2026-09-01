package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Database connected successfully")
	return db, nil
}

func Migrate(db *sql.DB) error {
	migrations := []string{
		// Phase 1: Core tables
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			display_name VARCHAR(100) NOT NULL,
			native_language VARCHAR(10) NOT NULL,
			target_languages TEXT[] DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE users ALTER COLUMN username TYPE VARCHAR(255)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,

		`CREATE TABLE IF NOT EXISTS chats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type VARCHAR(20) NOT NULL CHECK (type IN ('direct', 'group')),
			name VARCHAR(100),
			created_by UUID NOT NULL REFERENCES users(id),
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_created_by ON chats(created_by)`,

		`CREATE TABLE IF NOT EXISTS chat_participants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(20) NOT NULL CHECK (role IN ('member', 'admin')),
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_read_message_id UUID,
			UNIQUE(chat_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_participants_chat_id ON chat_participants(chat_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_participants_user_id ON chat_participants(user_id)`,

		`CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			sender_id UUID NOT NULL REFERENCES users(id),
			text TEXT NOT NULL,
			original_language VARCHAR(10),
			translations JSONB DEFAULT '{}',
			delivery_status VARCHAR(20) NOT NULL DEFAULT 'sent' CHECK (delivery_status IN ('sent', 'delivered', 'failed')),
			reply_to_id UUID REFERENCES messages(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_text_search ON messages USING gin(to_tsvector('english', text))`,

		// Per-recipient read / delivery ticks (task 6.1). One row per
		// (message, recipient); delivered/read timestamps drive the sent /
		// delivered / read tick shown to the sender.
		`CREATE TABLE IF NOT EXISTS message_receipts (
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			received_at TIMESTAMP,
			read_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (message_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_message_receipts_chat ON message_receipts(chat_id, message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_receipts_user ON message_receipts(user_id)`,

		// Message actions (task 6.2). Soft delete keeps the row so replies and
		// forwards to a deleted message keep working and history is recoverable;
		// deleted_at IS NULL rows are excluded from chat history and search.
		// Forwarded messages are copies authored by the forwarder that keep a
		// trail to the original author / message / chat for the "Forwarded from"
		// label. All columns are additive (IF NOT EXISTS) so existing rows stay
		// NULL, which reads as "not forwarded, not deleted".
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS forwarded_from_message_id UUID REFERENCES messages(id) ON DELETE SET NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS forwarded_from_chat_id UUID REFERENCES chats(id) ON DELETE SET NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS forwarded_from_sender_id UUID REFERENCES users(id) ON DELETE SET NULL`,

		// Chat-scoped pin (task 6.2): any participant may pin up to N messages;
		// the list is ordered newest-first. One row per (chat, message).
		`CREATE TABLE IF NOT EXISTS pinned_messages (
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			pinned_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (chat_id, message_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pinned_messages_chat ON pinned_messages(chat_id, created_at DESC)`,

		// Archive & mute (task 6.4): per-user, per-chat conversation preferences.
		// Archive hides a chat from the user's main list (archived_at set) without
		// leaving the chat; mute silences notifications (is_muted) — optionally
		// until a timestamp (muted_until) so a mute can be timed or indefinite
		// (NULL + is_muted = indefinite). Preferences are strictly user-scoped:
		// one user archiving a chat doesn't affect the other participants.
		// A LEFT JOIN from chat lists means a missing row reads as
		// "not archived, not muted", so the table needs no seeding.
		`CREATE TABLE IF NOT EXISTS chat_preferences (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			archived_at TIMESTAMP,
			is_muted BOOLEAN NOT NULL DEFAULT false,
			muted_until TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, chat_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_preferences_user ON chat_preferences(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_preferences_chat ON chat_preferences(chat_id)`,

		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(500) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token)`,

		// Password reset tokens (forgot password flow).
		`CREATE TABLE IF NOT EXISTS password_resets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(64) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_password_resets_token_hash ON password_resets(token_hash)`,

		// Invite-only launch waitlist.
		`CREATE TABLE IF NOT EXISTS waitlist_entries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			spoken_languages TEXT[] NOT NULL DEFAULT '{}',
			target_languages TEXT[] NOT NULL,
			reasons TEXT[] NOT NULL,
			comments TEXT NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved')),
			queue_position BIGSERIAL UNIQUE NOT NULL,
			approved_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE waitlist_entries ADD COLUMN IF NOT EXISTS comments TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE waitlist_entries ADD COLUMN IF NOT EXISTS spoken_languages TEXT[] NOT NULL DEFAULT '{}'`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name = 'waitlist_entries' AND column_name = 'spoken_language') THEN
				EXECUTE 'UPDATE waitlist_entries SET spoken_languages = ARRAY[spoken_language]
					WHERE spoken_language IS NOT NULL AND spoken_language <> ''''
					AND (array_length(spoken_languages, 1) IS NULL OR array_length(spoken_languages, 1) = 0)';
			END IF;
		END $$;`,
		`ALTER TABLE waitlist_entries DROP COLUMN IF EXISTS spoken_language`,
		`CREATE INDEX IF NOT EXISTS idx_waitlist_entries_status_position ON waitlist_entries(status, queue_position)`,
		`ALTER TABLE waitlist_entries DROP CONSTRAINT IF EXISTS waitlist_entries_status_check`,
		`ALTER TABLE waitlist_entries ADD CONSTRAINT waitlist_entries_status_check CHECK (status IN ('pending', 'approved', 'declined'))`,

		// Durable email outbox: every notification is persisted here before it
		// is handed to SMTP so none are lost, and failed sends are retried by
		// the NotificationService background worker.
		`CREATE TABLE IF NOT EXISTS email_outbox (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			recipient VARCHAR(255) NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			sent_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_outbox_status_next ON email_outbox(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS idx_email_outbox_recipient ON email_outbox(recipient)`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			waitlist_entry_id UUID NOT NULL REFERENCES waitlist_entries(id) ON DELETE CASCADE,
			email VARCHAR(255) NOT NULL,
			token_hash VARCHAR(64) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			redeemed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations(token_hash)`,

		// Contacts & Invites epic (REQ 2.4 / FR-22-23): the table is the single
		// source for the waitlist invitation flow AND self-service invites a
		// registered user sends to an off-platform contact. waitlist_entry_id is
		// nullable to support the latter (invites not born from the waitlist).
		`ALTER TABLE invitations ALTER COLUMN waitlist_entry_id DROP NOT NULL`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS inviter_user_id UUID REFERENCES users(id) ON DELETE CASCADE`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'email' CHECK (channel IN ('email', 'sms', 'whatsapp'))`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS recipient VARCHAR(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS name VARCHAR(100) NOT NULL DEFAULT ''`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'sent' CHECK (status IN ('pending', 'sent', 'redeemed', 'expired'))`,
		`ALTER TABLE invitations ADD COLUMN IF NOT EXISTS sent_at TIMESTAMP`,
		`CREATE INDEX IF NOT EXISTS idx_invitations_inviter ON invitations(inviter_user_id)`,

		// Phase 2: Multi-device support - Clients table
		`CREATE TABLE IF NOT EXISTS clients (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_type VARCHAR(20) NOT NULL CHECK (device_type IN ('mobile', 'web', 'desktop')),
			device_info JSONB DEFAULT '{}',
			connection_status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (connection_status IN ('online', 'offline')),
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clients_user_id ON clients(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_clients_user_status ON clients(user_id, connection_status)`,

		// Phase 2: Offline message delivery - Inbox table
		`CREATE TABLE IF NOT EXISTS inbox (
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			delivery_attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ttl TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
			PRIMARY KEY (client_id, message_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_client_created ON inbox(client_id, created_at)`,

		// Phase 2: User settings extensions
		`CREATE TABLE IF NOT EXISTS user_settings (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			grammar_enabled BOOLEAN DEFAULT true,
			vocabulary_enabled BOOLEAN DEFAULT true,
			difficulty_level VARCHAR(20) DEFAULT 'intermediate' CHECK (difficulty_level IN ('beginner', 'intermediate', 'advanced')),
			transcript_recording BOOLEAN DEFAULT true,
			message_retention_days INTEGER DEFAULT 365,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// FR-25 Feature toggles: per-account switches for auto-translation,
		// auto-grammar, and learning highlights. Default to enabled; setting any
		// of them to false prevents the corresponding server-side job from being
		// enqueued (translation jobs are never enqueued when off). Additive and
		// backfilled (NOT NULL DEFAULT true) so existing rows stay valid.
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS translation_enabled BOOLEAN NOT NULL DEFAULT true`,
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS grammar_auto BOOLEAN NOT NULL DEFAULT true`,
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS highlights_enabled BOOLEAN NOT NULL DEFAULT true`,
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS last_seen_visibility VARCHAR(20) NOT NULL DEFAULT 'everyone'`,
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS profile_photo_visibility VARCHAR(20) NOT NULL DEFAULT 'everyone'`,
		`ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS contacts_visibility VARCHAR(20) NOT NULL DEFAULT 'everyone'`,
		`ALTER TABLE user_settings DROP CONSTRAINT IF EXISTS user_settings_privacy_check`,
		`ALTER TABLE user_settings ADD CONSTRAINT user_settings_privacy_check CHECK (last_seen_visibility IN ('everyone','contacts','nobody') AND profile_photo_visibility IN ('everyone','contacts','nobody') AND contacts_visibility IN ('everyone','contacts','nobody'))`,

		// Phase 2: Media attachments
		`CREATE TABLE IF NOT EXISTS media_attachments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL CHECK (type IN ('image', 'video', 'audio', 'document')),
			file_name VARCHAR(255) NOT NULL,
			file_size BIGINT NOT NULL,
			mime_type VARCHAR(100) NOT NULL,
			url TEXT NOT NULL,
			thumbnail_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_media_message_id ON media_attachments(message_id)`,
		// Task 6.5/6.7: widen the media type constraint to allow 'link' (shared
		// URLs) and 'location' (shared map pins) alongside the file-backed types
		// so the gallery's Links tab and the location surface are queryable.
		// DROP IF EXISTS + ADD keeps the migration idempotent on every boot.
		`ALTER TABLE media_attachments DROP CONSTRAINT IF EXISTS media_attachments_type_check`,
		`ALTER TABLE media_attachments ADD CONSTRAINT media_attachments_type_check CHECK (type IN ('image', 'video', 'audio', 'document', 'link', 'location'))`,
		// Task 6.7 (location sharing): latitude/longitude + an optional label for
		// a shared map pin. These stay NULL for every other media type; the URL
		// column still holds a client-facing map link and the message fan-out +
		// history read paths expose these fields so the client renders the pin.
		`ALTER TABLE media_attachments ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION`,
		`ALTER TABLE media_attachments ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION`,
		`ALTER TABLE media_attachments ADD COLUMN IF NOT EXISTS location_name VARCHAR(255)`,
		// Gallery ordering is by created_at (newest first) within a chat, so the
		// paginated read paths skip a sort of the full attachment set.
		`CREATE INDEX IF NOT EXISTS idx_media_created_at ON media_attachments(created_at DESC)`,

		// Phase 3: Vocabulary management
		`CREATE TABLE IF NOT EXISTS vocabulary (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			term VARCHAR(255) NOT NULL,
			language VARCHAR(10) NOT NULL,
			translation VARCHAR(500) NOT NULL,
			definition TEXT,
			context_message_id UUID REFERENCES messages(id),
			context_sentence TEXT,
			context_chat_id UUID REFERENCES chats(id),
			review_count INTEGER DEFAULT 0,
			correct_count INTEGER DEFAULT 0,
			next_review TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			interval_days INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, term, language)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_user_id ON vocabulary(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_user_due ON vocabulary(user_id, next_review)`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_user_language ON vocabulary(user_id, language)`,

		// Phase 3: Call sessions
		`CREATE TABLE IF NOT EXISTS call_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			type VARCHAR(10) NOT NULL CHECK (type IN ('audio', 'video')),
			status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended')),
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ended_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_chat_id ON call_sessions(chat_id)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_status_active ON call_sessions(status, started_at) WHERE status = 'active'`,
		`ALTER TABLE call_sessions ADD COLUMN IF NOT EXISTS participants JSONB NOT NULL DEFAULT '[]'`,
		`ALTER TABLE call_sessions ADD COLUMN IF NOT EXISTS initiator_id UUID REFERENCES users(id) ON DELETE SET NULL`,

		// Phase 3: Call participants
		`CREATE TABLE IF NOT EXISTS call_participants (
			call_id UUID NOT NULL REFERENCES call_sessions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			left_at TIMESTAMP,
			PRIMARY KEY (call_id, user_id)
		)`,

		// Phase 3: Call transcripts
		`CREATE TABLE IF NOT EXISTS call_transcripts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			call_id UUID NOT NULL REFERENCES call_sessions(id) ON DELETE CASCADE,
			segments JSONB DEFAULT '[]',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_transcripts_call_id ON call_transcripts(call_id)`,

		// Phase 2: Presence tracking (using Redis primarily, but backup in DB)
		`CREATE TABLE IF NOT EXISTS presence_log (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL,
			device_type VARCHAR(20),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_presence_user_time ON presence_log(user_id, timestamp DESC)`,

		// Phase 2: Rate limiting tracking
		`CREATE TABLE IF NOT EXISTS rate_limits (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			action_type VARCHAR(50) NOT NULL,
			count INTEGER DEFAULT 1,
			window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, action_type)
		)`,

		// Roles & account state: role gates admin/moderation endpoints,
		// suspended_at is a reversible soft ban, deleted_at is a permanent
		// soft delete (account is blocked but history/chats remain intact).
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'member'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,

		// Onboarding (REQ 2.1): structured first/last name so the profile can
		// show a composed displayName and deterministic initials/avatar (2.2).
		// Nullable so accounts created before this migration stay valid; the
		// composed display_name is the derived, still-editable display name.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(100)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(100)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL`,

		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT false`,
		`CREATE TABLE IF NOT EXISTS phone_otps (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			phone VARCHAR(20) NOT NULL,
			code_hash VARCHAR(64) NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_phone_otps_user_phone ON phone_otps(user_id, phone, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_phone_otps_expires ON phone_otps(expires_at)`,

		// Billing plan & cutover grace. plan_grace_until is the explicit
		// grace-upgrade window for accounts created before a paid cutover: the
		// entitlement service treats them as premium until the deadline, then
		// they drop to their stored plan (free). Both are additive and default
		// to free, so every existing user is migrated to the free plan.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS plan VARCHAR(20) NOT NULL DEFAULT 'free'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS plan_grace_until TIMESTAMP`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_plan_check`,
		`ALTER TABLE users ADD CONSTRAINT users_plan_check CHECK (plan IN ('free', 'premium'))`,

		// Durable translation jobs: one row per (message, target language).
		// Rows are the source of truth; Redis pub/sub is only the near-real-time
		// trigger. The queue worker + sweeper + startup recovery are implemented
		// in TranslationQueueService.
		`CREATE TABLE IF NOT EXISTS translation_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			source_lang VARCHAR(10),
			target_lang VARCHAR(10) NOT NULL,
			priority INT NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
			result TEXT,
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_at TIMESTAMP,
			completed_at TIMESTAMP,
			UNIQUE(message_id, target_lang)
		)`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS idx_translation_jobs_status ON translation_jobs(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS idx_translation_jobs_message ON translation_jobs(message_id)`,

		// Monetization (Phase 1.5): subscription state lives on the user row so
		// entitlement resolution stays a single-row read. subscription_id is the
		// provider-side subscription id; subscription_plan_id records which plan
		// it maps to (PayPal Billing plan id) for analytics and checkout reuse.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS premium_since TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_id VARCHAR(255)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_provider VARCHAR(20) NOT NULL DEFAULT 'paypal'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_plan_id VARCHAR(255)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS next_billing_date TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_payment_at TIMESTAMP`,

		// REQ 2.2 / FR-20: reserved upload path for a custom avatar image. The
		// Phase 1 initials avatar is derived from the user's name plus a
		// deterministic color (computed server-side), so this stays NULL until
		// attachment infrastructure ships.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_subscription_id ON users(subscription_id) WHERE subscription_id IS NOT NULL`,
		// grace_notified_at marks users whose grace period has already triggered
		// the "premium ended" email, so the expiry sweeper is idempotent.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS grace_notified_at TIMESTAMP`,

		// Durable webhook/event log. provider_event_id is the idempotency key so
		// re-delivered webhooks never double-process. user_id is resolved when
		// the event is handled (nullable for events that arrive before a user is
		// matched, e.g. payer-email fallback retries).
		`CREATE TABLE IF NOT EXISTS subscription_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			event_type VARCHAR(100) NOT NULL,
			provider VARCHAR(20) NOT NULL DEFAULT 'paypal',
			provider_event_id VARCHAR(255) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'received' CHECK (status IN ('received', 'processed', 'failed', 'ignored')),
			payload JSONB NOT NULL DEFAULT '{}',
			error TEXT,
			received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			handled_at TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_events_provider_event ON subscription_events(provider, provider_event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_events_user ON subscription_events(user_id, received_at DESC)`,

		// Audit trail for every plan change (admin grants/revokes AND provider
		// lifecycle events). actor_id is the admin for manual changes, null for
		// system/webhook-driven changes.
		`CREATE TABLE IF NOT EXISTS plan_changes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
			from_plan VARCHAR(20) NOT NULL,
			to_plan VARCHAR(20) NOT NULL,
			grace_until TIMESTAMP,
			source VARCHAR(50) NOT NULL DEFAULT 'webhook',
			reason TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_plan_changes_user ON plan_changes(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_plan_changes_created ON plan_changes(created_at)`,

		// Report & Block (safety rails, REQ §8.2). blocked_users holds directed
		// blocks; enforcement treats either direction as blocking communication.
		`CREATE TABLE IF NOT EXISTS blocked_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			reason TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_users_pair ON blocked_users(blocker_id, blocked_id)`,
		`CREATE INDEX IF NOT EXISTS idx_blocked_users_blocked ON blocked_users(blocked_id)`,

		`CREATE TABLE IF NOT EXISTS reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (type IN ('user', 'message')),
			reported_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
			chat_id UUID REFERENCES chats(id) ON DELETE SET NULL,
			reason TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'dismissed')),
			resolver_id UUID REFERENCES users(id) ON DELETE SET NULL,
			resolution_note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			resolved_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports(reported_user_id, created_at DESC)`,

		// Durable AI grammar analysis jobs (one row per analysis request). Mirrors
		// translation_jobs: rows are the source of truth, Redis pub/sub is the
		// near-real-time trigger, and the worker + sweeper + startup recovery live
		// in GrammarQueueService. Results are pushed to the requesting user only.
		`CREATE TABLE IF NOT EXISTS grammar_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
			text TEXT NOT NULL,
			language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
			result JSONB,
			provider_used VARCHAR(50),
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_at TIMESTAMP,
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_jobs_user_created ON grammar_jobs(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_jobs_status ON grammar_jobs(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_jobs_message ON grammar_jobs(message_id)`,

		// -------------------------------------------------------------------
		// FR-30 Quality pipeline — lineage + cross-model evaluation
		// -------------------------------------------------------------------
		// Lineage metadata on the durable translation/grammar jobs so every AI
		// output can be attributed to the provider, model prompt version, and
		// measured latency/tokens that produced it. This is what makes KPIs
		// (accuracy, p95 latency, cost/1k tokens, cache hit rate) derivable.
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS provider VARCHAR(100)`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS model VARCHAR(200)`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS prompt_version VARCHAR(50) NOT NULL DEFAULT 'v2'`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS latency_ms INTEGER`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS tokens INTEGER`,
		`ALTER TABLE translation_jobs ADD COLUMN IF NOT EXISTS cache_hit BOOLEAN NOT NULL DEFAULT false`,
		`CREATE INDEX IF NOT EXISTS idx_translation_jobs_provider ON translation_jobs(provider, created_at DESC)`,

		`ALTER TABLE grammar_jobs ADD COLUMN IF NOT EXISTS provider_used_lineage VARCHAR(100)`,
		`ALTER TABLE grammar_jobs ADD COLUMN IF NOT EXISTS model VARCHAR(200)`,
		`ALTER TABLE grammar_jobs ADD COLUMN IF NOT EXISTS prompt_version VARCHAR(50) NOT NULL DEFAULT 'v2'`,
		`ALTER TABLE grammar_jobs ADD COLUMN IF NOT EXISTS latency_ms INTEGER`,
		`ALTER TABLE grammar_jobs ADD COLUMN IF NOT EXISTS tokens INTEGER`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_jobs_provider ON grammar_jobs(provider_used_lineage, created_at DESC)`,

		// Cross-model evaluation: one row per scored translation. The evaluator
		// is a *different* model than the producer; rows start 'pending', are
		// picked up by the QualityEvaluatorService worker, and end 'done'/'failed'.
		`CREATE TABLE IF NOT EXISTS translation_evals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			translation_job_id UUID REFERENCES translation_jobs(id) ON DELETE CASCADE,
			message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			source_lang VARCHAR(10),
			target_lang VARCHAR(10) NOT NULL,
			source_text TEXT NOT NULL,
			translated_text TEXT NOT NULL,
			producer_provider VARCHAR(100),
			evaluator_provider VARCHAR(100),
			accuracy_score NUMERIC(5,2),
			fluency_score NUMERIC(5,2),
			cefr_level VARCHAR(2),
			critique TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_at TIMESTAMP,
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_translation_evals_status ON translation_evals(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_translation_evals_job ON translation_evals(translation_job_id)`,

		// Cross-model evaluation for grammar analyses. criterion_scores holds
		// per-pattern scores (grammar accuracy, structure, helpfulness) as JSON.
		`CREATE TABLE IF NOT EXISTS grammar_evals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			grammar_job_id UUID REFERENCES grammar_jobs(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL,
			producer_provider VARCHAR(100),
			evaluator_provider VARCHAR(100),
			accuracy_score NUMERIC(5,2),
			cefr_level VARCHAR(2),
			criterion_scores JSONB NOT NULL DEFAULT '{}',
			critique TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_at TIMESTAMP,
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_evals_status ON grammar_evals(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_evals_job ON grammar_evals(grammar_job_id)`,

		// -------------------------------------------------------------------
		// Learning engine: pair-aware structured courses, vocabulary mining,
		// staged SRS, lessons, scenarios, placement, streaks, and score data.
		// Translation/grammar are broadly model-powered; these tables represent
		// the curated learning layer that is enabled per native->target pair.
		// -------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS curriculum_courses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			target_language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL DEFAULT 'en',
			title VARCHAR(255) NOT NULL,
			version VARCHAR(50) NOT NULL DEFAULT 'v1',
			is_active BOOLEAN NOT NULL DEFAULT true,
			support_tier VARCHAR(30) NOT NULL DEFAULT 'full_course' CHECK (support_tier IN ('full_course', 'beta_ai_assisted')),
			metadata JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(target_language, native_language, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_curriculum_courses_active ON curriculum_courses(target_language, native_language, is_active)`,

		`CREATE TABLE IF NOT EXISTS curriculum_units (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
			cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			ordinal INTEGER NOT NULL,
			slug VARCHAR(120) NOT NULL,
			title VARCHAR(255) NOT NULL,
			can_do_statement TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			estimated_minutes INTEGER NOT NULL DEFAULT 30,
			checkpoint_required BOOLEAN NOT NULL DEFAULT true,
			metadata JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(course_id, ordinal),
			UNIQUE(course_id, slug)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_curriculum_units_course_level ON curriculum_units(course_id, cefr_level, ordinal)`,

		`CREATE TABLE IF NOT EXISTS learning_pair_capabilities (
			native_language VARCHAR(10) NOT NULL,
			target_language VARCHAR(10) NOT NULL,
			support_tier VARCHAR(30) NOT NULL DEFAULT 'vocab_only' CHECK (support_tier IN ('full_course', 'beta_ai_assisted', 'vocab_only', 'disabled')),
			active_course_id UUID REFERENCES curriculum_courses(id) ON DELETE SET NULL,
			placement_enabled BOOLEAN NOT NULL DEFAULT false,
			roadmap_enabled BOOLEAN NOT NULL DEFAULT false,
			scenarios_enabled BOOLEAN NOT NULL DEFAULT false,
			srs_enabled BOOLEAN NOT NULL DEFAULT true,
			mining_enabled BOOLEAN NOT NULL DEFAULT true,
			grammar_feedback_enabled BOOLEAN NOT NULL DEFAULT true,
			quality_notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (native_language, target_language)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_pair_capabilities_tier ON learning_pair_capabilities(support_tier)`,

		`CREATE TABLE IF NOT EXISTS user_language_profiles (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			native_language VARCHAR(10) NOT NULL DEFAULT 'en',
			target_language VARCHAR(10) NOT NULL,
			current_cefr_level VARCHAR(2) NOT NULL DEFAULT 'A1' CHECK (current_cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			readiness_score INTEGER NOT NULL DEFAULT 0 CHECK (readiness_score >= 0 AND readiness_score <= 1000),
			active_course_id UUID REFERENCES curriculum_courses(id) ON DELETE SET NULL,
			active_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			placement_status VARCHAR(20) NOT NULL DEFAULT 'not_started' CHECK (placement_status IN ('not_started', 'in_progress', 'completed', 'skipped', 'self_selected')),
			primary_goal VARCHAR(30) NOT NULL DEFAULT 'conversational_fluency' CHECK (primary_goal IN ('conversational_fluency', 'structured_study', 'travel', 'work', 'exam_prep')),
			daily_goal_items INTEGER NOT NULL DEFAULT 10,
			mining_enabled BOOLEAN NOT NULL DEFAULT true,
			nudges_enabled BOOLEAN NOT NULL DEFAULT true,
			scenario_hints_enabled BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, native_language, target_language)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_language_profiles_active_unit ON user_language_profiles(active_unit_id)`,

		// Migration (task 2.3): allow onboarding level self-selection to mark a
		// profile as placement_status='self_selected'. CREATE TABLE IF NOT EXISTS
		// does not touch existing tables, so drop and re-add the CHECK with the
		// extended value set. Idempotent on every boot.
		`ALTER TABLE user_language_profiles DROP CONSTRAINT IF EXISTS user_language_profiles_placement_status_check`,
		`ALTER TABLE user_language_profiles ADD CONSTRAINT user_language_profiles_placement_status_check CHECK (placement_status IN ('not_started', 'in_progress', 'completed', 'skipped', 'self_selected'))`,

		`CREATE TABLE IF NOT EXISTS grammar_points (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
			unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			slug VARCHAR(120) NOT NULL,
			cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			title VARCHAR(255) NOT NULL,
			short_explanation TEXT NOT NULL DEFAULT '',
			examples JSONB NOT NULL DEFAULT '[]',
			prerequisites TEXT[] NOT NULL DEFAULT '{}',
			metadata JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(course_id, slug)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_grammar_points_unit ON grammar_points(unit_id)`,

		`CREATE TABLE IF NOT EXISTS lexical_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
			unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			language VARCHAR(10) NOT NULL,
			lemma VARCHAR(255) NOT NULL,
			display_text VARCHAR(255) NOT NULL,
			part_of_speech VARCHAR(40) NOT NULL DEFAULT 'unknown',
			cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			translations JSONB NOT NULL DEFAULT '{}',
			forms JSONB NOT NULL DEFAULT '{}',
			tags TEXT[] NOT NULL DEFAULT '{}',
			frequency_rank INTEGER,
			is_chunk BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(course_id, language, lemma, part_of_speech)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lexical_items_unit ON lexical_items(unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_lexical_items_lookup ON lexical_items(course_id, language, lemma)`,

		`CREATE TABLE IF NOT EXISTS curriculum_lessons (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			unit_id UUID NOT NULL REFERENCES curriculum_units(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			slug VARCHAR(120) NOT NULL,
			type VARCHAR(40) NOT NULL CHECK (type IN ('vocabulary', 'grammar', 'reading', 'listening', 'production', 'scenario_intro', 'checkpoint')),
			title VARCHAR(255) NOT NULL,
			objective TEXT NOT NULL DEFAULT '',
			estimated_minutes INTEGER NOT NULL DEFAULT 5,
			metadata JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(unit_id, ordinal),
			UNIQUE(unit_id, slug)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_curriculum_lessons_unit ON curriculum_lessons(unit_id, ordinal)`,

		`CREATE TABLE IF NOT EXISTS curriculum_lesson_steps (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			lesson_id UUID NOT NULL REFERENCES curriculum_lessons(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			type VARCHAR(40) NOT NULL CHECK (type IN ('intro', 'mcq', 'cloze', 'free_recall', 'translation', 'listening', 'speaking', 'production', 'chat_prompt', 'explanation')),
			prompt JSONB NOT NULL DEFAULT '{}',
			answer_key JSONB NOT NULL DEFAULT '{}',
			content_refs JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(lesson_id, ordinal)
		)`,

		`CREATE TABLE IF NOT EXISTS scenario_scripts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
			unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			slug VARCHAR(120) NOT NULL,
			title VARCHAR(255) NOT NULL,
			domain VARCHAR(80) NOT NULL,
			cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			can_do_statement TEXT NOT NULL,
			ai_role_name VARCHAR(120) NOT NULL,
			ai_role_description TEXT NOT NULL,
			opening_line TEXT NOT NULL,
			max_turns INTEGER NOT NULL DEFAULT 10,
			estimated_minutes INTEGER NOT NULL DEFAULT 5,
			completion_criteria JSONB NOT NULL DEFAULT '{}',
			metadata JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(course_id, slug)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scenario_scripts_course_level ON scenario_scripts(course_id, cefr_level)`,

		`CREATE TABLE IF NOT EXISTS scenario_phases (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_id UUID NOT NULL REFERENCES scenario_scripts(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			title VARCHAR(255) NOT NULL,
			learner_goal TEXT NOT NULL,
			required_intents TEXT[] NOT NULL DEFAULT '{}',
			chunk_bank JSONB NOT NULL DEFAULT '[]',
			new_lexical_item_ids UUID[] NOT NULL DEFAULT '{}',
			grammar_point_ids UUID[] NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(scenario_id, ordinal)
		)`,

		`CREATE TABLE IF NOT EXISTS user_unit_progress (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			unit_id UUID NOT NULL REFERENCES curriculum_units(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'locked' CHECK (status IN ('locked', 'available', 'in_progress', 'completed', 'skipped_by_placement')),
			progress_pct INTEGER NOT NULL DEFAULT 0 CHECK (progress_pct >= 0 AND progress_pct <= 100),
			competency_score INTEGER NOT NULL DEFAULT 0 CHECK (competency_score >= 0 AND competency_score <= 1000),
			lessons_completed INTEGER NOT NULL DEFAULT 0,
			checkpoint_score INTEGER,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, unit_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_unit_progress_user_status ON user_unit_progress(user_id, target_language, status)`,

		`CREATE TABLE IF NOT EXISTS user_lesson_attempts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			lesson_id UUID NOT NULL REFERENCES curriculum_lessons(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'abandoned')),
			score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
			correct_count INTEGER NOT NULL DEFAULT 0,
			total_count INTEGER NOT NULL DEFAULT 0,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			metadata JSONB NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_lesson_attempts_user ON user_lesson_attempts(user_id, target_language, started_at DESC)`,

		`CREATE TABLE IF NOT EXISTS lesson_step_results (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			attempt_id UUID NOT NULL REFERENCES user_lesson_attempts(id) ON DELETE CASCADE,
			step_id UUID REFERENCES curriculum_lesson_steps(id) ON DELETE SET NULL,
			user_answer JSONB NOT NULL DEFAULT '{}',
			correct BOOLEAN NOT NULL DEFAULT false,
			score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
			feedback JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS lemma VARCHAR(255)`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS normalized_term VARCHAR(255)`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS part_of_speech VARCHAR(40) DEFAULT 'unknown'`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS is_chunk BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_type VARCHAR(30) NOT NULL DEFAULT 'chat'`,
		`ALTER TABLE vocabulary DROP CONSTRAINT IF EXISTS vocabulary_source_type_check`,
		`ALTER TABLE vocabulary ADD CONSTRAINT vocabulary_source_type_check CHECK (source_type IN ('chat', 'manual', 'scenario', 'lesson', 'import', 'teacher_push')) NOT VALID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_message_id UUID REFERENCES messages(id) ON DELETE SET NULL`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_scenario_run_id UUID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS cefr_level VARCHAR(2)`,
		`ALTER TABLE vocabulary DROP CONSTRAINT IF EXISTS vocabulary_cefr_level_check`,
		`ALTER TABLE vocabulary ADD CONSTRAINT vocabulary_cefr_level_check CHECK (cefr_level IS NULL OR cefr_level IN ('A1', 'A2', 'B1', 'B2')) NOT VALID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS curriculum_lexical_item_id UUID REFERENCES lexical_items(id) ON DELETE SET NULL`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS curriculum_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS route_status VARCHAR(30) NOT NULL DEFAULT 'bonus'`,
		`ALTER TABLE vocabulary DROP CONSTRAINT IF EXISTS vocabulary_route_status_check`,
		`ALTER TABLE vocabulary ADD CONSTRAINT vocabulary_route_status_check CHECK (route_status IN ('upcoming_unit', 'completed_unit', 'current_unit', 'bonus', 'ignored')) NOT VALID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS mastery_stage INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE vocabulary DROP CONSTRAINT IF EXISTS vocabulary_mastery_stage_check`,
		`ALTER TABLE vocabulary ADD CONSTRAINT vocabulary_mastery_stage_check CHECK (mastery_stage >= 1 AND mastery_stage <= 5) NOT VALID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS mastery_state VARCHAR(30) NOT NULL DEFAULT 'new'`,
		`ALTER TABLE vocabulary DROP CONSTRAINT IF EXISTS vocabulary_mastery_state_check`,
		`ALTER TABLE vocabulary ADD CONSTRAINT vocabulary_mastery_state_check CHECK (mastery_state IN ('new', 'learning', 'reviewing', 'mastered', 'leech', 'ignored')) NOT VALID`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS ease_factor NUMERIC(4,2) NOT NULL DEFAULT 2.50`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS lapses INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS stage_success_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS production_success_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS spontaneous_use_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS teachability_score NUMERIC(5,2) NOT NULL DEFAULT 0`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS confidence NUMERIC(4,3) NOT NULL DEFAULT 0.5`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_user_stage_due ON vocabulary(user_id, language, mastery_stage, next_review)`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_curriculum_unit ON vocabulary(user_id, curriculum_unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vocabulary_normalized ON vocabulary(user_id, language, normalized_term)`,

		`CREATE TABLE IF NOT EXISTS vocabulary_practice_attempts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			stage INTEGER NOT NULL CHECK (stage >= 1 AND stage <= 5),
			activity_type VARCHAR(40) NOT NULL,
			prompt JSONB NOT NULL DEFAULT '{}',
			answer JSONB NOT NULL DEFAULT '{}',
			correct BOOLEAN NOT NULL DEFAULT false,
			quality INTEGER NOT NULL DEFAULT 0 CHECK (quality >= 0 AND quality <= 5),
			latency_ms INTEGER,
			source_session_id UUID,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vocab_attempts_vocab_created ON vocabulary_practice_attempts(vocabulary_id, created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS vocabulary_sources (
			vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
			source_type VARCHAR(30) NOT NULL,
			source_id UUID,
			sentence TEXT NOT NULL DEFAULT '',
			seen_count INTEGER NOT NULL DEFAULT 1,
			first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (vocabulary_id, source_type, source_id)
		)`,

		`CREATE TABLE IF NOT EXISTS word_mining_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
			source_type VARCHAR(30) NOT NULL DEFAULT 'chat' CHECK (source_type IN ('chat', 'scenario', 'lesson')),
			source_text TEXT NOT NULL,
			source_language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL DEFAULT 'en',
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed', 'ignored')),
			result JSONB,
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_at TIMESTAMP,
			completed_at TIMESTAMP,
			UNIQUE(user_id, message_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_word_mining_jobs_status ON word_mining_jobs(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS idx_word_mining_jobs_user ON word_mining_jobs(user_id, created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS mined_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			job_id UUID REFERENCES word_mining_jobs(id) ON DELETE SET NULL,
			chat_id UUID REFERENCES chats(id) ON DELETE SET NULL,
			message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
			source_type VARCHAR(30) NOT NULL DEFAULT 'chat',
			surface_text VARCHAR(255) NOT NULL,
			lemma VARCHAR(255) NOT NULL,
			normalized_text VARCHAR(255) NOT NULL,
			language VARCHAR(10) NOT NULL,
			part_of_speech VARCHAR(40) NOT NULL DEFAULT 'unknown',
			translation VARCHAR(500) NOT NULL DEFAULT '',
			definition TEXT NOT NULL DEFAULT '',
			context_sentence TEXT NOT NULL DEFAULT '',
			text_span JSONB NOT NULL DEFAULT '{}',
			cefr_level VARCHAR(2) CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			confidence NUMERIC(4,3) NOT NULL DEFAULT 0.5,
			teachability_score NUMERIC(5,2) NOT NULL DEFAULT 0,
			is_chunk BOOLEAN NOT NULL DEFAULT false,
			is_proper_noun BOOLEAN NOT NULL DEFAULT false,
			grammar_tags TEXT[] NOT NULL DEFAULT '{}',
			curriculum_lexical_item_id UUID REFERENCES lexical_items(id) ON DELETE SET NULL,
			curriculum_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			route_status VARCHAR(30) NOT NULL DEFAULT 'bonus' CHECK (route_status IN ('upcoming_unit', 'completed_unit', 'current_unit', 'bonus', 'ignored')),
			status VARCHAR(30) NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate', 'auto_added', 'accepted', 'ignored', 'merged')),
			route_reason TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mined_items_user_status ON mined_items(user_id, language, status, teachability_score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_mined_items_message ON mined_items(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mined_items_lookup ON mined_items(user_id, language, normalized_text)`,

		`CREATE TABLE IF NOT EXISTS user_grammar_mastery (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			confidence NUMERIC(4,3) NOT NULL DEFAULT 0.3,
			seen_count INTEGER NOT NULL DEFAULT 0,
			correct_count INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			production_success_count INTEGER NOT NULL DEFAULT 0,
			next_review_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_error_text TEXT,
			last_seen_at TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, grammar_point_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_grammar_mastery_due ON user_grammar_mastery(user_id, target_language, next_review_at)`,

		`CREATE TABLE IF NOT EXISTS learning_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			mode VARCHAR(40) NOT NULL DEFAULT 'daily' CHECK (mode IN ('daily', 'quick_drill', 'vocabulary', 'lesson', 'scenario', 'grammar', 'streak_recovery')),
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'abandoned')),
			source_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
			source_lesson_id UUID REFERENCES curriculum_lessons(id) ON DELETE SET NULL,
			planned_item_count INTEGER NOT NULL DEFAULT 0,
			completed_item_count INTEGER NOT NULL DEFAULT 0,
			score INTEGER NOT NULL DEFAULT 0,
			xp_awarded INTEGER NOT NULL DEFAULT 0,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			metadata JSONB NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_sessions_user_created ON learning_sessions(user_id, target_language, started_at DESC)`,

		`CREATE TABLE IF NOT EXISTS learning_session_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			item_type VARCHAR(40) NOT NULL CHECK (item_type IN ('vocabulary', 'grammar', 'lesson_step', 'scenario_prompt', 'reflection')),
			vocabulary_id UUID REFERENCES vocabulary(id) ON DELETE SET NULL,
			grammar_point_id UUID REFERENCES grammar_points(id) ON DELETE SET NULL,
			lesson_step_id UUID REFERENCES curriculum_lesson_steps(id) ON DELETE SET NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			result JSONB,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'skipped')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(session_id, ordinal)
		)`,

		`CREATE TABLE IF NOT EXISTS scenario_runs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			scenario_id UUID NOT NULL REFERENCES scenario_scripts(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL DEFAULT 'en',
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'abandoned', 'failed')),
			scaffold_level VARCHAR(20) NOT NULL DEFAULT 'guided' CHECK (scaffold_level IN ('guided', 'hinted', 'unscaffolded')),
			current_phase_ordinal INTEGER NOT NULL DEFAULT 1,
			phase_scores JSONB NOT NULL DEFAULT '{}',
			covered_intents TEXT[] NOT NULL DEFAULT '{}',
			score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
			xp_awarded INTEGER NOT NULL DEFAULT 0,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			metadata JSONB NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scenario_runs_user ON scenario_runs(user_id, target_language, started_at DESC)`,

		`CREATE TABLE IF NOT EXISTS scenario_turns (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			run_id UUID NOT NULL REFERENCES scenario_runs(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			speaker VARCHAR(20) NOT NULL CHECK (speaker IN ('user', 'ai', 'system')),
			text TEXT NOT NULL,
			translation TEXT NOT NULL DEFAULT '',
			phase_ordinal INTEGER NOT NULL,
			evaluation JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(run_id, ordinal)
		)`,

		`CREATE TABLE IF NOT EXISTS placement_attempts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			native_language VARCHAR(10) NOT NULL DEFAULT 'en',
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'abandoned')),
			estimated_cefr VARCHAR(2) CHECK (estimated_cefr IN ('A1', 'A2', 'B1', 'B2')),
			readiness_score INTEGER NOT NULL DEFAULT 0,
			ability_estimate NUMERIC(5,2) NOT NULL DEFAULT 0,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			metadata JSONB NOT NULL DEFAULT '{}'
		)`,

		`CREATE TABLE IF NOT EXISTS placement_responses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			attempt_id UUID NOT NULL REFERENCES placement_attempts(id) ON DELETE CASCADE,
			item_ref VARCHAR(120) NOT NULL,
			item_type VARCHAR(40) NOT NULL,
			cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			prompt JSONB NOT NULL DEFAULT '{}',
			user_answer JSONB NOT NULL DEFAULT '{}',
			correct BOOLEAN NOT NULL DEFAULT false,
			score INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS user_activity_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			event_type VARCHAR(60) NOT NULL,
			source_type VARCHAR(40) NOT NULL DEFAULT 'learning',
			source_id UUID,
			xp INTEGER NOT NULL DEFAULT 0,
			payload JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_activity_events_user_date ON user_activity_events(user_id, target_language, created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS daily_learning_stats (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			activity_date DATE NOT NULL DEFAULT CURRENT_DATE,
			xp INTEGER NOT NULL DEFAULT 0,
			items_completed INTEGER NOT NULL DEFAULT 0,
			reviews_completed INTEGER NOT NULL DEFAULT 0,
			lessons_completed INTEGER NOT NULL DEFAULT 0,
			scenarios_completed INTEGER NOT NULL DEFAULT 0,
			corrections_completed INTEGER NOT NULL DEFAULT 0,
			minutes_active INTEGER NOT NULL DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, target_language, activity_date)
		)`,

		`CREATE TABLE IF NOT EXISTS fluency_score_snapshots (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_language VARCHAR(10) NOT NULL,
			current_cefr_level VARCHAR(2) NOT NULL CHECK (current_cefr_level IN ('A1', 'A2', 'B1', 'B2')),
			readiness_score INTEGER NOT NULL CHECK (readiness_score >= 0 AND readiness_score <= 1000),
			component_scores JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS teacher_applications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			bio TEXT NOT NULL CHECK (char_length(bio) >= 10 AND char_length(bio) <= 1000),
			languages TEXT[] NOT NULL,
			expertise TEXT NOT NULL DEFAULT '',
			rate_cents INTEGER NOT NULL CHECK (rate_cents > 0),
			video_url TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','assessment_passed','recording_uploaded','certs_verified','review_scheduled','reviewed','approved','rejected','needs_work')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_applications_user ON teacher_applications(user_id)`,
		`CREATE TABLE IF NOT EXISTS teacher_certificates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			application_id UUID NOT NULL REFERENCES teacher_applications(id) ON DELETE CASCADE,
			type VARCHAR(30) NOT NULL CHECK (type IN ('teaching_degree','language_certificate','other')),
			issuer VARCHAR(255) NOT NULL,
			year INTEGER NOT NULL CHECK (year >= 1900 AND year <= 2030),
			file_url TEXT NOT NULL,
			verified BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_certs_application ON teacher_certificates(application_id)`,
		`CREATE TABLE IF NOT EXISTS tutor_reviews (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
			comment TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(teacher_user_id, student_user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_reviews_teacher ON tutor_reviews(teacher_user_id, rating)`,
		`CREATE TABLE IF NOT EXISTS tutor_trial_credits (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			credits INTEGER NOT NULL DEFAULT 1 CHECK (credits >= 0),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tutor_availability (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL CHECK (end_time > start_time),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_availability_teacher ON tutor_availability(teacher_user_id, start_time)`,
		`CREATE TABLE IF NOT EXISTS tutor_bookings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL CHECK (end_time > start_time),
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','cancelled','completed')),
			is_trial BOOLEAN NOT NULL DEFAULT false,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_bookings_teacher ON tutor_bookings(teacher_user_id, start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_bookings_student ON tutor_bookings(student_user_id, start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_bookings_status ON tutor_bookings(status)`,
		`ALTER TABLE tutor_bookings ADD COLUMN IF NOT EXISTS review_notes TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tutor_bookings ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP`,
		`ALTER TABLE tutor_bookings ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP`,
		`CREATE TABLE IF NOT EXISTS teacher_srs_pushes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			booking_id UUID REFERENCES tutor_bookings(id) ON DELETE SET NULL,
			language VARCHAR(10) NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			item_count INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_srs_pushes_teacher ON teacher_srs_pushes(teacher_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_srs_pushes_student ON teacher_srs_pushes(student_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_srs_pushes_booking ON teacher_srs_pushes(booking_id)`,
		`CREATE TABLE IF NOT EXISTS teacher_srs_push_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			push_id UUID NOT NULL REFERENCES teacher_srs_pushes(id) ON DELETE CASCADE,
			vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
			term VARCHAR(255) NOT NULL,
			translation VARCHAR(500) NOT NULL DEFAULT '',
			definition TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_srs_push_items_push ON teacher_srs_push_items(push_id)`,
		`CREATE INDEX IF NOT EXISTS idx_teacher_srs_push_items_vocab ON teacher_srs_push_items(vocabulary_id)`,
		`CREATE TABLE IF NOT EXISTS teacher_payout_methods (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL CHECK (type IN ('paypal','bank')),
			label VARCHAR(100) NOT NULL,
			details VARCHAR(255) NOT NULL,
			is_default BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payout_methods_teacher ON teacher_payout_methods(teacher_user_id, is_default)`,
		`CREATE TABLE IF NOT EXISTS teacher_payouts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			teacher_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
			fee_cents INTEGER NOT NULL DEFAULT 0,
			gross_cents INTEGER NOT NULL DEFAULT 0,
			method_id UUID REFERENCES teacher_payout_methods(id) ON DELETE SET NULL,
			destination TEXT NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','completed','failed')),
			reference VARCHAR(64) NOT NULL UNIQUE,
			paypal_batch_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payouts_teacher_created ON teacher_payouts(teacher_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_payouts_status ON teacher_payouts(status)`,

		`CREATE TABLE IF NOT EXISTS data_deletion_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed','failed')),
			requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deletion_requests_user ON data_deletion_requests(user_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Migration error: %v", err)
			return err
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}
