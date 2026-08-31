package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// Unified queue tuning.
const (
	grammarClozeLimit = 4
	queueDefaultLimit = 12
	queueMaxLimit     = 24
)

// SRSQueueService implements the FR-32 "unified queue": it interleaves the
// card-level spaced-repetition backlog from a learner's seed (curriculum) and
// personal (mined) vocabulary cards with due grammar-cloze items, so a single
// endpoint drives the daily review session.
type SRSQueueService struct {
	db       *sql.DB
	practice *PracticeService
	seed     *SeedQueueService
}

func NewSRSQueueService(db *sql.DB, practice *PracticeService, seed *SeedQueueService) *SRSQueueService {
	return &SRSQueueService{db: db, practice: practice, seed: seed}
}

// GetQueue builds the interleaved review queue for a user/pair, lazily seeding
// a fresh learner's vocabulary so the queue is never empty.
func (s *SRSQueueService) GetQueue(ctx context.Context, userID, nativeLang, targetLang string, limit int) (*models.SRSQueue, error) {
	targetLang = normalizeLang(targetLang)
	nativeLang = normalizeLang(nativeLang)
	if nativeLang == "" {
		nativeLang = "en"
	}
	if limit <= 0 {
		limit = queueDefaultLimit
	}
	if limit > queueMaxLimit {
		limit = queueMaxLimit
	}

	seeded, err := s.seed.EnsureSeeded(ctx, userID, nativeLang, targetLang)
	if err != nil {
		return nil, err
	}

	dueCards, err := s.practice.GetDueCards(ctx, userID, targetLang, limit)
	if err != nil {
		return nil, err
	}
	newCards, err := s.practice.GetNewCards(ctx, userID, targetLang, limit)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	vocabItems := make([]models.SRSQueueItem, 0, len(dueCards)+len(newCards))
	for _, c := range dueCards {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		vocabItems = append(vocabItems, s.vocabItem(&c, true))
	}
	for _, c := range newCards {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		vocabItems = append(vocabItems, s.vocabItem(&c, false))
	}

	grammar, err := s.grammarItems(ctx, userID, targetLang, grammarClozeLimit)
	if err != nil {
		return nil, err
	}

	items := interleaveQueue(vocabItems, grammar, limit)

	return &models.SRSQueue{
		TargetLanguage: targetLang,
		TotalDue:       s.countDue(ctx, userID, targetLang),
		TotalNew:       s.countNew(ctx, userID, targetLang),
		Seeded:         seeded,
		Items:          items,
	}, nil
}

func (s *SRSQueueService) vocabItem(c *VocabularyCard, due bool) models.SRSQueueItem {
	stage := nextStageForCard(c)
	_, q := s.practice.BuildVocabQuestion(c, stage)
	return models.SRSQueueItem{
		ID:         c.ID,
		Origin:     originOfCard(c),
		ItemType:   "vocabulary",
		Due:        due,
		NextReview: c.NextReview,
		Answer:     c.Term,
		Question:   q,
	}
}

// grammarItems returns due grammar-point cloze items (personal items mined from
// the GrammarService via user_grammar_mastery scheduling).
func (s *SRSQueueService) grammarItems(ctx context.Context, userID, targetLang string, limit int) ([]models.SRSQueueItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id::text, g.title, g.short_explanation, COALESCE(g.examples::text,'[]'), m.next_review_at
		FROM user_grammar_mastery m
		JOIN grammar_points g ON g.id = m.grammar_point_id
		WHERE m.user_id = $1 AND m.target_language = $2 AND m.next_review_at <= CURRENT_TIMESTAMP
		ORDER BY m.confidence ASC, m.next_review_at ASC
		LIMIT $3`, userID, targetLang, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.SRSQueueItem{}
	for rows.Next() {
		var id, title, expl, examples string
		var nextReview time.Time
		if err := rows.Scan(&id, &title, &expl, &examples, &nextReview); err != nil {
			return nil, err
		}
		q := buildGrammarClozeQuestion(title, expl, examples)
		if q == nil {
			continue
		}
		items = append(items, models.SRSQueueItem{
			ID:         id,
			Origin:     models.QueueOriginGrammar,
			ItemType:   "grammar",
			Due:        true,
			NextReview: nextReview,
			Answer:     q.answer,
			Question:   q.question,
		})
	}
	return items, rows.Err()
}

func (s *SRSQueueService) countDue(ctx context.Context, userID, targetLang string) int {
	var n int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND next_review <= CURRENT_TIMESTAMP`,
		userID, targetLang).Scan(&n)
	return n
}

func (s *SRSQueueService) countNew(ctx context.Context, userID, targetLang string) int {
	var n int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND mastery_stage < 2`,
		userID, targetLang).Scan(&n)
	return n
}

// originOfCard tags a vocabulary card as seed (curriculum-materialized) or
// personal (mined from chats/scenarios/manual saves).
func originOfCard(c *VocabularyCard) string {
	if c.SourceType == seedSourceType {
		return models.QueueOriginSeed
	}
	return models.QueueOriginPersonal
}

// interleaveQueue spreads grammar items through the vocabulary backlog so the
// learner doesn't sit through a long run of vocabulary cards, and caps the
// result at limit.
func interleaveQueue(vocab, grammar []models.SRSQueueItem, limit int) []models.SRSQueueItem {
	out := make([]models.SRSQueueItem, 0, len(vocab)+len(grammar))
	freq := 1
	if len(vocab) > 0 && len(grammar) > 0 {
		freq = maxInt(1, len(vocab)/len(grammar))
	}
	vi, gi := 0, 0
	for vi < len(vocab) || gi < len(grammar) {
		if vi >= len(vocab) {
			out = append(out, grammar[gi])
			gi++
			continue
		}
		if gi >= len(grammar) {
			out = append(out, vocab[vi])
			vi++
			continue
		}
		if vi == 0 || vi%freq == 0 {
			out = append(out, grammar[gi])
			gi++
		}
		out = append(out, vocab[vi])
		vi++
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// grammarClozeQuestion is a buildable grammar-cloze item plus its expected
// blanked answer.
type grammarClozeQuestion struct {
	question models.SessionQuestion
	answer   string
}

// buildGrammarClozeQuestion turns a grammar point's first example into a
// fill-in-the-blank cloze (a content word is blanked; the learner types it).
func buildGrammarClozeQuestion(title, explanation, examplesJSON string) *grammarClozeQuestion {
	var examples []string
	_ = json.Unmarshal([]byte(examplesJSON), &examples)
	if len(examples) == 0 {
		return nil
	}
	sentence := strings.TrimSpace(examples[0])
	blank, cloze := blankClozeWord(sentence)
	if blank == "" || cloze == "" {
		return nil
	}
	return &grammarClozeQuestion{
		question: models.SessionQuestion{
			ItemType:     "grammar",
			ActivityType: "grammar_cloze",
			PromptType:   "cued_recall",
			Prompt: models.SessionPrompt{
				Text:        cloze,
				Source:      title,
				GrammarHint: explanation,
			},
		},
		answer: blank,
	}
}

// blankClozeWord blanks the first content word (non-stopword, alphabetic, >=3
// chars) in a sentence and returns the expected blank plus the cloze text.
func blankClozeWord(sentence string) (blank, cloze string) {
	words := strings.Fields(sentence)
	if len(words) == 0 {
		return "", ""
	}
	idx := -1
	for i, w := range words {
		trimmed := strings.Trim(w, ".,;!?¿¡\"'")
		if len([]rune(trimmed)) >= 3 && !isStopword(trimmed, "") {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = len(words) - 1
	}
	blank = strings.Trim(words[idx], ".,;!?¿¡\"'()")
	for i, w := range words {
		if i == idx {
			words[i] = "______"
			_ = w
		}
	}
	return blank, strings.Join(words, " ")
}
