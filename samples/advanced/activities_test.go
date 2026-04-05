package main

import (
	"context"
	"testing"
)

func TestSendEmail(t *testing.T) {
	out, err := SendEmail(context.Background(), SendEmailInput{
		To:      "user@example.com",
		Subject: "Test",
		Body:    "Hello",
		Cc:      []string{"cc@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Recipients != 2 {
		t.Errorf("Recipients = %d, want 2", out.Recipients)
	}
	if out.Status != "sent" {
		t.Errorf("Status = %q, want %q", out.Status, "sent")
	}
}

func TestSendEmailMissingTo(t *testing.T) {
	_, err := SendEmail(context.Background(), SendEmailInput{Subject: "X", Body: "Y"})
	if err == nil {
		t.Fatal("expected error for missing To")
	}
}

func TestSendSMS(t *testing.T) {
	out, err := SendSMS(context.Background(), SendSMSInput{
		PhoneNumber: "+1234567890",
		Message:     "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Segments != 1 {
		t.Errorf("Segments = %d, want 1", out.Segments)
	}
}

func TestSendSlack(t *testing.T) {
	out, err := SendSlack(context.Background(), SendSlackInput{
		Channel: "general",
		Message: "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Channel != "#general" {
		t.Errorf("Channel = %q, want %q", out.Channel, "#general")
	}
	if !out.OK {
		t.Error("OK should be true")
	}
}

func TestSendWebhook(t *testing.T) {
	out, err := SendWebhook(context.Background(), SendWebhookInput{
		URL: "https://httpbin.org/post",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", out.StatusCode)
	}
}

func TestLogNotification(t *testing.T) {
	err := LogNotification(context.Background(), LogNotificationInput{
		Message: "test",
		Level:   "info",
	})
	if err != nil {
		t.Fatal(err)
	}
}
