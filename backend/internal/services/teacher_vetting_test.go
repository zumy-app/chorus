package services

import "testing"

func TestTeacherVetting_RubricWeightsSum(t *testing.T) {
	sum := RubricWeightsSum()
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("weights sum %v != 1.0", sum)
	}
}

func TestTeacherVetting_ScoreRubric(t *testing.T) {
	rs := RubricScore{Scores: map[string]int{"pronunciation": 4, "fluency": 4, "pedagogy": 5, "engagement": 4, "accuracy": 4, "conduct": 5}}
	w, err := ScoreRubric(rs)
	if err != nil {
		t.Fatal(err)
	}
	if w < 4.3 || w > 4.4 {
		t.Fatalf("weighted %v out of expected", w)
	}
	_, err = ScoreRubric(RubricScore{Scores: map[string]int{"pronunciation": 4}})
	if err == nil {
		t.Fatal("expected missing criterion error")
	}
}

func TestTeacherVetting_Decide(t *testing.T) {
	pass := RubricScore{Scores: map[string]int{"pronunciation": 4, "fluency": 4, "pedagogy": 4, "engagement": 4, "accuracy": 4, "conduct": 4}}
	d, err := Decide(pass, true)
	if err != nil {
		t.Fatal(err)
	}
	if d.Decision != VettingApproved {
		t.Fatalf("expected approved got %s", d.Decision)
	}
	fail := RubricScore{Scores: map[string]int{"pronunciation": 2, "fluency": 4, "pedagogy": 4, "engagement": 4, "accuracy": 4, "conduct": 4}}
	d2, _ := Decide(fail, true)
	if d2.Decision != VettingRejected {
		t.Fatalf("expected rejected for hard fail, got %s", d2.Decision)
	}
	noPed := RubricScore{Scores: map[string]int{"pronunciation": 4, "fluency": 4, "pedagogy": 3, "engagement": 4, "accuracy": 4, "conduct": 4}}
	d3, _ := Decide(noPed, true)
	if d3.Decision == VettingApproved {
		t.Fatal("should not approve without pedagogy 4+")
	}
	d4, _ := Decide(pass, false)
	if d4.Decision != VettingNeedsWork {
		t.Fatalf("certs incomplete should be needs_work, got %s", d4.Decision)
	}
}

func TestTeacherVetting_ValidateRecording(t *testing.T) {
	if err := ValidateRecording(RecordingMeta{DurationSec: 120, SizeBytes: 10 << 20, MimeType: "video/mp4", AudioRMS: 0.05}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecording(RecordingMeta{DurationSec: 30}); err == nil {
		t.Fatal("expected duration error")
	}
}

func TestTeacherVetting_ValidateCertificates(t *testing.T) {
	ok, _ := ValidateCertificates([]Certificate{{Type: CertTeachingDegree, Issuer: "Uni", Year: 2020, FileURL: "https://cdn/c.pdf"}})
	if !ok {
		t.Fatal("expected cert ok")
	}
	ok2, _ := ValidateCertificates([]Certificate{{Type: CertOther, Issuer: "X", Year: 2020, FileURL: "https://cdn/c.pdf"}})
	if ok2 {
		t.Fatal("other alone should not pass")
	}
}

func TestTeacherVetting_ValidateApplication(t *testing.T) {
	if err := ValidateApplication(TeacherApplication{Bio: "hi", Languages: []string{"es"}, RateCents: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateApplication(TeacherApplication{Bio: "", Languages: []string{"es"}, RateCents: 1000}); err == nil {
		t.Fatal("expected bio error")
	}
	if !ValidateAssessment(70) || ValidateAssessment(69.9) {
		t.Fatal("assessment threshold wrong")
	}
}
