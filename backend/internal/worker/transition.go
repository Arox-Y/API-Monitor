package worker

import "github.com/ariyam/api-monitor/internal/models"

// Transition is what should happen to the database after a check.
//
// Deciding this is kept as a PURE FUNCTION - no database, no clock, no
// network. Given the same inputs it always returns the same answer, which
// makes the most important logic in the product trivial to test exhaustively.
//
// The database work that follows is mechanical; the thinking is all here.
type Transition struct {
	NewStatus models.MonitorStatus

	// OpenIncident is true only when a monitor has just STARTED failing.
	OpenIncident bool

	// CloseIncident is true only when a monitor has just RECOVERED.
	CloseIncident bool
}

// DecideTransition works out what a check result means for a monitor.
//
// The whole product hinges on one idea:
//
//	INCIDENTS ARE CREATED BY TRANSITIONS, NOT BY STATES.
//
// Only the moment a monitor goes from healthy to failing opens an incident.
// Staying broken does not open another. A site down for an hour and checked
// every minute produces ONE incident, not sixty.
//
// Getting this wrong is the classic bug in a monitoring tool: an incident per
// failed check floods the history and makes downtime impossible to measure.
//
//	| previous | result  | outcome                               |
//	|----------|---------|---------------------------------------|
//	| pending  | success | -> up                                 |
//	| pending  | failure | -> down, OPEN an incident             |
//	| up       | success | -> up (nothing else)                  |
//	| up       | failure | -> down, OPEN an incident             |
//	| down     | failure | -> down (NO second incident)          |
//	| down     | success | -> up, CLOSE the open incident        |
//
// Note that `pending` (never checked) behaves like `up`: a monitor that fails
// its very first check is genuinely down, and deserves an incident.
func DecideTransition(previous models.MonitorStatus, success bool) Transition {
	if success {
		return Transition{
			NewStatus: models.StatusUp,
			// Close an incident only if one was open, i.e. we were down.
			CloseIncident: previous == models.StatusDown,
		}
	}

	return Transition{
		NewStatus: models.StatusDown,
		// Open one only if we were NOT already down. This single condition is
		// what stops sixty incidents for one outage.
		OpenIncident: previous != models.StatusDown,
	}
}
