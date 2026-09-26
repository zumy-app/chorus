package services

import (
	"encoding/json"
	"testing"

	"github.com/chorus/messenger/internal/models"
)

// TestInterleaveQueue_Smoke verifies the unified queue interleaves grammar items
// through the vocabulary backlog and never drops or duplicates items, while
// respecting the limit.
func TestInterleaveQueue_Smoke(t *testing.T) {
	vocab := []models.SRSQueueItem{
		{ID: "v1"}, {ID: "v2"}, {ID: "v3"}, {ID: "v4"}, {ID: "v5"},
	}
	grammar := []models.SRSQueueItem{{ID: "g1"}, {ID: "g2"}}

	out := interleaveQueue(vocab, grammar, 20)
	if len(out) != len(vocab)+len(grammar) {
		t.Fatalf("interleave dropped items: got %d want %d", len(out), len(vocab)+len(grammar))
	}
	// Both grammar items must be present.
	counts := map[string]int{}
	for _, it := range out {
		counts[it.ID]++
	}
	for _, id := range []string{"v1", "v2", "v3", "v4", "v5", "g1", "g2"} {
		if counts[id] != 1 {
			t.Errorf("expected exactly one %q, got %d", id, counts[id])
		}
	}
}

func TestInterleaveQueue_RespectsLimit(t *testing.T) {
	vocab := make([]models.SRSQueueItem, 10)
	for i := range vocab {
		vocab[i] = models.SRSQueueItem{ID: "v"}
	}
	grammar := []models.SRSQueueItem{{ID: "g"}}
	out := interleaveQueue(vocab, grammar, 5)
	if len(out) != 5 {
		t.Fatalf("expected cap 5, got %d", len(out))
	}
}

func TestInterleaveQueue_GrammarOnly(t *testing.T) {
	grammar := []models.SRSQueueItem{{ID: "g1"}, {ID: "g2"}}
	out := interleaveQueue(nil, grammar, 10)
	if len(out) != 2 {
		t.Fatalf("expected 2 grammar items, got %d", len(out))
	}
}

// TestBlankClozeWord verifies a content word is blanked and returned as the
// expected answer, leaving the rest of the sentence intact with a placeholder.
func TestBlankClozeWord(t *testing.T) {
	blank, cloze := blankClozeWord("Quisiera un café con leche, por favor.")
	if blank == "" {
		t.Fatalf("expected a non-empty blanked answer")
	}
	if !containsSub(cloze, "______") {
		t.Errorf("cloze should contain a blank placeholder, got %q", cloze)
	}
	if containsSub(cloze, blank) {
		t.Errorf("cloze should not contain the full answer %q: %q", blank, cloze)
	}
}

func TestBlankClozeWord_Empty(t *testing.T) {
	blank, cloze := blankClozeWord("")
	if blank != "" || cloze != "" {
		t.Errorf("empty sentence should yield empty cloze, got %q / %q", blank, cloze)
	}
}

// TestBuildGrammarClozeQuestion verifies a grammar point's example becomes a
// gradeable cloze with an expected answer.
func TestBuildGrammarClozeQuestion(t *testing.T) {
	examples, _ := json.Marshal([]string{"Quisiera un café, por favor."})
	q := buildGrammarClozeQuestion("Querer, quisiera", "Use 'quisiera' to ask politely.", string(examples))
	if q == nil {
		t.Fatal("expected a cloze question from examples")
	}
	if q.answer == "" {
		t.Fatalf("expected a cloze answer, got empty")
	}
	if q.question.ItemType != "grammar" || q.question.ActivityType != "grammar_cloze" {
		t.Errorf("unexpected item metadata: %+v", q.question)
	}
	if q.question.Prompt.Source != "Querer, quisiera" {
		t.Errorf("expected grammar title as source, got %q", q.question.Prompt.Source)
	}
}

func TestBuildGrammarClozeQuestion_NoExamples(t *testing.T) {
	if q := buildGrammarClozeQuestion("title", "", "[]"); q != nil {
		t.Errorf("expected nil for a grammar point without examples, got %+v", q)
	}
}

// TestOriginOfCard verifies seed vs personal origin tagging.
func TestOriginOfCard(t *testing.T) {
	if got := originOfCard(&models.VocabularyCard{SourceType: seedSourceType}); got != models.QueueOriginSeed {
		t.Errorf("seed card should be tagged seed, got %q", got)
	}
	if got := originOfCard(&models.VocabularyCard{SourceType: "chat"}); got != models.QueueOriginPersonal {
		t.Errorf("chat-mined card should be tagged personal, got %q", got)
	}
	if got := originOfCard(&models.VocabularyCard{SourceType: "manual"}); got != models.QueueOriginPersonal {
		t.Errorf("manual card should be tagged personal, got %q", got)
	}
}

// helpers ----------------------------------------------------------------//

func containsSub(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
