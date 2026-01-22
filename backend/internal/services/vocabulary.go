package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// VocabularyService handles vocabulary management and spaced repetition
type VocabularyService struct {
	db    *sql.DB
	redis *redis.Client
}

// NewVocabularyService creates a new Vocabulary service
func NewVocabularyService(db *sql.DB, redis *redis.Client) *VocabularyService {
	return &VocabularyService{
		db:    db,
		redis: redis,
	}
}

// SaveWord saves a word to the user's vocabulary
func (s *VocabularyService) SaveWord(userID string, req models.SaveVocabularyRequest, messageService *MessageService, translationService *TranslationService) (*models.VocabularyEntry, error) {
	ctx := context.Background()
	// Get the message for context
	message, err := messageService.GetMessageByID(ctx, req.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Get translation
	translation, err := translationService.Translate(req.Term, "en") // Translate to English
	if err != nil {
		translation = "" // Continue without translation
	}

	// Get definition (could use an external dictionary API in production)
	definition := s.generateDefinition(req.Term, req.Language)

	var entry models.VocabularyEntry
	err = s.db.QueryRow(`
		INSERT INTO vocabulary (
			user_id, term, language, translation, definition,
			context_message_id, context_sentence, context_chat_id,
			next_review
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP + INTERVAL '1 day')
		ON CONFLICT (user_id, term, language) 
		DO UPDATE SET 
			translation = EXCLUDED.translation,
			definition = EXCLUDED.definition,
			context_message_id = EXCLUDED.context_message_id,
			context_sentence = EXCLUDED.context_sentence,
			context_chat_id = EXCLUDED.context_chat_id
		RETURNING id, user_id, term, language, translation, definition,
			context_message_id, context_sentence, context_chat_id,
			review_count, correct_count, next_review, interval_days, created_at
	`, userID, req.Term, req.Language, translation, definition,
		message.ID, message.Text, message.ChatID).Scan(
		&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
		&entry.Translation, &entry.Definition,
		&entry.Context.MessageID, &entry.Context.Sentence, &entry.Context.ChatID,
		&entry.LearningData.ReviewCount, &entry.LearningData.CorrectCount,
		&entry.LearningData.NextReview, &entry.LearningData.Interval, &entry.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save vocabulary: %w", err)
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("vocab:%s:due", userID))
	s.redis.Del(ctx, fmt.Sprintf("vocab:%s:all", userID))

	return &entry, nil
}

// GetUserVocabulary returns all vocabulary entries for a user
func (s *VocabularyService) GetUserVocabulary(userID string, language string, limit, offset int) ([]models.VocabularyEntry, int, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("vocab:%s:all:%s:%d:%d", userID, language, limit, offset)

	// Try cache first
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var entries []models.VocabularyEntry
		if json.Unmarshal([]byte(cached), &entries) == nil {
			// Get total count
			var total int
			s.db.QueryRow(`SELECT COUNT(*) FROM vocabulary WHERE user_id = $1 AND ($2 = '' OR language = $2)`, userID, language).Scan(&total)
			return entries, total, nil
		}
	}

	// Query database
	query := `
		SELECT id, user_id, term, language, translation, definition,
			context_message_id, context_sentence, context_chat_id,
			review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1 AND ($2 = '' OR language = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(query, userID, language, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get vocabulary: %w", err)
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
			return nil, 0, err
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

	// Get total count
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM vocabulary WHERE user_id = $1 AND ($2 = '' OR language = $2)`, userID, language).Scan(&total)

	// Cache result
	if jsonData, err := json.Marshal(entries); err == nil {
		s.redis.Set(ctx, cacheKey, jsonData, 5*time.Minute)
	}

	return entries, total, nil
}

// GetDueVocabulary returns vocabulary entries due for review
func (s *VocabularyService) GetDueVocabulary(userID string, limit int) ([]models.VocabularyEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, term, language, translation, definition,
			context_message_id, context_sentence, context_chat_id,
			review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1 AND next_review <= CURRENT_TIMESTAMP
		ORDER BY next_review ASC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due vocabulary: %w", err)
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

// RecordPracticeResult records the result of a practice session
func (s *VocabularyService) RecordPracticeResult(userID, vocabularyID string, correct bool) error {
	ctx := context.Background()

	// Get current learning data
	var reviewCount, correctCount, intervalDays int
	err := s.db.QueryRow(`
		SELECT review_count, correct_count, interval_days
		FROM vocabulary
		WHERE id = $1 AND user_id = $2
	`, vocabularyID, userID).Scan(&reviewCount, &correctCount, &intervalDays)

	if err != nil {
		return fmt.Errorf("failed to get vocabulary: %w", err)
	}

	// Apply spaced repetition algorithm (SM-2 simplified)
	reviewCount++
	if correct {
		correctCount++
		// Increase interval using exponential growth
		intervalDays = s.calculateNextInterval(intervalDays, float64(correctCount)/float64(reviewCount))
	} else {
		// Reset to shorter interval on incorrect
		intervalDays = 1
	}

	// Update database
	_, err = s.db.Exec(`
		UPDATE vocabulary
		SET review_count = $1, correct_count = $2, interval_days = $3,
			next_review = CURRENT_TIMESTAMP + ($3 || ' days')::INTERVAL
		WHERE id = $4 AND user_id = $5
	`, reviewCount, correctCount, intervalDays, vocabularyID, userID)

	if err != nil {
		return fmt.Errorf("failed to update vocabulary: %w", err)
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("vocab:%s:due", userID))

	return nil
}

// calculateNextInterval calculates the next review interval using SM-2 algorithm
func (s *VocabularyService) calculateNextInterval(currentInterval int, accuracy float64) int {
	// Base multiplier based on accuracy
	var multiplier float64
	if accuracy >= 0.9 {
		multiplier = 2.5
	} else if accuracy >= 0.7 {
		multiplier = 2.0
	} else if accuracy >= 0.5 {
		multiplier = 1.5
	} else {
		multiplier = 1.0
	}

	newInterval := float64(currentInterval) * multiplier

	// Cap at 365 days
	if newInterval > 365 {
		newInterval = 365
	}

	return int(math.Ceil(newInterval))
}

// DeleteVocabulary deletes a vocabulary entry
func (s *VocabularyService) DeleteVocabulary(userID, vocabularyID string) error {
	ctx := context.Background()

	result, err := s.db.Exec(`
		DELETE FROM vocabulary WHERE id = $1 AND user_id = $2
	`, vocabularyID, userID)

	if err != nil {
		return fmt.Errorf("failed to delete vocabulary: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("vocabulary not found")
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("vocab:%s:due", userID))
	s.redis.Del(ctx, fmt.Sprintf("vocab:%s:all", userID))

	return nil
}

// GetLearningProgress returns learning statistics for a user
func (s *VocabularyService) GetLearningProgress(userID string) (map[string]interface{}, error) {
	var totalWords, wordsReviewed, wordsMastered int
	var languages []string

	// Get total words and reviewed words
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN review_count > 0 THEN 1 END),
			COUNT(CASE WHEN interval_days >= 30 THEN 1 END)
		FROM vocabulary WHERE user_id = $1
	`, userID).Scan(&totalWords, &wordsReviewed, &wordsMastered)
	if err != nil {
		return nil, err
	}

	// Get languages
	rows, err := s.db.Query(`
		SELECT DISTINCT language FROM vocabulary WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err == nil {
			languages = append(languages, lang)
		}
	}

	// Get due today
	var dueToday int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM vocabulary 
		WHERE user_id = $1 AND next_review <= CURRENT_TIMESTAMP
	`, userID).Scan(&dueToday)

	// Get streak (consecutive days with activity)
	var streak int
	s.db.QueryRow(`
		WITH daily_activity AS (
			SELECT DATE(created_at) as activity_date
			FROM vocabulary WHERE user_id = $1
			UNION
			SELECT DATE(next_review - (interval_days || ' days')::INTERVAL)
			FROM vocabulary WHERE user_id = $1 AND review_count > 0
		)
		SELECT COUNT(DISTINCT activity_date)
		FROM daily_activity
		WHERE activity_date >= CURRENT_DATE - INTERVAL '30 days'
	`, userID).Scan(&streak)

	return map[string]interface{}{
		"totalWords":    totalWords,
		"wordsReviewed": wordsReviewed,
		"wordsMastered": wordsMastered,
		"dueToday":      dueToday,
		"streak":        streak,
		"languages":     languages,
	}, nil
}

// SearchVocabulary finds entries matching a query with optional language filter.
func (s *VocabularyService) SearchVocabulary(userID, query, language string, limit int) ([]models.VocabularyEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	sqlQuery := `
		SELECT id, user_id, term, language, translation, definition,
		       context_message_id, context_sentence, context_chat_id,
		       review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1
		  AND (term ILIKE $2 OR translation ILIKE $2 OR definition ILIKE $2)
	`
	args := []interface{}{userID, "%" + query + "%"}

	if language != "" {
		sqlQuery += " AND language = $3"
		args = append(args, language)
	}

	sqlQuery += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := s.db.Query(sqlQuery, args...)
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

// generateDefinition generates a simple definition (placeholder for dictionary API)
func (s *VocabularyService) generateDefinition(term, language string) string {
	// In production, this would call an external dictionary API
	return fmt.Sprintf("Definition for '%s' in %s", term, language)
}

// GetVocabularyByID returns a single vocabulary entry
func (s *VocabularyService) GetVocabularyByID(userID, vocabularyID string) (*models.VocabularyEntry, error) {
	var entry models.VocabularyEntry
	entry.LearningData = &models.LearningData{}

	var contextMsgID, contextSentence, contextChatID sql.NullString

	err := s.db.QueryRow(`
		SELECT id, user_id, term, language, translation, definition,
			context_message_id, context_sentence, context_chat_id,
			review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE id = $1 AND user_id = $2
	`, vocabularyID, userID).Scan(
		&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
		&entry.Translation, &entry.Definition,
		&contextMsgID, &contextSentence, &contextChatID,
		&entry.LearningData.ReviewCount, &entry.LearningData.CorrectCount,
		&entry.LearningData.NextReview, &entry.LearningData.Interval, &entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vocabulary not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vocabulary: %w", err)
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

	return &entry, nil
}
