package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ariyam/api-monitor/internal/models"
)

// fakeFinder and fakeStore are hand-written stand-ins for the database.
//
// This is why the scheduler depends on the MonitorFinder and ResultStore
// INTERFACES rather than on the repository: these two small types let every
// test below run in milliseconds with no PostgreSQL anywhere.
//
// No mocking framework, no code generation - about forty lines of ordinary Go.

type fakeFinder struct {
	mu       sync.Mutex
	monitors []models.Monitor
	err      error
	calls    int
}

func (f *fakeFinder) FindDue(ctx context.Context, limit int) ([]models.Monitor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.monitors, nil
}

type recorded struct {
	monitorID  uuid.UUID
	result     CheckResult
	transition Transition
}

type fakeStore struct {
	mu sync.Mutex
	// Every write is captured so tests can assert on what the scheduler
	// decided, without a database.
	records []recorded
	err     error

	// concurrent tracks how many RecordResult calls overlap, which is how the
	// concurrency-limit test measures the semaphore.
	concurrent    int
	maxConcurrent int
}

func (f *fakeStore) RecordResult(ctx context.Context, monitorID uuid.UUID, result CheckResult, transition Transition) error {
	f.mu.Lock()
	f.concurrent++
	if f.concurrent > f.maxConcurrent {
		f.maxConcurrent = f.concurrent
	}
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.concurrent--
		f.mu.Unlock()
	}()

	if f.err != nil {
		return f.err
	}

	f.mu.Lock()
	f.records = append(f.records, recorded{monitorID, result, transition})
	f.mu.Unlock()
	return nil
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.records)
}

func testMonitor(url string, status models.MonitorStatus) models.Monitor {
	return models.Monitor{
		ID:              uuid.New(),
		Name:            "Test",
		URL:             url,
		Method:          http.MethodGet,
		ExpectedStatus:  200,
		TimeoutSeconds:  5,
		IntervalSeconds: 60,
		Enabled:         true,
		Status:          status,
	}
}

func TestScheduler_ChecksAndRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusPending),
	}}
	store := &fakeStore{}

	s := NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))})
	s.runOnce(context.Background())

	if store.count() != 1 {
		t.Fatalf("recorded %d results, want 1", store.count())
	}

	rec := store.records[0]
	if !rec.result.Success {
		t.Errorf("expected a successful check, got: %s", rec.result.Error)
	}
	if rec.transition.NewStatus != models.StatusUp {
		t.Errorf("status = %q, want up", rec.transition.NewStatus)
	}
	if rec.transition.OpenIncident {
		t.Error("a healthy check must not open an incident")
	}
}

func TestScheduler_FailureOpensIncident(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusUp), // was healthy
	}}
	store := &fakeStore{}

	NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).runOnce(context.Background())

	rec := store.records[0]
	if rec.transition.NewStatus != models.StatusDown {
		t.Errorf("status = %q, want down", rec.transition.NewStatus)
	}
	if !rec.transition.OpenIncident {
		t.Error("going from up to failing must OPEN an incident")
	}
}

func TestScheduler_StillDownDoesNotOpenAnother(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusDown), // ALREADY down
	}}
	store := &fakeStore{}

	NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).runOnce(context.Background())

	if store.records[0].transition.OpenIncident {
		t.Error("a monitor that was already down must not open a SECOND incident")
	}
}

func TestScheduler_RecoveryClosesIncident(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusDown),
	}}
	store := &fakeStore{}

	NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).runOnce(context.Background())

	rec := store.records[0]
	if !rec.transition.CloseIncident {
		t.Error("recovering from down must CLOSE the open incident")
	}
	if rec.transition.NewStatus != models.StatusUp {
		t.Errorf("status = %q, want up", rec.transition.NewStatus)
	}
}

// The semaphore must genuinely cap simultaneous checks.
func TestScheduler_RespectsConcurrencyLimit(t *testing.T) {
	// A slow server, so many checks are in flight at once.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	monitors := make([]models.Monitor, 20)
	for i := range monitors {
		monitors[i] = testMonitor(server.URL, models.StatusUp)
	}

	store := &fakeStore{}
	s := NewScheduler(&fakeFinder{monitors: monitors}, store,
		SchedulerConfig{Tick: time.Hour, Concurrency: 3, Checker: NewChecker(WithSSRFProtection(false))})

	s.runOnce(context.Background())

	if store.count() != 20 {
		t.Errorf("recorded %d of 20 monitors", store.count())
	}
	// The limit is on CHECKS. Writes happen after each check completes, so
	// overlapping writes can never exceed the number of concurrent checkers.
	if store.maxConcurrent > 3 {
		t.Errorf("saw %d concurrent writes with a limit of 3", store.maxConcurrent)
	}
}

// A database failure must not stop the worker.
func TestScheduler_SurvivesFinderError(t *testing.T) {
	finder := &fakeFinder{err: errors.New("database is on fire")}
	store := &fakeStore{}

	// The real assertion is that this does not panic or hang.
	NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).
		runOnce(context.Background())

	if store.count() != 0 {
		t.Error("nothing should have been recorded")
	}
}

// A failure while SAVING must not stop the other monitors being checked.
func TestScheduler_SurvivesStoreError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusUp),
		testMonitor(server.URL, models.StatusUp),
	}}
	store := &fakeStore{err: errors.New("write failed")}

	NewScheduler(finder, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).
		runOnce(context.Background())
	// Reaching here without a panic is the assertion.
}

// Run must return promptly when its context is cancelled.
func TestScheduler_StopsOnContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	finder := &fakeFinder{monitors: []models.Monitor{
		testMonitor(server.URL, models.StatusUp),
	}}
	s := NewScheduler(finder, &fakeStore{}, SchedulerConfig{Tick: 50 * time.Millisecond, Checker: NewChecker(WithSSRFProtection(false))})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done) // only reached when Run returns
	}()

	time.Sleep(120 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Run returned, as it should.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop within 2s of cancellation")
	}
}

// An empty result set must be handled without fuss.
func TestScheduler_NothingDue(t *testing.T) {
	store := &fakeStore{}
	NewScheduler(&fakeFinder{monitors: nil}, store, SchedulerConfig{Tick: time.Hour, Checker: NewChecker(WithSSRFProtection(false))}).
		runOnce(context.Background())

	if store.count() != 0 {
		t.Error("nothing was due, so nothing should have been recorded")
	}
}
