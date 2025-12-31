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
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(30) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			display_name VARCHAR(100) NOT NULL,
			native_language VARCHAR(10) NOT NULL,
			target_languages TEXT[] DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
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
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}
