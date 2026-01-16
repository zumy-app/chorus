package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// SearchService handles message and content search
type SearchService struct {
	db    *sql.DB
	redis *redis.Client
}

// NewSearchService creates a new Search service
func NewSearchService(db *sql.DB, redis *redis.Client) *SearchService {
	return &SearchService{
		db:    db,
		redis: redis,
	}
}

// SearchMessagesSimple preserves the legacy signature used by older handlers.
func (s *SearchService) SearchMessagesSimple(userID, chatID, query string, limit int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	req := models.SearchRequest{
		Query: query,
		ChatIDs: func() []string {
			if chatID == "" {
				return nil
			}
			return []string{chatID}
		}(),
		Limit:  limit,
		Offset: 0,
	}

	res, err := s.SearchMessages(userID, req)
	if err != nil {
		return nil, err
	}
	return res.Messages, nil
}

// SearchMessages searches for messages matching the query
func (s *SearchService) SearchMessages(userID string, req models.SearchRequest) (*models.SearchResult, error) {
	ctx := context.Background()

	// Set defaults
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Build the query
	query := `
		SELECT m.id, m.chat_id, m.sender_id, m.text, 
			   COALESCE(m.original_language, '') as original_language,
			   COALESCE(m.translations::text, '{}') as translations,
			   m.delivery_status, m.reply_to_id, m.created_at
		FROM messages m
		JOIN chat_participants cp ON m.chat_id = cp.chat_id
		WHERE cp.user_id = $1
		AND to_tsvector('english', m.text) @@ plainto_tsquery('english', $2)
	`

	args := []interface{}{userID, req.Query}
	argNum := 3

	// Filter by chat IDs if provided
	if len(req.ChatIDs) > 0 {
		query += fmt.Sprintf(" AND m.chat_id = ANY($%d)", argNum)
		args = append(args, req.ChatIDs)
		argNum++
	}

	// Filter by language if provided
	if req.Language != "" {
		query += fmt.Sprintf(" AND m.original_language = $%d", argNum)
		args = append(args, req.Language)
		argNum++
	}

	// Get total count
	countQuery := strings.Replace(query,
		"SELECT m.id, m.chat_id, m.sender_id, m.text,",
		"SELECT COUNT(*)", 1)
	countQuery = strings.Split(countQuery, "COALESCE")[0]
	countQuery = strings.TrimSuffix(countQuery, ", ")
	countQuery += " FROM messages m JOIN chat_participants cp ON m.chat_id = cp.chat_id WHERE cp.user_id = $1 AND to_tsvector('english', m.text) @@ plainto_tsquery('english', $2)"

	var total int
	err := s.db.QueryRow(countQuery, args[:2]...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY m.created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, req.Limit, req.Offset)

	// Execute search
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var translationsJSON string
		var replyToID sql.NullString

		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
			&msg.OriginalLanguage, &translationsJSON,
			&msg.DeliveryStatus, &replyToID, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}

		if replyToID.Valid {
			msg.ReplyToID = &replyToID.String
		}

		// Parse translations JSON
		if translationsJSON != "" && translationsJSON != "{}" {
			msg.Translations = make(map[string]string)
			// Simple JSON parsing for translations
		}

		messages = append(messages, msg)
	}

	// Cache search result
	cacheKey := fmt.Sprintf("search:%s:%s:%d:%d", userID, req.Query, req.Limit, req.Offset)
	s.redis.Set(ctx, cacheKey, total, 5*time.Minute)

	return &models.SearchResult{
		Messages: messages,
		Total:    total,
		HasMore:  req.Offset+len(messages) < total,
	}, nil
}

// SearchByExactText searches for exact text matches
func (s *SearchService) SearchByExactText(userID, text string, limit int) ([]models.Message, error) {
	query := `
		SELECT m.id, m.chat_id, m.sender_id, m.text, 
			   COALESCE(m.original_language, '') as original_language,
			   COALESCE(m.translations::text, '{}') as translations,
			   m.delivery_status, m.reply_to_id, m.created_at
		FROM messages m
		JOIN chat_participants cp ON m.chat_id = cp.chat_id
		WHERE cp.user_id = $1
		AND m.text ILIKE $2
		ORDER BY m.created_at DESC
		LIMIT $3
	`

	pattern := "%" + text + "%"
	rows, err := s.db.Query(query, userID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var translationsJSON string
		var replyToID sql.NullString

		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
			&msg.OriginalLanguage, &translationsJSON,
			&msg.DeliveryStatus, &replyToID, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}

		if replyToID.Valid {
			msg.ReplyToID = &replyToID.String
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SearchInChat searches for messages within a specific chat
func (s *SearchService) SearchInChat(userID, chatID, query string, limit int) ([]models.Message, error) {
	// Verify user is a participant
	var participantID string
	err := s.db.QueryRow(`
		SELECT user_id FROM chat_participants WHERE chat_id = $1 AND user_id = $2
	`, chatID, userID).Scan(&participantID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not a chat participant")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify participation: %w", err)
	}

	sqlQuery := `
		SELECT id, chat_id, sender_id, text, 
			   COALESCE(original_language, '') as original_language,
			   COALESCE(translations::text, '{}') as translations,
			   delivery_status, reply_to_id, created_at
		FROM messages
		WHERE chat_id = $1
		AND to_tsvector('english', text) @@ plainto_tsquery('english', $2)
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := s.db.Query(sqlQuery, chatID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search chat messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var translationsJSON string
		var replyToID sql.NullString

		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
			&msg.OriginalLanguage, &translationsJSON,
			&msg.DeliveryStatus, &replyToID, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}

		if replyToID.Valid {
			msg.ReplyToID = &replyToID.String
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SearchVocabulary searches vocabulary entries
func (s *SearchService) SearchVocabulary(userID, query string, limit int) ([]models.VocabularyEntry, error) {
	sqlQuery := `
		SELECT id, user_id, term, language, translation, definition,
			   context_message_id, context_sentence, context_chat_id,
			   review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1
		AND (term ILIKE $2 OR translation ILIKE $2 OR definition ILIKE $2)
		ORDER BY created_at DESC
		LIMIT $3
	`

	pattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, userID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vocabulary: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabularyEntry
	for rows.Next() {
		var entry models.VocabularyEntry
		entry.LearningData = &models.LearningData{}

		var contextMsgID, contextSentence, contextChatID sql.NullString

		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
			&entry.Translation, &entry.Definition,
			&contextMsgID, &contextSentence, &contextChatID,
			&entry.LearningData.ReviewCount, &entry.LearningData.CorrectCount,
			&entry.LearningData.NextReview, &entry.LearningData.Interval, &entry.CreatedAt,
		); err != nil {
			return nil, err
		}

		if contextMsgID.Valid {
			entry.Context.MessageID = contextMsgID.String
		}
		if contextSentence.Valid {
			entry.Context.Sentence = contextSentence.String
		}
		if contextChatID.Valid {
			entry.Context.ChatID = contextChatID.String
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// SearchVocabularyWithLanguage supports an optional language filter for compatibility with older handlers.
func (s *SearchService) SearchVocabularyWithLanguage(userID, query, language string, limit int) ([]models.VocabularyEntry, error) {
	if language == "" {
		return s.SearchVocabulary(userID, query, limit)
	}

	sqlQuery := `
		SELECT id, user_id, term, language, translation, definition,
			   context_message_id, context_sentence, context_chat_id,
			   review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1 AND language = $2
		  AND (term ILIKE $3 OR translation ILIKE $3 OR definition ILIKE $3)
		ORDER BY created_at DESC
		LIMIT $4
	`

	pattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, userID, language, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vocabulary: %w", err)
	}
	defer rows.Close()

	var entries []models.VocabularyEntry
	for rows.Next() {
		var entry models.VocabularyEntry
		entry.LearningData = &models.LearningData{}

		var contextMsgID, contextSentence, contextChatID sql.NullString

		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
			&entry.Translation, &entry.Definition,
			&contextMsgID, &contextSentence, &contextChatID,
			&entry.LearningData.ReviewCount, &entry.LearningData.CorrectCount,
			&entry.LearningData.NextReview, &entry.LearningData.Interval, &entry.CreatedAt,
		); err != nil {
			return nil, err
		}

		if contextMsgID.Valid {
			entry.Context.MessageID = contextMsgID.String
		}
		if contextSentence.Valid {
			entry.Context.Sentence = contextSentence.String
		}
		if contextChatID.Valid {
			entry.Context.ChatID = contextChatID.String
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// SearchChats finds chats a user participates in by partial name.
func (s *SearchService) SearchChats(userID, query string) ([]models.Chat, error) {
	sqlQuery := `
		SELECT DISTINCT c.id, c.name, c.type, c.created_at
		FROM chats c
		INNER JOIN chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1 AND c.name ILIKE $2
		ORDER BY c.created_at DESC
		LIMIT 20
	`

	rows, err := s.db.Query(sqlQuery, userID, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search chats: %w", err)
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		if err := rows.Scan(&chat.ID, &chat.Name, &chat.Type, &chat.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// SearchContacts finds users by username, display name, or email, excluding the requester.
func (s *SearchService) SearchContacts(userID, query string) ([]models.User, error) {
	sqlQuery := `
		SELECT u.id, u.username, u.email, u.display_name,
		       u.native_language, u.target_languages, u.created_at
		FROM users u
		WHERE u.id != $1
		  AND (u.username ILIKE $2 OR u.display_name ILIKE $2 OR u.email ILIKE $2)
		ORDER BY u.username
		LIMIT 20
	`

	rows, err := s.db.Query(sqlQuery, userID, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.DisplayName,
			&user.NativeLanguage, &user.TargetLanguages, &user.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetSearchSuggestions returns search suggestions based on previous searches
func (s *SearchService) GetSearchSuggestions(userID, prefix string, limit int) ([]string, error) {
	ctx := context.Background()

	// Get recent search terms from Redis
	cacheKey := fmt.Sprintf("search_history:%s", userID)
	searches, err := s.redis.ZRevRange(ctx, cacheKey, 0, int64(limit*2)).Result()
	if err != nil {
		return nil, err
	}

	// Filter by prefix
	var suggestions []string
	prefixLower := strings.ToLower(prefix)
	for _, search := range searches {
		if strings.HasPrefix(strings.ToLower(search), prefixLower) {
			suggestions = append(suggestions, search)
			if len(suggestions) >= limit {
				break
			}
		}
	}

	return suggestions, nil
}

// RecordSearch records a search for suggestions
func (s *SearchService) RecordSearch(userID, query string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("search_history:%s", userID)

	// Add to sorted set with timestamp as score
	score := float64(time.Now().Unix())
	s.redis.ZAdd(ctx, cacheKey, redis.Z{Score: score, Member: query})

	// Keep only last 100 searches
	s.redis.ZRemRangeByRank(ctx, cacheKey, 0, -101)

	return nil
}

// GetRecentSearches returns recent searches for a user
func (s *SearchService) GetRecentSearches(userID string, limit int) ([]string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("search_history:%s", userID)

	return s.redis.ZRevRange(ctx, cacheKey, 0, int64(limit-1)).Result()
}

// ClearSearchHistory clears search history for a user
func (s *SearchService) ClearSearchHistory(userID string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("search_history:%s", userID)

	return s.redis.Del(ctx, cacheKey).Err()
}
