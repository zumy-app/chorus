package services

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

// ---------------------------------------------------------------------------
// PracticeService — answer grading + depth-of-processing SRS
// ---------------------------------------------------------------------------

func TestGradeAnswer_CuedRecall(t *testing.T) {
	s := &PracticeService{}
	cases := []struct {
		correct, answer, prompt string
		wantCorrect             bool
		wantQuality             int
	}{
		{"estoy", "estoy", "cloze", true, 4},
		{"estoy", "ESTOY", "recognition", true, 4},
		{"estoy", "no sé", "cued_recall", false, 1},
		{"estoy", "", "cloze", false, 1},
		{"", "estoy", "cloze", false, 0},
	}
	for _, c := range cases {
		ok, q := s.GradeAnswer(c.correct, c.answer, c.prompt)
		if ok != c.wantCorrect || q != c.wantQuality {
			t.Errorf("GradeAnswer(%q,%q,%q)=(_ %t,%d), want (%t,%d)", c.correct, c.answer, c.prompt, ok, q, c.wantCorrect, c.wantQuality)
		}
	}
}

func TestGradeAnswer_FreeRecallNearMiss(t *testing.T) {
	s := &PracticeService{}
	// "voy a" vs "boy a" is a one-edit near-miss -> accepted at quality 3.
	ok, q := s.GradeAnswer("voy a", "boy a", "free_recall")
	if !ok || q != 3 {
		t.Errorf("expected near-miss quality 3, got (%t,%d)", ok, q)
	}
	ok, q = s.GradeAnswer("estoy", "estas", "production")
	if ok || q != 1 {
		t.Errorf("expected wrong, got (%t,%d)", ok, q)
	}
}

func TestNormalizeAnswer_FoldsAccentsAndPunctuation(t *testing.T) {
	cases := map[string]string{
		"  Estoy  ":       "estoy",
		"¿Dónde?":         "donde",
		"café con leche,": "cafe con leche",
		"adiós":           "adios",
		"  Me   llamo":    "me llamo",
	}
	for in, want := range cases {
		if got := normalizeAnswer(in); got != want {
			t.Errorf("normalizeAnswer(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestWordDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"voy", "boy", 1},
		{"casa", "casa", 0},
		{"perro", "gato", 4},
	}
	for _, c := range cases {
		if got := wordDistance(c.a, c.b); got != c.want {
			t.Errorf("wordDistance(%q,%q)=%d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSRSHelpers(t *testing.T) {
	if requiredSuccesses(1) != 1 || requiredSuccesses(2) != 1 || requiredSuccesses(3) != 2 || requiredSuccesses(4) != 2 {
		t.Errorf("requiredSuccesses unexpected")
	}
	if easeDelta(5) != 0.15 || easeDelta(4) != 0.05 || easeDelta(3) != -0.05 || easeDelta(0) != -0.20 {
		t.Errorf("easeDelta unexpected")
	}
	if promptTypeForStage(stageRecognition) != "recognition" || promptTypeForStage(stageCuedRecall) != "cued_recall" || promptTypeForStage(stageFreeRecall) != "free_recall" || promptTypeForStage(stageProduction) != "production" {
		t.Errorf("promptTypeForStage unexpected")
	}
	if got := clamp(1.3, 3.0, 0.5); got != 1.3 {
		t.Errorf("clamp lo: %v", got)
	}
	if got := clamp(1.3, 3.0, 4.0); got != 3.0 {
		t.Errorf("clamp hi: %v", got)
	}
	// nextInterval: stage 1->2, 2->3, 3->5, 4 first->7.
	if nextInterval(1, 2.5, 1) != 2 || nextInterval(1, 2.5, 2) != 3 || nextInterval(1, 2.5, 3) != 5 || nextInterval(1, 2.5, 4) != 7 {
		t.Errorf("nextInterval unexpected")
	}
	if learningOrNew(&models.VocabularyCard{MasteryStage: 1, ReviewCount: 1}) != stateNew {
		t.Errorf("new card should be new state")
	}
	if learningOrNew(&models.VocabularyCard{MasteryStage: 2, ReviewCount: 3}) != stateLearning {
		t.Errorf("reviewed card should be learning")
	}
	if masteryStateOf(&models.VocabularyCard{MasteryStage: 1}) != stateNew {
		t.Errorf("stage1 should be new")
	}
	if masteryStateOf(&models.VocabularyCard{MasteryStage: 2}) != stateLearning {
		t.Errorf("stage2 should be learning")
	}
	if masteryStateOf(&models.VocabularyCard{MasteryStage: 3}) != stateReviewing {
		t.Errorf("stage3 should be reviewing")
	}
	if masteryStateOf(&models.VocabularyCard{MasteryStage: 4}) != stateMastered {
		t.Errorf("stage4 should be mastered")
	}
}

func TestUpdateVocabAfterAttempt_CorrectAdvancesStage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec(`UPDATE vocabulary SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	s := &PracticeService{db: db}
	card := &models.VocabularyCard{ID: "c1", UserID: "u1", MasteryStage: 1, ReviewCount: 0, CorrectCount: 0, IntervalDays: 1, EaseFactor: 2.5}
	if err := s.UpdateVocabAfterAttempt(context.Background(), card, stageRecognition, 5, "recognition"); err != nil {
		t.Fatalf("UpdateVocabAfterAttempt: %v", err)
	}
	if card.MasteryStage != 2 {
		t.Errorf("expected stage advance to 2, got %d", card.MasteryStage)
	}
	if card.ReviewCount != 1 || card.CorrectCount != 1 {
		t.Errorf("review/correct counts wrong: %d/%d", card.ReviewCount, card.CorrectCount)
	}
	if card.IntervalDays != 2 {
		t.Errorf("expected interval 2 (stage1), got %v", card.IntervalDays)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateVocabAfterAttempt_WrongResetsInterval(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectExec(`UPDATE vocabulary SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	s := &PracticeService{db: db}
	card := &models.VocabularyCard{ID: "c1", UserID: "u1", MasteryStage: 3, ReviewCount: 4, StageSuccessCount: 1, IntervalDays: 10, EaseFactor: 2.5}
	if err := s.UpdateVocabAfterAttempt(context.Background(), card, stageFreeRecall, 1, "free_recall"); err != nil {
		t.Fatalf("UpdateVocabAfterAttempt: %v", err)
	}
	if card.IntervalDays != 1 {
		t.Errorf("wrong answer should reset interval to 1, got %v", card.IntervalDays)
	}
	if card.Lapses != 1 {
		t.Errorf("expected a lapse, got %d", card.Lapses)
	}
	// stage>=3 demotes one stage.
	if card.MasteryStage != 2 {
		t.Errorf("expected demotion to 2, got %d", card.MasteryStage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// WordMiningService — normalization, tokenization, teachability
// ---------------------------------------------------------------------------

func TestNormalizeLearningTerm(t *testing.T) {
	cases := map[string]string{
		"  Café con  Leche ": "cafe con leche",
		"¿Cómo estás?":       "como estas",
		"l'appareil":         "l'appareil",
		"¡Hola!":             "hola",
	}
	for in, want := range cases {
		if got := NormalizeLearningTerm(in, "es"); got != want {
			t.Errorf("NormalizeLearningTerm(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestTokenizeCandidates(t *testing.T) {
	toks := tokenizeCandidates("Hola, ¿cómo estás? — muy bien.")
	want := []string{"Hola", "cómo", "estás", "muy", "bien"}
	if len(toks) != len(want) {
		t.Fatalf("tokenize got %v, want %v", toks, want)
	}
	for i := range want {
		if toks[i] != want[i] {
			t.Errorf("token[%d]=%q, want %q", i, toks[i], want[i])
		}
	}
}

func TestContextSentence(t *testing.T) {
	long := "Hola, me llamo Ana y quiero pedir un café con leche porque tengo mucha sed en esta tarde calurosa en la ciudad."
	got := contextSentence(long, "café")
	if got == "" || !containsFold(got, "café") {
		t.Errorf("contextSentence should contain the term, got %q", got)
	}
	if contextSentence("", "x") != "" {
		t.Errorf("empty text should yield empty context")
	}
}

func TestIsStopword(t *testing.T) {
	if !isStopword("DE", "es") {
		t.Errorf("'de' should be a stopword")
	}
	if isStopword("café", "es") {
		t.Errorf("'café' should not be a stopword")
	}
}

func TestTeachability(t *testing.T) {
	s := &WordMiningService{}
	base := MinedCandidate{SurfaceText: "café", Confidence: 0.9}

	curr := s.teachability(base, "current_unit")
	bonus := s.teachability(base, "bonus")
	if curr < bonus {
		t.Errorf("current_unit (%v) should outrank bonus (%v)", curr, bonus)
	}
	if curr > 100 || curr < 0 {
		t.Errorf("teachability out of range: %v", curr)
	}

	proper := s.teachability(MinedCandidate{SurfaceText: "Ana", Confidence: 0.9, IsProperNoun: true}, "current_unit")
	if proper >= curr {
		t.Errorf("proper-noun penalty should lower the score: %v vs %v", proper, curr)
	}
	chunk := s.teachability(MinedCandidate{SurfaceText: "café leche", Confidence: 0.9, IsChunk: true}, "current_unit")
	nonChunk := s.teachability(MinedCandidate{SurfaceText: "café leche", Confidence: 0.9, IsChunk: false}, "current_unit")
	if chunk <= nonChunk {
		t.Errorf("chunk bonus should raise the score: %v vs %v", chunk, nonChunk)
	}
}

// ---------------------------------------------------------------------------
// SessionComposerService — interleaving + scoring helpers
// ---------------------------------------------------------------------------

func TestInterleaveItems_CombinesVocabAndOther(t *testing.T) {
	items := []sessionComposedItem{
		{itemType: "vocabulary", activityType: "v1"},
		{itemType: "grammar", activityType: "g1"},
		{itemType: "vocabulary", activityType: "v2"},
		{itemType: "lesson_step", activityType: "l1"},
		{itemType: "vocabulary", activityType: "v3"},
	}
	out := interleaveItems(items)
	if len(out) != len(items) {
		t.Fatalf("interleave dropped items: %d != %d", len(out), len(items))
	}
	// Preserve the multiset of item types.
	before := map[string]int{}
	after := map[string]int{}
	for _, it := range items {
		before[it.activityType]++
	}
	for _, it := range out {
		after[it.activityType]++
	}
	if len(before) != len(after) {
		t.Fatalf("interleave lost item types: %v vs %v", before, after)
	}
	// Non-vocabulary items must be present (interleaved / not dropped).
	foundOther := false
	for _, it := range out {
		if it.itemType != "vocabulary" {
			foundOther = true
		}
	}
	if !foundOther {
		t.Fatalf("no non-vocabulary item interleaved: %v", out)
	}
}

func TestSessionScoringHelpers(t *testing.T) {
	if scaleScore(5) != 200 || scaleScore(3) != 150 || scaleScore(0) != 0 {
		t.Errorf("scaleScore unexpected")
	}
	if feedbackForQuality(true, "") != "Correct! Nice work." {
		t.Errorf("positive feedback wrong")
	}
	if feedbackForQuality(false, "") == "Correct! Nice work." {
		t.Errorf("negative feedback wrong")
	}
	if stepFeedback(true) == stepFeedback(false) {
		t.Errorf("step feedback should differ")
	}
	if intFromAny(3.7) != 3 || intFromAny("x") != 0 {
		t.Errorf("intFromAny wrong")
	}
	if strFromAny("hi") != "hi" || strFromAny(42) != "" {
		t.Errorf("strFromAny wrong")
	}
	if maxInt(2, 7) != 7 {
		t.Errorf("maxInt wrong")
	}
}

func TestNextStageForCard(t *testing.T) {
	if nextStageForCard(&models.VocabularyCard{MasteryStage: 0}) != stageRecognition {
		t.Errorf("fresh card should start at recognition")
	}
	if nextStageForCard(&models.VocabularyCard{MasteryStage: 2}) != 2 {
		t.Errorf("mid card stays at its stage")
	}
	if nextStageForCard(&models.VocabularyCard{MasteryStage: 5}) != stageProduction {
		t.Errorf("mastered card practices production")
	}
}

// ---------------------------------------------------------------------------
// ScenarioService — intent detection + coverage
// ---------------------------------------------------------------------------

func TestDetectIntents(t *testing.T) {
	cases := []struct {
		msg, intent string
		want        bool
	}{
		{"Hola, buenos días.", "greet", true},
		{"Quisiera un café con leche, por favor.", "order_drink", true},
		{"Para llevar, por favor.", "customize", true},
		{"¿Cuánto cuesta?", "pay", true},
		{"Gracias.", "close", true},
		{"Tengo un perro.", "order_drink", false},
	}
	all := []string{"greet", "order_drink", "customize", "pay", "close"}
	for _, c := range cases {
		hits := detectIntents(c.msg, all)
		got := containsStr(hits, c.intent)
		if got != c.want {
			t.Errorf("detectIntents(%q) got %v, intent %q present=%v want %v", c.msg, hits, c.intent, got, c.want)
		}
	}
}

func TestIntentsCovered(t *testing.T) {
	if !intentsCovered([]string{"greet", "order_drink"}, []string{"greet", "order_drink"}) {
		t.Errorf("all covered should pass")
	}
	if intentsCovered([]string{"greet"}, []string{"greet", "order_drink"}) {
		t.Errorf("missing intent should fail")
	}
	if !intentsCovered(nil, nil) {
		t.Errorf("no required intents always covered")
	}
}

func TestMergeStrings_Dedupes(t *testing.T) {
	out := mergeStrings([]string{"a", "b"}, []string{"b", "c"})
	if len(out) != 3 {
		t.Errorf("merge should dedupe, got %v", out)
	}
}

// ---------------------------------------------------------------------------
// LessonService — step grading
// ---------------------------------------------------------------------------

func TestGradeStepAnswer(t *testing.T) {
	// accepted list match
	ok, q, _ := gradeStepAnswer("cloze", []byte(`{"accepted":["estoy"],"correct":"estoy"}`), "estoy")
	if !ok || q != 4 {
		t.Errorf("accepted answer should grade correct/4, got (%t,%d)", ok, q)
	}
	// wrong
	ok, q, _ = gradeStepAnswer("cloze", []byte(`{"accepted":["estoy"],"correct":"estoy"}`), "soy")
	if ok || q != 1 {
		t.Errorf("wrong answer should grade incorrect/1, got (%t,%d)", ok, q)
	}
	// free_form always accepted
	ok, q, _ = gradeStepAnswer("production", []byte(`{"free_form":true}`), "anything")
	if !ok || q != 4 {
		t.Errorf("free_form should be accepted, got (%t,%d)", ok, q)
	}
}

// helpers ----------------------------------------------------------------//

func containsFold(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func containsStr(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
