package services

import "testing"

func TestValidateCaptionRating(t *testing.T) {
	if err := ValidateCaptionRating(3); err != nil { t.Fatal(err) }
	if err := ValidateCaptionRating(0); err == nil { t.Fatal("expected error") }
	if err := ValidateCaptionRating(6); err == nil { t.Fatal("expected error") }
}
func TestValidateCaptionCorrection(t *testing.T) {
	if err := ValidateCaptionCorrection("ok"); err != nil { t.Fatal(err) }
	if err := ValidateCaptionCorrection(string(make([]rune, 6000))); err == nil { t.Fatal("expected error") }
}
