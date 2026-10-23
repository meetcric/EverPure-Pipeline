package events

import (
	"testing"

	"pipelinepulse/internal/model"
)

func TestPublishReachesOnlyThatRun(t *testing.T) {
	b := NewBroker()
	r1, unsub1 := b.Subscribe("run-1")
	r2, unsub2 := b.Subscribe("run-2")
	defer unsub1()
	defer unsub2()

	b.Publish("run-1", model.StatusRunning)

	if got := <-r1; got != model.StatusRunning {
		t.Errorf("run-1 got %q, want running", got)
	}
	select {
	case got := <-r2:
		t.Errorf("run-2 must not get run-1's event, got %q", got)
	default:
	}
}

// The harness opens and closes the run page 50 times: every unsubscribe must
// leave the broker as empty as it started.
func TestUnsubscribeLeavesNothingBehind(t *testing.T) {
	b := NewBroker()
	for i := 0; i < 50; i++ {
		_, unsub := b.Subscribe("run-1")
		unsub()
		unsub() // calling it twice is safe
	}
	if len(b.subs) != 0 {
		t.Fatalf("broker still holds %d run entries", len(b.subs))
	}
	b.Publish("run-1", model.StatusPassed) // no subscribers: must not block or panic
}
