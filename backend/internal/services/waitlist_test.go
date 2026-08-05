package services

import (
	"testing"

	"github.com/chorus/messenger/internal/models"
)

func TestValidateWaitlistRequestRequiresInterestFields(t *testing.T) {
	err := ValidateWaitlistRequest(models.WaitlistRequest{
		Email:           "learner@example.com",
		SpokenLanguage:  "en",
		TargetLanguages: []string{"es"},
		Reasons:         []string{"travel"},
	})
	if err != nil {
		t.Fatalf("expected valid waitlist request, got %v", err)
	}

	err = ValidateWaitlistRequest(models.WaitlistRequest{Email: "learner@example.com"})
	if err == nil {
		t.Fatal("expected missing interest fields to be rejected")
	}
}
