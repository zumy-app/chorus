package handlers

import "testing"

// Auto grammar analysis must never break messaging: with no settings service
// configured the toggle reads as enabled (fail-open), and with no learning
// profile service no user matches (fail-closed, no spurious jobs).
func TestGrammarAutoEnabled_DefaultTrueWhenNoSettings(t *testing.T) {
	h := NewMessageHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if !h.grammarAutoEnabled("user-1") {
		t.Fatalf("expected grammar auto-analysis enabled by default when settings service is nil")
	}
}

func TestTargetLanguageMatches_FalseWhenNoProfileService(t *testing.T) {
	h := NewMessageHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if h.targetLanguageMatches(t.Context(), "user-1", "es") {
		t.Fatalf("expected no match when learning profile service is nil")
	}
}

func TestSetGrammarQueue_AttachesWithoutPanic(t *testing.T) {
	h := NewMessageHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.SetGrammarQueue(nil)
	if h.grammarQueue != nil {
		t.Fatalf("expected nil grammar queue after SetGrammarQueue(nil)")
	}
}
