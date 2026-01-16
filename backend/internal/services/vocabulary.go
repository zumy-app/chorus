package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

type VocabularyService struct {
	db                 *sql.DB
	translationService *TranslationService
}

func NewVocabularyService(db *sql.DB, translationService *TranslationService) *VocabularyService {
	return &VocabularyService{
		db:                 db,
		translationService: translationService,
	}
}

// SaveVocabulary saves a new vocabulary entry from a message
func (s *VocabularyService) SaveVocabulary(ctx context.Context, userID string, req models.SaveVocabularyRequest, messageText string, chatID string) (*models.VocabularyEntry, error) {
	// Get translation
	translation, err := s.translationService.Translate(req.Term, "en")
	if err != nil {
		return nil, fmt.Errorf("failed to translate term: %w", err)
	}
	
	// Generate definition (in real implementation, use dictionary API)
	definition := fmt.Sprintf("Definition of '%s' in %s", req.Term, req.Language)
	
	vocabID := uuid.New().String()
	
	query := `
		INSERT INTO vocabulary (
			id, user_id, term, language, translation, definition, 
			context_message_id, context_sentence, context_chat_id,
			review_count, correct_count, next_review, interval_days, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	_, err = s.db.ExecContext(ctx, query,
		vocabID, userID, req.Term, req.Language, translation, definition,
		req.MessageID, messageText, chatID,
		0, 0, time.Now().Add(24 * time.Hour), 1, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save vocabulary: %w", err)
	}
	
	return &models.VocabularyEntry{
		ID:          vocabID,
		UserID:      userID,
		Term:        req.Term,
		Language:    req.Language,
		Translation: translation,
		Definition:  definition,
		Context: models.VocabContext{
			MessageID: req.MessageID,
			Sentence:  messageText,
			ChatID:    chatID,
		},
		LearningData: &models.LearningData{
			ReviewCount:  0,
			CorrectCount: 0,
			NextReview:   time.Now().Add(24 * time.Hour),
			Interval:     1,
		},
		CreatedAt:    time.Now(),
	}, nil
}

// GetVocabularyDueForReview retrieves vocabulary items due for review
func (s *VocabularyService) GetVocabularyDueForReview(ctx context.Context, userID string, limit int) ([]models.VocabularyEntry, error) {
	query := `
		SELECT id, user_id, term, language, translation, definition, 
		       context_message_id, context_sentence, context_chat_id,
		       review_count, correct_count, next_review, interval_days, created_at
		FROM vocabulary
		WHERE user_id = $1 
		  AND next_review <= $2
		ORDER BY next_review ASC
		LIMIT $3
	`
	
	rows, err := s.db.QueryContext(ctx, query, userID, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query due vocabulary: %w", err)
	}
	defer rows.Close()
	
	entries := []models.VocabularyEntry{}
	for rows.Next() {
		var entry models.VocabularyEntry
		var contextMessageID sql.NullString
		var contextSentence sql.NullString
		var contextChatID sql.NullString
		var learningData models.LearningData
		
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
			&entry.Translation, &entry.Definition,
			&contextMessageID, &contextSentence, &contextChatID,
			&learningData.ReviewCount, &learningData.CorrectCount,
			&learningData.NextReview, &learningData.Interval, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		if contextMessageID.Valid {
			entry.Context.MessageID = contextMessageID.String
		}
		if contextSentence.Valid {
			entry.Context.Sentence = contextSentence.String
		}
		if contextChatID.Valid {
			entry.Context.ChatID = contextChatID.String
		}
		
		entry.LearningData = &learningData
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// UpdatePracticeResult updates learning progress based on practice result
func (s *VocabularyService) UpdatePracticeResult(ctx context.Context, userID string, vocabularyID string, correct bool) error {
	// Get current entry
	var learningDataJSON []byte
	query := `SELECT learning_data FROM vocabulary_entries WHERE id = $1 AND user_id = $2`
	err := s.db.QueryRowContext(ctx, query, vocabularyID, userID).Scan(&learningDataJSON)
	if err != nil {
		return fmt.Errorf("vocabulary entry not found: %w", err)
	}
	
	var learningData models.LearningData
	json.Unmarshal(learningDataJSON, &learningData)
	
	// Update learning data using spaced repetition algorithm (SM-2)
	learningData.ReviewCount++
	
	if correct {
		learningData.CorrectCount++
		// Increase interval (spaced repetition)
		learningData.Interval = calculateNextInterval(learningData.Interval, true)
	} else {
		// Reset interval on incorrect answer
		learningData.Interval = 1
	}
	
	// Calculate next review date
	learningData.NextReview = time.Now().Add(time.Duration(learningData.Interval) * 24 * time.Hour)
	
	// Update database
	updatedJSON, _ := json.Marshal(learningData)
	updateQuery := `UPDATE vocabulary_entries SET learning_data = $1 WHERE id = $2 AND user_id = $3`
	_, err = s.db.ExecContext(ctx, updateQuery, updatedJSON, vocabularyID, userID)
	if err != nil {
		return fmt.Errorf("failed to update practice result: %w", err)
	}
	
	return nil
}

// calculateNextInterval implements simplified SM-2 spaced repetition algorithm
func calculateNextInterval(currentInterval int, correct bool) int {
	if !correct {
		return 1
	}
	
	// Simplified SM-2 intervals
	switch currentInterval {
	case 1:
		return 3
	case 3:
		return 7
	case 7:
		return 14
	case 14:
		return 30
	case 30:
		return 60
	default:
		return currentInterval * 2
	}
}

// GetUserVocabulary retrieves all vocabulary for a user
func (s *VocabularyService) GetUserVocabulary(ctx context.Context, userID string, language string, limit int, offset int) ([]models.VocabularyEntry, error) {
	var query string
	var args []interface{}
	
	if language != "" {
		query = `
			SELECT id, user_id, term, language, translation, definition, 
			       context, learning_data, created_at
			FROM vocabulary_entries
			WHERE user_id = $1 AND language = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{userID, language, limit, offset}
	} else {
		query = `
			SELECT id, user_id, term, language, translation, definition, 
			       context, learning_data, created_at
			FROM vocabulary_entries
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userID, limit, offset}
	}
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vocabulary: %w", err)
	}
	defer rows.Close()
	
	entries := []models.VocabularyEntry{}
	for rows.Next() {
		var entry models.VocabularyEntry
		var contextJSON, learningDataJSON []byte
		
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
			&entry.Translation, &entry.Definition, &contextJSON,
			&learningDataJSON, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(contextJSON, &entry.Context)
		
		var learningData models.LearningData
		json.Unmarshal(learningDataJSON, &learningData)
		entry.LearningData = &learningData
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// DeleteVocabulary removes a vocabulary entry
func (s *VocabularyService) DeleteVocabulary(ctx context.Context, userID string, vocabularyID string) error {
	query := `DELETE FROM vocabulary_entries WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, vocabularyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete vocabulary: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("vocabulary entry not found")
	}
	
	return nil
}

// GetLearningProgress generates learning statistics
type LearningProgress struct {
	TotalWords       int            `json:"totalWords"`
	WordsLearned     int            `json:"wordsLearned"`
	WordsDueToday    int            `json:"wordsDueToday"`
	CurrentStreak    int            `json:"currentStreak"`
	LongestStreak    int            `json:"longestStreak"`
	ByLanguage       map[string]int `json:"byLanguage"`
	SuccessRate      float64        `json:"successRate"`
	TotalReviews     int            `json:"totalReviews"`
	CorrectReviews   int            `json:"correctReviews"`
}

func (s *VocabularyService) GetLearningProgress(ctx context.Context, userID string) (*LearningProgress, error) {
	progress := &LearningProgress{
		ByLanguage: make(map[string]int),
	}
	
	// Count total words
	query := `SELECT COUNT(*) FROM vocabulary_entries WHERE user_id = $1`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&progress.TotalWords)
	if err != nil {
		return nil, err
	}
	
	// Count words with at least one correct review (considered "learned")
	query = `
		SELECT COUNT(*) 
		FROM vocabulary_entries 
		WHERE user_id = $1 
		  AND (learning_data->>'correctCount')::int > 0
	`
	err = s.db.QueryRowContext(ctx, query, userID).Scan(&progress.WordsLearned)
	if err != nil {
		return nil, err
	}
	
	// Count words due today
	query = `
		SELECT COUNT(*) 
		FROM vocabulary_entries 
		WHERE user_id = $1 
		  AND (learning_data->>'nextReview')::timestamp <= $2
	`
	err = s.db.QueryRowContext(ctx, query, userID, time.Now()).Scan(&progress.WordsDueToday)
	if err != nil {
		return nil, err
	}
	
	// Get count by language
	query = `
		SELECT language, COUNT(*) as count
		FROM vocabulary_entries
		WHERE user_id = $1
		GROUP BY language
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var language string
		var count int
		rows.Scan(&language, &count)
		progress.ByLanguage[language] = count
	}
	
	// Calculate success rate
	query = `
		SELECT 
			SUM((learning_data->>'reviewCount')::int) as total_reviews,
			SUM((learning_data->>'correctCount')::int) as correct_reviews
		FROM vocabulary_entries
		WHERE user_id = $1
	`
	var totalReviews, correctReviews sql.NullInt64
	err = s.db.QueryRowContext(ctx, query, userID).Scan(&totalReviews, &correctReviews)
	if err == nil && totalReviews.Valid {
		progress.TotalReviews = int(totalReviews.Int64)
		progress.CorrectReviews = int(correctReviews.Int64)
		if progress.TotalReviews > 0 {
			progress.SuccessRate = float64(progress.CorrectReviews) / float64(progress.TotalReviews) * 100
		}
	}
	
	// Calculate streak (simplified - would need review history table for accurate tracking)
	progress.CurrentStreak = 0
	progress.LongestStreak = 0
	
	return progress, nil
}

// GetVocabularyByID retrieves a specific vocabulary entry
func (s *VocabularyService) GetVocabularyByID(ctx context.Context, userID string, vocabularyID string) (*models.VocabularyEntry, error) {
	query := `
		SELECT id, user_id, term, language, translation, definition, 
		       context, learning_data, created_at
		FROM vocabulary_entries
		WHERE id = $1 AND user_id = $2
	`
	
	var entry models.VocabularyEntry
	var contextJSON, learningDataJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, vocabularyID, userID).Scan(
		&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
		&entry.Translation, &entry.Definition, &contextJSON,
		&learningDataJSON, &entry.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vocabulary entry not found")
		}
		return nil, err
	}
	
	json.Unmarshal(contextJSON, &entry.Context)
	
	var learningData models.LearningData
	json.Unmarshal(learningDataJSON, &learningData)
	entry.LearningData = &learningData
	
	return &entry, nil
}

// SearchVocabulary searches user's vocabulary
func (s *VocabularyService) SearchVocabulary(ctx context.Context, userID string, searchTerm string, language string) ([]models.VocabularyEntry, error) {
	var query string
	var args []interface{}
	
	if language != "" {
		query = `
			SELECT id, user_id, term, language, translation, definition, 
			       context, learning_data, created_at
			FROM vocabulary_entries
			WHERE user_id = $1 
			  AND language = $2
			  AND (term ILIKE $3 OR translation ILIKE $3 OR definition ILIKE $3)
			ORDER BY created_at DESC
			LIMIT 50
		`
		args = []interface{}{userID, language, "%" + searchTerm + "%"}
	} else {
		query = `
			SELECT id, user_id, term, language, translation, definition, 
			       context, learning_data, created_at
			FROM vocabulary_entries
			WHERE user_id = $1 
			  AND (term ILIKE $2 OR translation ILIKE $2 OR definition ILIKE $2)
			ORDER BY created_at DESC
			LIMIT 50
		`
		args = []interface{}{userID, "%" + searchTerm + "%"}
	}
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search vocabulary: %w", err)
	}
	defer rows.Close()
	
	entries := []models.VocabularyEntry{}
	for rows.Next() {
		var entry models.VocabularyEntry
		var contextJSON, learningDataJSON []byte
		
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Term, &entry.Language,
			&entry.Translation, &entry.Definition, &contextJSON,
			&learningDataJSON, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(contextJSON, &entry.Context)
		
		var learningData models.LearningData
		json.Unmarshal(learningDataJSON, &learningData)
		entry.LearningData = &learningData
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}
