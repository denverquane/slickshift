package bot

import (
	"testing"
	"time"
)

// Regression: triggering processing blocked when the queue was full, which hung the command/API handler that triggered it
func TestTriggerRedemptionProcessing_DoesNotBlockWhenFull(t *testing.T) {
	bot := &Bot{redemptionTrigger: make(chan string, 1)}

	done := make(chan struct{})
	go func() {
		bot.triggerRedemptionProcessing("1")
		bot.triggerRedemptionProcessing("2") // the queue is full
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Triggering processing blocked when the queue was full")
	}
	if userID := <-bot.redemptionTrigger; userID != "1" {
		t.Errorf("The queued trigger should be kept, got %q", userID)
	}
}
