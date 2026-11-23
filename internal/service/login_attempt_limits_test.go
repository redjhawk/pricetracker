package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/internal/store"
)

// Login attempt limits by client address (issue #266, FR-LOGIN-LIM-001..011).

const (
	clientA       = "198.51.100.1"
	clientB       = "198.51.100.2"
	rightPassword = "correct horse battery"
	wrongPassword = "wrong password"
)

var limitStart = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

var limitAnswers = map[string]struct {
	status  int
	message string
}{
	"INVALID_CREDENTIALS": {401, "Incorrect username or password."},
	"LOGIN_LOCKED":        {429, "Too many failed attempts. Wait one minute and try again."},
	"LOGIN_BUSY":          {429, "Too many login attempts are being checked. Try again in a moment."},
}

// limitClock is a goroutine-safe clock for service.now.
type limitClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *limitClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *limitClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

// newLimitService returns a service with users alice and bob and a clock at limitStart.
func newLimitService(t *testing.T) (*Service, *limitClock) {
	t.Helper()
	service, _ := newSessionService(t)
	clock := &limitClock{now: limitStart}
	service.now = clock.Now
	for _, username := range []string{"alice", "bob"} {
		if _, err := service.CreateUser(context.Background(), username, rightPassword); err != nil {
			t.Fatal(err)
		}
	}
	return service, clock
}

type loginResult struct {
	token string
	user  store.User
	err   error
}

func login(service *Service, ctx context.Context, client, username, password string) loginResult {
	token, user, err := service.Login(ctx, client, username, password)
	return loginResult{token, user, err}
}

func expectLoggedIn(t *testing.T, result loginResult, username, what string) {
	t.Helper()
	if result.err != nil || result.token == "" || result.user.Username != username {
		t.Fatalf("%s: want a login as %s, got token %q user %+v error %v", what, username, result.token, result.user, result.err)
	}
}

func expectRefused(t *testing.T, result loginResult, code, what string) {
	t.Helper()
	want := limitAnswers[code]
	var typed *Error
	if !errors.As(result.err, &typed) || typed.Code != code || typed.Status != want.status || typed.Message != want.message || result.token != "" {
		t.Fatalf("%s: want %d %s %q without a token, got token %q error %#v", what, want.status, code, want.message, result.token, result.err)
	}
}

// expectLogin logs in synchronously and fails unless it succeeds.
func expectLogin(t *testing.T, service *Service, client, username, password, what string) {
	t.Helper()
	expectLoggedIn(t, login(service, context.Background(), client, username, password), username, what)
}

// expectLoginError logs in synchronously and fails unless the error has the code.
func expectLoginError(t *testing.T, service *Service, client, username, password, code, what string) {
	t.Helper()
	expectRefused(t, login(service, context.Background(), client, username, password), code, what)
}

// failLogins sends count wrong passwords, each answered with wrong credentials.
func failLogins(t *testing.T, service *Service, client, username string, count int) {
	t.Helper()
	for attempt := 1; attempt <= count; attempt++ {
		expectLoginError(t, service, client, username, wrongPassword, "INVALID_CREDENTIALS", fmt.Sprintf("%s/%s wrong password %d", client, username, attempt))
	}
}

func failureEntry(service *Service, key loginKey) (loginFailures, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	entry, ok := service.loginFailures[key]
	return entry, ok
}

func failureCount(service *Service) int {
	service.mu.Lock()
	defer service.mu.Unlock()
	return len(service.loginFailures)
}

// checkHold holds the first password check of a login running until release, so that
// further logins must wait for the password-check slot.
type checkHold struct {
	service  *Service
	entered  chan struct{}
	release  chan struct{}
	released sync.Once
	running  sync.WaitGroup
}

func newCheckHold(t *testing.T, service *Service) *checkHold {
	t.Helper()
	hold := &checkHold{service: service, entered: make(chan struct{}), release: make(chan struct{})}
	var first atomic.Bool
	service.beforePasswordCheck = func(context.Context) {
		if first.CompareAndSwap(false, true) {
			close(hold.entered)
			<-hold.release
		}
	}
	// Runs before the service and database are closed: every started call must end first.
	t.Cleanup(func() {
		hold.Release()
		hold.running.Wait()
	})
	return hold
}

func (h *checkHold) Release() { h.released.Do(func() { close(h.release) }) }

// run calls work in a goroutine that the cleanup waits for.
func (h *checkHold) run(work func()) {
	h.running.Add(1)
	go func() {
		defer h.running.Done()
		work()
	}()
}

// start begins a login in a goroutine; its result arrives on the returned channel.
func (h *checkHold) start(ctx context.Context, client, username, password string) <-chan loginResult {
	results := make(chan loginResult, 1)
	h.run(func() { results <- login(h.service, ctx, client, username, password) })
	return results
}

// holdRunning starts a login and returns once its password check is held running.
func (h *checkHold) holdRunning(t *testing.T, client, username, password string) <-chan loginResult {
	t.Helper()
	results := h.start(context.Background(), client, username, password)
	select {
	case <-h.entered:
	case result := <-results:
		t.Fatalf("the password check was not held (BeforePasswordCheck not called): %+v", result)
	case <-time.After(5 * time.Second):
		t.Fatal("the password check was not held (BeforePasswordCheck not called)")
	}
	return results
}

// waitForWaiting waits until count logins wait for the password-check slot.
func waitForWaiting(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for WaitingLoginChecks() != count {
		if time.Now().After(deadline) {
			t.Fatalf("%d logins waiting for a password check, want %d", WaitingLoginChecks(), count)
		}
		time.Sleep(time.Millisecond)
	}
}

// atOnce returns the result of a call that must not wait for the held password checks.
func atOnce(t *testing.T, results <-chan loginResult, what string) loginResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(2 * time.Second):
		t.Fatalf("%s: did not return while password checks are held", what)
		return loginResult{}
	}
}

// stillWaiting fails if the call has already returned.
func stillWaiting(t *testing.T, results <-chan loginResult, what string) {
	t.Helper()
	select {
	case result := <-results:
		t.Fatalf("%s: returned while the password check is held: %+v", what, result)
	case <-time.After(100 * time.Millisecond):
	}
}

// finished returns the result of a call once the held checks were released.
func finished(t *testing.T, results <-chan loginResult, what string) loginResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(10 * time.Second):
		t.Fatalf("%s: did not finish after the release", what)
		return loginResult{}
	}
}

func waitingClient(index int) string { return fmt.Sprintf("203.0.113.%d", 10+index) }

func TestLoginLimits_TC001_FailuresAreCountedPerPairOnly(t *testing.T) {
	t.Run("other client does not add to the count", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		failLogins(t, service, clientB, "alice", 1)
		expectLogin(t, service, clientA, "alice", rightPassword, "A after 4 own failures and 1 of B")
		// B's single failure does not combine with A's: B needs 4 more to be locked.
		failLogins(t, service, clientA, "alice", 4)
		failLogins(t, service, clientB, "alice", 4)
		expectLoginError(t, service, clientB, "alice", rightPassword, "LOGIN_LOCKED", "B after 5 failures")
		expectLogin(t, service, clientA, "alice", rightPassword, "A while B is locked")
	})
	t.Run("username letter case is the same pair", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "Alice", 3)
		failLogins(t, service, clientA, "alice", 2)
		expectLoginError(t, service, clientA, "ALICE", rightPassword, "LOGIN_LOCKED", "A for ALICE")
	})
	t.Run("unknown username", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "nobody", 5)
		expectLoginError(t, service, clientA, "nobody", wrongPassword, "LOGIN_LOCKED", "6th attempt for nobody")
		expectLogin(t, service, clientA, "alice", rightPassword, "A as alice")
	})
}

func TestLoginLimits_TC002_FiveFailuresLockThePairForOneMinute(t *testing.T) {
	t.Run("lock lasts one minute and refusals do not extend it", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 5)
		for _, offset := range []time.Duration{0, 30 * time.Second, 45 * time.Second, 59 * time.Second} {
			clock.Set(limitStart.Add(offset))
			expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", fmt.Sprintf("right password at T+%s", offset))
			expectLoginError(t, service, clientA, "alice", wrongPassword, "LOGIN_LOCKED", fmt.Sprintf("wrong password at T+%s", offset))
		}
		clock.Set(limitStart.Add(time.Minute))
		expectLogin(t, service, clientA, "alice", rightPassword, "right password at T+1min")
	})
	t.Run("after the lock five new failures are needed", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 5)
		clock.Set(limitStart.Add(time.Minute))
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "after 4 new failures")
	})
	t.Run("five new failures lock again", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 5)
		clock.Set(limitStart.Add(time.Minute))
		failLogins(t, service, clientA, "alice", 5)
		expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", "after 5 new failures")
	})
	t.Run("no lock at four failures", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "after 4 failures")
	})
}

func TestLoginLimits_TC003_LockedPairDoesNotAffectOthers(t *testing.T) {
	service, _ := newLimitService(t)
	ctx := context.Background()
	failLogins(t, service, clientB, "alice", 4)
	failLogins(t, service, clientA, "alice", 5)
	expectLogin(t, service, clientB, "alice", rightPassword, "B for alice (5th attempt after 4 failures)")
	failLogins(t, service, clientA, "bob", 4)
	expectLogin(t, service, clientA, "bob", rightPassword, "A for bob")
	expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", "A for alice")

	adminPassword, _, err := service.ResetAdminPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}
	failLogins(t, service, clientA, "admin", 5)
	expectLoginError(t, service, clientA, "admin", adminPassword, "LOGIN_LOCKED", "A for admin")
	expectLogin(t, service, clientB, "admin", adminPassword, "B for admin")
}

func TestLoginLimits_TC004_LockedAttemptIsRefusedWithoutWaiting(t *testing.T) {
	service, _ := newLimitService(t)
	ctx := context.Background()
	failLogins(t, service, clientA, "alice", 5)
	failLogins(t, service, clientA, "nobody", 5)
	hold := newCheckHold(t, service)
	held := hold.holdRunning(t, clientB, "alice", rightPassword)
	expectRefused(t, atOnce(t, hold.start(ctx, clientA, "alice", rightPassword), "locked with a held check"), "LOGIN_LOCKED", "locked with a held check")

	var waiting []<-chan loginResult
	for index := 0; index < maxWaitingLoginChecks-1; index++ {
		waiting = append(waiting, hold.start(ctx, waitingClient(index), "bob", rightPassword))
	}
	waitForWaiting(t, maxWaitingLoginChecks-1)
	expectRefused(t, atOnce(t, hold.start(ctx, clientA, "alice", rightPassword), "locked with Max-1 waiting"), "LOGIN_LOCKED", "locked with Max-1 waiting")
	// The locked attempt took no waiting place: one more login is accepted to wait.
	last := hold.start(ctx, waitingClient(maxWaitingLoginChecks-1), "bob", rightPassword)
	waitForWaiting(t, maxWaitingLoginChecks)
	stillWaiting(t, last, "unlocked login after the locked refusal")
	waiting = append(waiting, last)
	// The lock is checked before the waiting places.
	expectRefused(t, atOnce(t, hold.start(ctx, clientA, "alice", rightPassword), "locked with Max waiting"), "LOGIN_LOCKED", "locked with Max waiting")
	expectRefused(t, atOnce(t, hold.start(ctx, clientA, "nobody", rightPassword), "locked unknown username"), "LOGIN_LOCKED", "locked unknown username")
	if count := WaitingLoginChecks(); count != maxWaitingLoginChecks {
		t.Fatalf("%d logins waiting after the locked refusals, want %d", count, maxWaitingLoginChecks)
	}

	hold.Release()
	expectLoggedIn(t, finished(t, held, "held login"), "alice", "held login of B")
	for index, results := range waiting {
		expectLoggedIn(t, finished(t, results, "waiting login"), "bob", fmt.Sprintf("waiting login %d", index))
	}
}

func TestLoginLimits_TC005_SuccessRestartsThePairCountOnly(t *testing.T) {
	t.Run("other client keeps its count", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		failLogins(t, service, clientB, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "A after 4 failures")
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "A after 4 new failures")
		failLogins(t, service, clientB, "alice", 1)
		expectLoginError(t, service, clientB, "alice", rightPassword, "LOGIN_LOCKED", "B after its 5th failure")
	})
	t.Run("repeated success", func(t *testing.T) {
		service, _ := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "first success")
		expectLogin(t, service, clientA, "alice", rightPassword, "second success")
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "after 4 failures")
	})
}

func TestLoginLimits_TC006_OldFailuresAndEndedLocksAreForgotten(t *testing.T) {
	t.Run("failures older than one hour are forgotten", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		clock.Set(limitStart.Add(time.Hour + time.Second))
		failLogins(t, service, clientA, "alice", 4)
		expectLogin(t, service, clientA, "alice", rightPassword, "after 4 old and 4 new failures")
	})
	t.Run("a failure within the hour keeps the earlier ones", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 4)
		clock.Set(limitStart.Add(59 * time.Minute))
		failLogins(t, service, clientA, "alice", 1)
		expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", "5th failure at T+59min")
	})
	t.Run("memory is bounded", func(t *testing.T) {
		service, clock := newLimitService(t)
		for index := 0; index < 1000; index++ {
			client := fmt.Sprintf("10.0.%d.%d", index/256, index%256)
			expectLoginError(t, service, client, "alice", wrongPassword, "INVALID_CREDENTIALS", client)
		}
		failLogins(t, service, clientA, "alice", 5)
		if count := failureCount(service); count != 1001 {
			t.Fatalf("%d remembered pairs, want 1001", count)
		}
		clock.Set(limitStart.Add(time.Hour + time.Second))
		expectLoginError(t, service, clientB, "bob", wrongPassword, "INVALID_CREDENTIALS", "new pair")
		if count := failureCount(service); count != 1 {
			t.Fatalf("%d remembered pairs after pruning, want 1", count)
		}
		if entry, ok := failureEntry(service, loginKey{clientB, "bob"}); !ok || entry.count != 1 {
			t.Fatalf("new pair %+v %v", entry, ok)
		}
	})
	t.Run("pruning keeps an active lock", func(t *testing.T) {
		service, clock := newLimitService(t)
		failLogins(t, service, clientA, "alice", 5)
		clock.Set(limitStart.Add(30 * time.Second))
		failLogins(t, service, clientB, "bob", 1)
		expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", "A after another pair's failure")
	})
}

func TestLoginLimits_TC007_WaitingAttemptIsCheckedWhenItsPairBecomesLocked(t *testing.T) {
	for _, test := range []struct {
		name, password string
	}{{"wrong password", wrongPassword}, {"right password", rightPassword}} {
		t.Run(test.name, func(t *testing.T) {
			service, clock := newLimitService(t)
			hold := newCheckHold(t, service)
			held := hold.holdRunning(t, clientB, "bob", rightPassword)
			waiting := hold.start(context.Background(), clientA, "alice", test.password)
			waitForWaiting(t, 1)
			key := loginKey{clientA, "alice"}
			for attempt := 0; attempt < maxFailedLogins; attempt++ {
				service.recordLoginFailure(key, limitStart)
			}
			entry, _ := failureEntry(service, key)
			lockedUntil := entry.lockedUntil
			if !lockedUntil.Equal(limitStart.Add(time.Minute)) {
				t.Fatalf("lock until %v, want %v", lockedUntil, limitStart.Add(time.Minute))
			}
			clock.Set(limitStart.Add(10 * time.Second))
			hold.Release()
			expectLoggedIn(t, finished(t, held, "held login"), "bob", "held login")
			result := finished(t, waiting, "waiting login")
			if test.password == wrongPassword {
				expectRefused(t, result, "INVALID_CREDENTIALS", "waiting wrong password")
				if entry, _ := failureEntry(service, key); !entry.lockedUntil.Equal(lockedUntil) {
					t.Fatalf("lock until %v after the waiting failure, want %v", entry.lockedUntil, lockedUntil)
				}
				expectLoginError(t, service, clientA, "alice", rightPassword, "LOGIN_LOCKED", "A after the waiting failure")
				return
			}
			expectLoggedIn(t, result, "alice", "waiting right password")
			if entry, ok := failureEntry(service, key); ok {
				t.Fatalf("failures kept after the success: %+v", entry)
			}
			expectLogin(t, service, clientA, "alice", rightPassword, "A after the waiting success")
		})
	}
}

func TestLoginLimits_TC008_CancelledWaitingLoginIsNotCounted(t *testing.T) {
	service, _ := newLimitService(t)
	failLogins(t, service, clientA, "alice", 4)
	hold := newCheckHold(t, service)
	held := hold.holdRunning(t, clientB, "bob", rightPassword)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiting := hold.start(ctx, clientA, "alice", wrongPassword)
	waitForWaiting(t, 1)
	cancel()
	result := atOnce(t, waiting, "cancelled login")
	if !errors.Is(result.err, context.Canceled) || errorCode(result.err) != "" || result.token != "" {
		t.Fatalf("cancelled login: token %q error %v, want the cancellation error", result.token, result.err)
	}
	hold.Release()
	expectLoggedIn(t, finished(t, held, "held login"), "bob", "held login")
	expectLogin(t, service, clientA, "alice", rightPassword, "A after 4 failures and a cancelled login")
}

func TestLoginLimits_TC011_BusyWhenMaxChecksAreWaiting(t *testing.T) {
	service, _ := newLimitService(t)
	ctx := context.Background()
	failLogins(t, service, clientA, "alice", 4)
	hold := newCheckHold(t, service)
	held := hold.holdRunning(t, clientB, "bob", rightPassword)
	var waiting []<-chan loginResult
	for index := 0; index < maxWaitingLoginChecks-1; index++ {
		waiting = append(waiting, hold.start(ctx, waitingClient(index), "bob", rightPassword))
	}
	waitForWaiting(t, maxWaitingLoginChecks-1)
	// Boundary: with Max - 1 waiting, one more login is accepted to wait.
	last := hold.start(ctx, waitingClient(maxWaitingLoginChecks-1), "bob", rightPassword)
	waitForWaiting(t, maxWaitingLoginChecks)
	stillWaiting(t, last, "login with Max-1 waiting")
	waiting = append(waiting, last)

	for attempt := 1; attempt <= 2; attempt++ {
		what := fmt.Sprintf("A busy attempt %d", attempt)
		expectRefused(t, atOnce(t, hold.start(ctx, clientA, "alice", rightPassword), what), "LOGIN_BUSY", what)
	}
	expectRefused(t, atOnce(t, hold.start(ctx, clientA, "nobody", rightPassword), "unknown username"), "LOGIN_BUSY", "unknown username")
	// User creation is not a login: it keeps waiting and is not refused busy.
	created := make(chan error, 1)
	hold.run(func() {
		_, err := service.CreateUser(ctx, "carol", rightPassword)
		created <- err
	})
	select {
	case err := <-created:
		t.Fatalf("user creation returned while the password check is held: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	hold.Release()
	expectLoggedIn(t, finished(t, held, "held login"), "bob", "held login")
	for index, results := range waiting {
		expectLoggedIn(t, finished(t, results, "waiting login"), "bob", fmt.Sprintf("waiting login %d", index))
	}
	select {
	case err := <-created:
		if err != nil {
			t.Fatalf("user creation after the release: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("user creation did not finish after the release")
	}
	// 4 failures + 2 busy refusals would have locked the pair if busy refusals were counted.
	expectLogin(t, service, clientA, "alice", rightPassword, "A after the busy refusals")
}

func TestLoginLimits_TC012_LoginWaitsWhileAPlaceIsFree(t *testing.T) {
	for _, test := range []struct {
		name, password string
	}{{"right password", rightPassword}, {"wrong password", wrongPassword}} {
		t.Run(test.name, func(t *testing.T) {
			service, _ := newLimitService(t)
			hold := newCheckHold(t, service)
			held := hold.holdRunning(t, clientA, "bob", rightPassword)
			waiting := hold.start(context.Background(), clientB, "alice", test.password)
			waitForWaiting(t, 1)
			stillWaiting(t, waiting, "B's login")
			hold.Release()
			expectLoggedIn(t, finished(t, held, "held login"), "bob", "held login")
			result := finished(t, waiting, "B's login")
			if test.password == rightPassword {
				expectLoggedIn(t, result, "alice", "B's login")
				return
			}
			expectRefused(t, result, "INVALID_CREDENTIALS", "B's login")
			if entry, ok := failureEntry(service, loginKey{clientB, "alice"}); !ok || entry.count != 1 {
				t.Fatalf("B/alice failures %+v %v, want 1", entry, ok)
			}
		})
	}
}
