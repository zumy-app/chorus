package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

func ValidateCaptionRating(rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	return nil
}

func ValidateCaptionCorrection(text string) error {
	if len([]rune(text)) > 5000 {
		return fmt.Errorf("corrected text too long")
	}
	return nil
}

type CaptionReviewService struct {
	db *sql.DB
}

func NewCaptionReviewService(db *sql.DB) *CaptionReviewService {
	return &CaptionReviewService{db: db}
}

func (s *CaptionReviewService) SubmitReview(ctx context.Context, callID string, segmentIndex int, reviewerID string, req models.CaptionReviewRequest) (*models.CaptionReview, error) {
	if err := ValidateCaptionRating(req.Rating); err != nil {
		return nil, err
	}
	if err := ValidateCaptionCorrection(req.CorrectedText); err != nil {
		return nil, err
	}
	if len([]rune(req.Feedback)) > 2000 {
		return nil, fmt.Errorf("feedback too long")
	}
	var callExists string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM call_sessions WHERE id=$1`, callID).Scan(&callExists); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCallNotFound
		}
		return nil, err
	}
	var segmentsJSON []byte
	var originalText, originalLang, translatedText, targetLang string
	if err := s.db.QueryRowContext(ctx, `SELECT segments FROM call_transcripts WHERE call_id=$1`, callID).Scan(&segmentsJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transcript not found")
		}
		return nil, err
	}
	var segs []models.TranscriptSegment
	_ = jsonUnmarshal(segmentsJSON, &segs)
	if segmentIndex < 0 || segmentIndex >= len(segs) {
		return nil, fmt.Errorf("segment index out of range")
	}
	seg := segs[segmentIndex]
	originalText = seg.OriginalText
	originalLang = seg.OriginalLanguage
	if originalLang == "" {
		originalLang = "en"
	}
	targetLang = strings.TrimSpace(req.TargetLanguage)
	if targetLang == "" && len(seg.Translations) > 0 {
		for k, v := range seg.Translations {
			targetLang = k
			translatedText = v
			break
		}
	} else if targetLang != "" {
		if v, ok := seg.Translations[targetLang]; ok {
			translatedText = v
		}
	}
	if translatedText == "" && req.CorrectedText == "" {
		for _, v := range seg.Translations {
			translatedText = v
			break
		}
	}
	if translatedText == "" {
		translatedText = originalText
	}
	if targetLang == "" {
		targetLang = "en"
	}
	id := uuid.New().String()
	var r models.CaptionReview
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO caption_reviews (id, call_id, segment_index, original_text, original_language, translated_text, target_language, reviewer_id, rating, corrected_text, feedback)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (call_id, segment_index, reviewer_id) DO UPDATE SET rating=EXCLUDED.rating, corrected_text=EXCLUDED.corrected_text, feedback=EXCLUDED.feedback, translated_text=EXCLUDED.translated_text, target_language=EXCLUDED.target_language
		RETURNING id, call_id, segment_index, original_text, original_language, translated_text, target_language, reviewer_id, rating, corrected_text, feedback, created_at
	`, id, callID, segmentIndex, originalText, originalLang, translatedText, targetLang, reviewerID, req.Rating, strings.TrimSpace(req.CorrectedText), strings.TrimSpace(req.Feedback)).Scan(
		&r.ID, &r.CallID, &r.SegmentIndex, &r.OriginalText, &r.OriginalLanguage, &r.TranslatedText, &r.TargetLanguage, &r.ReviewerID, &r.Rating, &r.CorrectedText, &r.Feedback, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save review: %w", err)
	}
	return &r, nil
}

func (s *CaptionReviewService) GetReviews(ctx context.Context, callID string, segmentIndex int) ([]models.CaptionReview, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, call_id, segment_index, original_text, original_language, translated_text, target_language, reviewer_id, rating, corrected_text, feedback, created_at FROM caption_reviews WHERE call_id=$1 AND segment_index=$2 ORDER BY created_at DESC`, callID, segmentIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CaptionReview
	for rows.Next() {
		var r models.CaptionReview
		if err := rows.Scan(&r.ID, &r.CallID, &r.SegmentIndex, &r.OriginalText, &r.OriginalLanguage, &r.TranslatedText, &r.TargetLanguage, &r.ReviewerID, &r.Rating, &r.CorrectedText, &r.Feedback, &r.CreatedAt); err != nil {
			continue
		}
		var name sql.NullString
		_ = s.db.QueryRowContext(ctx, `SELECT display_name FROM users WHERE id=$1`, r.ReviewerID).Scan(&name)
		if name.Valid {
			r.ReviewerName = name.String
		}
		out = append(out, r)
	}
	if out == nil {
		out = []models.CaptionReview{}
	}
	return out, rows.Err()
}

func (s *CaptionReviewService) GetReviewQueue(ctx context.Context, reviewerID string, limit, offset int) ([]models.CaptionReviewQueueItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, call_id, segments FROM call_transcripts ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	type raw struct {
		callID string
		segs   []models.TranscriptSegment
	}
	var all []raw
	for rows.Next() {
		var id, callID string
		var j []byte
		if err := rows.Scan(&id, &callID, &j); err != nil {
			continue
		}
		var segs []models.TranscriptSegment
		_ = jsonUnmarshal(j, &segs)
		all = append(all, raw{callID: callID, segs: segs})
	}
	var items []models.CaptionReviewQueueItem
	for _, r := range all {
		for idx, seg := range r.segs {
			if strings.TrimSpace(seg.OriginalText) == "" {
				continue
			}
			items = append(items, models.CaptionReviewQueueItem{
				CallID: r.callID, SegmentIndex: idx, OriginalText: seg.OriginalText, OriginalLanguage: seg.OriginalLanguage, Translations: seg.Translations, SpeakerID: seg.SpeakerID, Confidence: seg.Confidence,
			})
		}
	}
	for i := range items {
		var cnt sql.NullInt64
		var avg sql.NullFloat64
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*), AVG(rating) FROM caption_reviews WHERE call_id=$1 AND segment_index=$2`, items[i].CallID, items[i].SegmentIndex).Scan(&cnt, &avg)
		if cnt.Valid {
			items[i].ReviewCount = int(cnt.Int64)
		}
		if avg.Valid {
			v := avg.Float64
			items[i].AvgRating = &v
		}
	}
	total := len(items)
	if offset >= total {
		return []models.CaptionReviewQueueItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func (s *CaptionReviewService) GetQualityStats(ctx context.Context) (*models.CaptionQualityStats, error) {
	var totalCaptions int
	rows, err := s.db.QueryContext(ctx, `SELECT segments FROM call_transcripts`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var j []byte
			_ = rows.Scan(&j)
			var segs []models.TranscriptSegment
			_ = jsonUnmarshal(j, &segs)
			totalCaptions += len(segs)
		}
	}
	var reviewedCount sql.NullInt64
	var avg sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*), AVG(rating) FROM caption_reviews`).Scan(&reviewedCount, &avg)
	stats := &models.CaptionQualityStats{TotalCaptions: totalCaptions, RatingCounts: map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}}
	if reviewedCount.Valid {
		stats.ReviewedCount = int(reviewedCount.Int64)
	}
	if avg.Valid {
		stats.AvgRating = avg.Float64
	}
	rcRows, err := s.db.QueryContext(ctx, `SELECT rating, COUNT(*) FROM caption_reviews GROUP BY rating`)
	if err == nil {
		defer rcRows.Close()
		for rcRows.Next() {
			var r, c int
			_ = rcRows.Scan(&r, &c)
			stats.RatingCounts[r] = c
		}
	}
	stats.PendingCount = totalCaptions - stats.ReviewedCount
	if stats.PendingCount < 0 {
		stats.PendingCount = 0
	}
	return stats, nil
}

func jsonUnmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
