package ratelimit

import (
	"sync"
	"testing"
	"time"
)

type clock struct {
	mutex sync.Mutex
	now   time.Time
}

func (c *clock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *clock) advance(by time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = c.now.Add(by)
}

func testLimiter(limit int, window time.Duration) (*Limiter, *clock) {
	timepiece := &clock{now: time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)}
	return NewWithClock(Rule{Limit: limit, Window: window}, timepiece.Now), timepiece
}

func TestAllowsUpToTheLimitAndThenRefuses(t *testing.T) {
	limiter, _ := testLimiter(3, time.Minute)

	for attempt := 1; attempt <= 3; attempt++ {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatalf("attempt %d was refused inside the limit", attempt)
		}
	}
	allowed, wait := limiter.Allow("client")
	if allowed {
		t.Fatal("the fourth attempt was allowed past the limit")
	}
	if wait <= 0 || wait > time.Minute {
		t.Fatalf("retry after = %v", wait)
	}
}

func TestCountsEachClientSeparately(t *testing.T) {
	limiter, _ := testLimiter(1, time.Minute)

	if allowed, _ := limiter.Allow("first"); !allowed {
		t.Fatal("first client was refused")
	}
	// One client exhausting its budget must not lock out anyone else.
	if allowed, _ := limiter.Allow("second"); !allowed {
		t.Fatal("a second client was refused for the first one's attempts")
	}
	if allowed, _ := limiter.Allow("first"); allowed {
		t.Fatal("the first client was allowed past its limit")
	}
}

func TestFreesUpAsTheWindowSlides(t *testing.T) {
	limiter, timepiece := testLimiter(2, time.Minute)

	limiter.Allow("client")
	timepiece.advance(30 * time.Second)
	limiter.Allow("client")

	if allowed, _ := limiter.Allow("client"); allowed {
		t.Fatal("a third attempt inside the window was allowed")
	}

	// The window slides rather than resetting: 31 seconds later only the first
	// hit has fallen out, so exactly one attempt is free.
	timepiece.advance(31 * time.Second)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("the window did not free up after the oldest hit expired")
	}
	if allowed, _ := limiter.Allow("client"); allowed {
		t.Fatal("more than one attempt was freed")
	}
}

func TestReportsWhenTheNextAttemptFits(t *testing.T) {
	limiter, timepiece := testLimiter(1, time.Minute)
	limiter.Allow("client")
	timepiece.advance(20 * time.Second)

	_, wait := limiter.Allow("client")

	// 40 seconds remain of the window opened by the first hit.
	if wait != 40*time.Second {
		t.Fatalf("retry after = %v, want 40s", wait)
	}
}

func TestNeverAsksTheClientToRetryImmediately(t *testing.T) {
	limiter, timepiece := testLimiter(1, time.Minute)
	limiter.Allow("client")
	timepiece.advance(59*time.Second + 900*time.Millisecond)

	_, wait := limiter.Allow("client")

	// Rounding down to zero would invite an immediate retry that also fails.
	if wait < time.Second {
		t.Fatalf("retry after = %v, want at least a second", wait)
	}
}

func TestAnUnsetRuleAllowsEverything(t *testing.T) {
	limiter := New(Rule{})

	for attempt := 0; attempt < 100; attempt++ {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatal("an unconfigured limiter refused a request")
		}
	}
}

func TestConcurrentClientsDoNotCorruptTheCounters(t *testing.T) {
	limiter, _ := testLimiter(100, time.Minute)
	var waiting sync.WaitGroup

	for worker := 0; worker < 50; worker++ {
		waiting.Add(1)
		go func() {
			defer waiting.Done()
			for attempt := 0; attempt < 2; attempt++ {
				limiter.Allow("shared")
			}
		}()
	}
	waiting.Wait()

	// Exactly the limit was consumed, so the next attempt must be refused.
	if allowed, _ := limiter.Allow("shared"); allowed {
		t.Fatal("concurrent attempts were undercounted")
	}
}

func TestBlockedReportsWithoutConsumingTheBudget(t *testing.T) {
	limiter, _ := testLimiter(2, time.Minute)

	// Asking many times must not itself exhaust the allowance, or a caller that
	// only counts failures would count its successes too.
	for attempt := 0; attempt < 10; attempt++ {
		if blocked, _ := limiter.Blocked("client"); blocked {
			t.Fatalf("Blocked() consumed the budget on attempt %d", attempt)
		}
	}

	limiter.Allow("client")
	limiter.Allow("client")

	blocked, wait := limiter.Blocked("client")
	if !blocked || wait <= 0 {
		t.Fatalf("Blocked() = %v, %v after the limit was reached", blocked, wait)
	}
}

func TestBlockedIsFalseForAnUnknownKey(t *testing.T) {
	limiter, _ := testLimiter(1, time.Minute)

	if blocked, _ := limiter.Blocked("never-seen"); blocked {
		t.Fatal("an unknown key was reported as blocked")
	}
}
