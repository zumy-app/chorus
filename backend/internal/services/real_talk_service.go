package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// RealTalkService serves the daily colloquial prompt set (plan V3 Phase 7).
// The full course+level dataset is compiled once per day into Redis under
// "realtalk:{courseID}:{cefrLevel}:{date}" with a 24h TTL, so the chat-ready
// prompts cost zero Postgres reads for every learner after the first request.
type RealTalkService struct {
	db    *sql.DB
	cache RealTalkCache
}

// RealTalkCache is the minimal Redis surface used by the service (satisfied by
// *redis.Client; tests inject a fake).
type RealTalkCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

func NewRealTalkService(db *sql.DB, cache RealTalkCache) *RealTalkService {
	return &RealTalkService{db: db, cache: cache}
}

// realtalkCategoryLabel maps seeded categories to the tab labels the hub UI
// filters on (Icebreakers / Deep Dives / Task-Based).
var realtalkCategoryLabel = map[string]string{
	"conversation_starter": "Icebreakers",
	"topic_injector":       "Deep Dives",
	"opinion_phrase":       "Task-Based",
}

// DailyPromptSet returns today's prompts for a course + CEFR level, from the
// daily Redis cache when warm, otherwise compiled from Postgres and cached.
func (s *RealTalkService) DailyPromptSet(ctx context.Context, courseID, cefrLevel string) ([]models.RealTalkPrompt, error) {
	if courseID == "" {
		return []models.RealTalkPrompt{}, nil
	}
	date := time.Now().UTC().Format("2006-01-02")
	cacheKey := fmt.Sprintf("realtalk:%s:%s:%s", courseID, cefrLevel, date)

	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
			var prompts []models.RealTalkPrompt
			if json.Unmarshal([]byte(cached), &prompts) == nil {
				return prompts, nil
			}
		}
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, category, prompt_for_learner, target_phrase, COALESCE(why_useful,''),
		       COALESCE(follow_up_chunks::text,'[]')
		FROM real_talk_prompts
		WHERE course_id = $1 AND cefr_level = $2 AND is_active = true
		ORDER BY category, created_at`, courseID, cefrLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prompts := []models.RealTalkPrompt{}
	for rows.Next() {
		var p models.RealTalkPrompt
		var category string
		var chunks string
		if err := rows.Scan(&p.ID, &category, &p.Text, &p.TargetPhrase, &p.WhyUseful, &chunks); err != nil {
			return nil, err
		}
		p.Category = realtalkCategoryLabel[category]
		if p.Category == "" {
			p.Category = category
		}
		if len(chunks) > 0 {
			var c []string
			if json.Unmarshal([]byte(chunks), &c) == nil {
				p.FollowUpChunks = c
			}
		}
		prompts = append(prompts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if s.cache != nil && len(prompts) > 0 {
		if data, err := json.Marshal(prompts); err == nil {
			_ = s.cache.Set(ctx, cacheKey, string(data), 24*time.Hour)
		}
	}
	return prompts, nil
}

// CourseForPair resolves the active course id for a language pair (empty when
// no full course exists).
func (s *RealTalkService) CourseForPair(ctx context.Context, nativeLang, targetLang string) (string, error) {
	var id sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT active_course_id::text FROM learning_pair_capabilities
		WHERE native_language = $1 AND target_language = $2`, nativeLang, targetLang).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if id.Valid {
		return id.String, nil
	}
	return "", nil
}
