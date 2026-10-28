package main

import (
	"testing"
	"time"
)

func TestDeadlineReminderDecision(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	l := learner{Email: "learner@example.edu", Course: "Algebra", Deadline: now.Add(24 * time.Hour)}
	tests := []struct {
		name       string
		suppressed bool
		wantAction string
		wantReason string
	}{
		{"hard bounce is reported and never sent", true, "suppress", "recipient is on the suppression list"},
		{"active learner receives a reminder", false, "send", "deadline reminder is due"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideDelivery(l, tt.suppressed, now)
			if got.Action != tt.wantAction || got.Reason != tt.wantReason {
				t.Fatalf("got %#v", got)
			}
		})
	}
}
