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
