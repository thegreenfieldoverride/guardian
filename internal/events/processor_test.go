package events

import (
	"testing"
)

// Coverage in this package stops here, and the reason is structural rather
// than an oversight.
//
// Processor holds two concrete collaborators that NewProcessor builds itself:
//
//	redisClient  *redis.Client   // NewProcessor pings it and fails if absent
//	triageEngine *ai.TriageEngine
//
// Neither is an interface, so there is no seam to substitute a fake. Every
// branch of ProcessEvent funnels into publishCollectiveStrategistEvent, which
// calls redisClient.XAdd directly. Testing the decision routing therefore needs
// a live Redis *and* a real AI provider, which makes it an integration test
// against two external services rather than a unit test of the routing.
//
// Making this package testable is a small refactor — accept interfaces for both
// collaborators and let NewProcessor construct the concrete ones — not more
// test-writing. Until that happens, the triage decision routing (auto-fix vs
// escalate vs ignore) is unverified, and that routing is what decides whether
// an incident reaches a human.

// generateEventID does not touch the receiver, so it can be exercised directly.
// The implementation is timestamp-based and its own comment concedes it is a
// placeholder ("in production, would use ULID or UUID"). Two events produced
// inside the same nanosecond tick would share an ID, and these IDs are used as
// correlation keys for events published downstream.
func TestGenerateEventIDIsUnique(t *testing.T) {
	p := &Processor{}

	const n = 10000
	seen := make(map[string]struct{}, n)

	for i := 0; i < n; i++ {
		id := p.generateEventID()
		if _, dup := seen[id]; dup {
			t.Fatalf("generateEventID returned a duplicate after %d calls: %q\n"+
				"IDs are timestamp-based (lg_<UnixNano>), so events created within the "+
				"same clock tick collide; these IDs are used as correlation keys", i+1, id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenerateEventIDHasExpectedPrefix(t *testing.T) {
	p := &Processor{}

	id := p.generateEventID()
	if len(id) < 4 || id[:3] != "lg_" {
		t.Errorf("generateEventID() = %q, want an lg_ prefix", id)
	}
}
