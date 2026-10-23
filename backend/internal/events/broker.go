// Package events is an in-process pub/sub for run status changes: PATCH /runs/{id}
// publishes, and every open GET /runs/{id}/events stream for that run receives it.
// It only sees changes made by this process, which is enough for one API container
// (see NOTES.md for Postgres LISTEN/NOTIFY when scaling out).
package events

import (
	"sync"

	"pipelinepulse/internal/model"
)

// subscriberBuffer is how many updates a subscriber can fall behind before we drop one.
// A run changes status at most twice (queued → running → finished), so it never fills up.
const subscriberBuffer = 8

// Broker implements service.StatusPublisher and httpapi.RunEvents.
type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan model.Status]struct{} // runID → set of subscriber channels

	closeOnce sync.Once
	done      chan struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subs: make(map[string]map[chan model.Status]struct{}),
		done: make(chan struct{}),
	}
}

// Subscribe returns a channel of status changes for runID and a function that removes it.
// The caller must call unsubscribe when it stops listening, or the channel leaks.
func (b *Broker) Subscribe(runID string) (<-chan model.Status, func()) {
	ch := make(chan model.Status, subscriberBuffer)

	b.mu.Lock()
	if b.subs[runID] == nil {
		b.subs[runID] = make(map[chan model.Status]struct{})
	}
	b.subs[runID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs[runID], ch)
			if len(b.subs[runID]) == 0 {
				delete(b.subs, runID) // don't keep an empty entry for every run ever watched
			}
			b.mu.Unlock()
		})
	}
	return ch, unsubscribe
}

// Publish sends status to every subscriber of runID. It never blocks: a subscriber
// whose buffer is full misses this update rather than stalling the PATCH request.
func (b *Broker) Publish(runID string, status model.Status) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[runID] {
		select {
		case ch <- status:
		default:
		}
	}
}

// Done is closed when the server shuts down, so open streams can end.
func (b *Broker) Done() <-chan struct{} {
	return b.done
}

// Close ends every open stream. Safe to call more than once.
func (b *Broker) Close() {
	b.closeOnce.Do(func() { close(b.done) })
}
