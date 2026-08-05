package services

import (
	"os"
	"strconv"
	"testing"
)

func TestSMTPEmailSenderRejectsMissingMailuConfiguration(t *testing.T) {
	sender := NewSMTPEmailSender("", 587, "", "", "")
	if err := sender.Send("learner@example.com", "Welcome", "<p>Welcome</p>"); err == nil {
		t.Fatal("expected SMTP sender to reject missing Mailu configuration")
	}
}

func TestSMTPEmailSenderLiveDelivery(t *testing.T) {
	if os.Getenv("MAILU_SMTP_LIVE_TEST") != "1" {
		t.Skip("set MAILU_SMTP_LIVE_TEST=1 to send a Mailu delivery test")
	}
	port, err := strconv.Atoi(os.Getenv("MAILU_SMTP_PORT"))
	if err != nil {
		t.Fatal("MAILU_SMTP_PORT must be set for the live test")
	}
	from := os.Getenv("MAILU_SMTP_FROM")
	sender := NewSMTPEmailSender(
		os.Getenv("MAILU_SMTP_HOST"), port, os.Getenv("MAILU_SMTP_USERNAME"),
		os.Getenv("MAILU_SMTP_PASSWORD"), from,
	)
	if err := sender.Send(from, "Chorus SMTP delivery test", "<p>Mailu SMTP is configured for Chorus.</p>"); err != nil {
		t.Fatalf("Mailu SMTP delivery failed: %v", err)
	}
}
