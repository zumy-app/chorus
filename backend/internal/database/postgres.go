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
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL`,

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
