package worker

import (
	"testing"

	"github.com/ariyam/api-monitor/internal/models"
)

// This is the highest-value test in the project.
//
// It is pure logic, so every case can be covered exhaustively in microseconds,
// and the behaviour it guards - one incident per outage - is what makes the
// incident history meaningful.
func TestDecideTransition(t *testing.T) {
	tests := []struct {
		name       string
		previous   models.MonitorStatus
		success    bool
		wantStatus models.MonitorStatus
		wantOpen   bool
		wantClose  bool
		why        string
	}{
		{
			name:     "first check ever succeeds",
			previous: models.StatusPending, success: true,
			wantStatus: models.StatusUp, wantOpen: false, wantClose: false,
			why: "nothing was wrong, so there is nothing to open or close",
		},
		{
			name:     "first check ever fails",
			previous: models.StatusPending, success: false,
			wantStatus: models.StatusDown, wantOpen: true, wantClose: false,
			why: "a monitor that fails immediately is genuinely down",
		},
		{
			name:     "healthy stays healthy",
			previous: models.StatusUp, success: true,
			wantStatus: models.StatusUp, wantOpen: false, wantClose: false,
			why: "the common case - do nothing but record the check",
		},
		{
			name:     "healthy starts failing",
			previous: models.StatusUp, success: false,
			wantStatus: models.StatusDown, wantOpen: true, wantClose: false,
			why: "THE transition that opens an incident",
		},
		{
			name:     "already down, still failing",
			previous: models.StatusDown, success: false,
			wantStatus: models.StatusDown, wantOpen: false, wantClose: false,
			why: "MOST IMPORTANT CASE: no second incident for the same outage",
		},
		{
			name:     "down recovers",
			previous: models.StatusDown, success: true,
			wantStatus: models.StatusUp, wantOpen: false, wantClose: true,
			why: "close the incident that was opened when it went down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideTransition(tt.previous, tt.success)

			if got.NewStatus != tt.wantStatus {
				t.Errorf("status = %q, want %q (%s)", got.NewStatus, tt.wantStatus, tt.why)
			}
			if got.OpenIncident != tt.wantOpen {
				t.Errorf("OpenIncident = %v, want %v (%s)", got.OpenIncident, tt.wantOpen, tt.why)
			}
			if got.CloseIncident != tt.wantClose {
				t.Errorf("CloseIncident = %v, want %v (%s)", got.CloseIncident, tt.wantClose, tt.why)
			}
		})
	}
}

// Simulates a real outage minute by minute and counts the incidents.
//
// This is the behaviour a user actually experiences, and it is the one a
// per-check implementation gets wrong.
func TestDecideTransition_OneOutageProducesOneIncident(t *testing.T) {
	// A site that works, breaks for 10 checks, then recovers.
	results := []bool{
		true, true, true, // healthy
		false, false, false, false, false, // broken
		false, false, false, false, false, // still broken
		true, true, // recovered
	}

	status := models.StatusPending
	opened, closed := 0, 0

	for _, success := range results {
		transition := DecideTransition(status, success)
		if transition.OpenIncident {
			opened++
		}
		if transition.CloseIncident {
			closed++
		}
		status = transition.NewStatus
	}

	if opened != 1 {
		t.Errorf("opened %d incidents for ONE outage, want 1", opened)
	}
	if closed != 1 {
		t.Errorf("closed %d incidents, want 1", closed)
	}
	if status != models.StatusUp {
		t.Errorf("final status = %q, want up", status)
	}
}

// Two separate outages must produce two separate incidents.
func TestDecideTransition_TwoOutagesProduceTwoIncidents(t *testing.T) {
	results := []bool{
		true,
		false, false, // outage 1
		true, true, // recovered
		false, false, false, // outage 2
		true, // recovered
	}

	status := models.StatusPending
	opened, closed := 0, 0

	for _, success := range results {
		transition := DecideTransition(status, success)
		if transition.OpenIncident {
			opened++
		}
		if transition.CloseIncident {
			closed++
		}
		status = transition.NewStatus
	}

	if opened != 2 {
		t.Errorf("opened %d incidents for TWO outages, want 2", opened)
	}
	if closed != 2 {
		t.Errorf("closed %d incidents, want 2", closed)
	}
}

// An incident must never be opened and closed by the same check - that would
// mean the database is asked to do two contradictory things in one transaction.
func TestDecideTransition_NeverOpensAndClosesTogether(t *testing.T) {
	for _, previous := range []models.MonitorStatus{
		models.StatusPending, models.StatusUp, models.StatusDown,
	} {
		for _, success := range []bool{true, false} {
			transition := DecideTransition(previous, success)
			if transition.OpenIncident && transition.CloseIncident {
				t.Errorf("previous=%q success=%v opened AND closed at once",
					previous, success)
			}
		}
	}
}

// A flapping site alternates up and down. Every down opens exactly one
// incident and every recovery closes it - so opens and closes stay balanced.
//
// This also documents a KNOWN LIMITATION: with a one-check threshold, a site
// that flaps produces an incident every single time. Real products absorb this
// with an N-consecutive-failure rule. See ADR-008.
func TestDecideTransition_FlappingIsBalanced(t *testing.T) {
	status := models.StatusUp
	opened, closed := 0, 0

	for i := 0; i < 10; i++ {
		for _, success := range []bool{false, true} {
			transition := DecideTransition(status, success)
			if transition.OpenIncident {
				opened++
			}
			if transition.CloseIncident {
				closed++
			}
			status = transition.NewStatus
		}
	}

	if opened != 10 || closed != 10 {
		t.Errorf("opened %d, closed %d; want 10 and 10", opened, closed)
	}
}
