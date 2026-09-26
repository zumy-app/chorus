package services

import (
	"errors"
	"math"
	"strings"
)

const (
	VettingPending          = "pending"
	VettingAssessmentPassed = "assessment_passed"
	VettingRecordingUploaded = "recording_uploaded"
	VettingCertsVerified    = "certs_verified"
	VettingReviewScheduled  = "review_scheduled"
	VettingReviewed         = "reviewed"
	VettingApproved         = "approved"
	VettingRejected         = "rejected"
	VettingNeedsWork        = "needs_work"
)

const (
	CertTeachingDegree     = "teaching_degree"
	CertLanguageCertificate = "language_certificate"
	CertOther              = "other"
)

var RubricWeights = map[string]float64{
	"pronunciation": 0.22,
	"fluency":       0.18,
	"pedagogy":      0.22,
	"engagement":    0.14,
	"accuracy":      0.14,
	"conduct":       0.10,
}

var RubricCriteria = []string{"pronunciation", "fluency", "pedagogy", "engagement", "accuracy", "conduct"}

type TeacherApplication struct {
	ID              string
	UserID          string
	Languages       []string
	Bio             string
	RateCents       int
	VideoURL        string
	AssessmentScore float64
	Status          string
}

type RecordingMeta struct {
	DurationSec float64
	SizeBytes   int64
	MimeType    string
	AudioRMS    float64
}

type Certificate struct {
	Type   string
	Issuer string
	Year   int
	FileURL string
}

type RubricScore struct {
	Scores map[string]int
	Notes  string
}

type VettingDecision struct {
	Decision   string
	Weighted   float64
	NeedsSecondReview bool
	Reasons    []string
}

func ValidateApplication(app TeacherApplication) error {
	if strings.TrimSpace(app.Bio) == "" {
		return errors.New("bio required")
	}
	if len(app.Languages) == 0 {
		return errors.New("at least one language required")
	}
	if app.RateCents <= 0 {
		return errors.New("rate required")
	}
	return nil
}

func ValidateAssessment(score float64) bool {
	return score >= 70
}

func ValidateRecording(m RecordingMeta) error {
	if m.DurationSec < 90 || m.DurationSec > 210 {
		return errors.New("recording duration must be 90-210s")
	}
	if m.SizeBytes > 100*1024*1024 {
		return errors.New("recording too large")
	}
	if m.MimeType != "" && m.MimeType != "video/mp4" && m.MimeType != "video/webm" && m.MimeType != "audio/mp4" && m.MimeType != "audio/webm" {
		return errors.New("unsupported mime")
	}
	if m.AudioRMS != 0 && m.AudioRMS < 0.01 {
		return errors.New("audio too quiet")
	}
	return nil
}

func ValidateCertificates(certs []Certificate) (bool, []string) {
	reasons := []string{}
	hasTeachingOrC1 := false
	for _, c := range certs {
		if c.Type == CertTeachingDegree || c.Type == CertLanguageCertificate {
			hasTeachingOrC1 = true
		}
		if strings.TrimSpace(c.Issuer) == "" || c.Year < 1900 || c.Year > 2030 {
			reasons = append(reasons, "invalid certificate fields")
		}
		if strings.TrimSpace(c.FileURL) == "" {
			reasons = append(reasons, "certificate file required")
		}
	}
	if !hasTeachingOrC1 {
		reasons = append(reasons, "at least one teaching or language certificate required")
	}
	return len(reasons) == 0 && hasTeachingOrC1, reasons
}

func ScoreRubric(rs RubricScore) (float64, error) {
	if len(rs.Scores) == 0 {
		return 0, errors.New("empty rubric")
	}
	sum := 0.0
	for _, crit := range RubricCriteria {
		v, ok := rs.Scores[crit]
		if !ok {
			return 0, errors.New("missing criterion: " + crit)
		}
		if v < 1 || v > 5 {
			return 0, errors.New("score out of range 1-5: " + crit)
		}
		sum += float64(v) * RubricWeights[crit]
	}
	return math.Round(sum*100) / 100, nil
}

func Decide(rs RubricScore, certsOK bool) (VettingDecision, error) {
	weighted, err := ScoreRubric(rs)
	if err != nil {
		return VettingDecision{}, err
	}
	reasons := []string{}
	hasHardFail := false
	for _, crit := range RubricCriteria {
		if rs.Scores[crit] <= 2 {
			hasHardFail = true
			reasons = append(reasons, crit+" <=2")
		}
	}
	hasPedagogy4 := rs.Scores["pedagogy"] >= 4
	needsSecond := weighted >= 3.4 && weighted < 3.7
	decision := VettingRejected
	switch {
	case !certsOK:
		decision = VettingNeedsWork
		reasons = append(reasons, "certs incomplete")
	case hasHardFail:
		decision = VettingRejected
	case weighted >= 3.6 && hasPedagogy4:
		decision = VettingApproved
	case weighted >= 3.4:
		decision = VettingNeedsWork
	default:
		decision = VettingRejected
	}
	if decision == VettingRejected && len(reasons) == 0 {
		reasons = append(reasons, "weighted below threshold")
	}
	return VettingDecision{Decision: decision, Weighted: weighted, NeedsSecondReview: needsSecond, Reasons: reasons}, nil
}

func RubricWeightsSum() float64 {
	s := 0.0
	for _, w := range RubricWeights {
		s += w
	}
	return s
}
