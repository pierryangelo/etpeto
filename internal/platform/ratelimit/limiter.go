// Package ratelimit caps how often one client may reach an expensive or
// sensitive route. It keeps its state in memory on purpose: the specification
// rules out Redis before there is a measured need, and a single API process
// serves one installation.
package ratelimit

import (
	"sync"
	"time"
)

// maxTrackedKeys bounds the memory a flood of distinct clients can occupy.
// Beyond it the oldest bucket is dropped, which at worst forgives a client that
// had been silent longest.
const maxTrackedKeys = 20_000

type Rule struct {
	// Limit is how many requests are allowed inside Window.
	Limit  int
	Window time.Duration
}

type bucket struct {
	// hits are the instants still inside the window, oldest first.
	hits []time.Time
}

type Limiter struct {
	rule    Rule
	now     func() time.Time
	mutex   sync.Mutex
	buckets map[string]*bucket
}

func New(rule Rule) *Limiter {
	return NewWithClock(rule, time.Now)
}

func NewWithClock(rule Rule, now func() time.Time) *Limiter {
	return &Limiter{rule: rule, now: now, buckets: map[string]*bucket{}}
}

// Blocked reports whether the key is currently over its limit, without
// recording an attempt. It exists because some callers only want to count the
// failures — a sign-in that succeeds must not consume the budget meant to hold
// back guessing.
func (l *Limiter) Blocked(key string) (bool, time.Duration) {
	if l.rule.Limit <= 0 || l.rule.Window <= 0 {
		return false, 0
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := l.now()
	entry, known := l.buckets[key]
	if !known {
		return false, 0
	}
	entry.hits = keepAfter(entry.hits, now.Add(-l.rule.Window))
	if len(entry.hits) < l.rule.Limit {
		return false, 0
	}
	wait := entry.hits[0].Add(l.rule.Window).Sub(now)
	if wait < time.Second {
		wait = time.Second
	}
	return true, wait
}

// Allow records an attempt and reports whether it fits within the rule. When it
// does not, it also returns how long the caller should wait, which is what the
// response tells the client in Retry-After.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l.rule.Limit <= 0 || l.rule.Window <= 0 {
		return true, 0
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := l.now()
	threshold := now.Add(-l.rule.Window)

	entry, known := l.buckets[key]
	if !known {
		l.evictIfCrowded(threshold)
		entry = &bucket{}
		l.buckets[key] = entry
	}
	entry.hits = keepAfter(entry.hits, threshold)

	if len(entry.hits) >= l.rule.Limit {
		// The window frees up when the oldest hit inside it falls out.
		oldest := entry.hits[0]
		wait := oldest.Add(l.rule.Window).Sub(now)
		if wait < time.Second {
			wait = time.Second
		}
		return false, wait
	}
	entry.hits = append(entry.hits, now)
	return true, 0
}

// evictIfCrowded first drops buckets that are entirely expired, and only then
// falls back to removing an arbitrary one. Callers hold the mutex.
func (l *Limiter) evictIfCrowded(threshold time.Time) {
	if len(l.buckets) < maxTrackedKeys {
		return
	}
	for key, entry := range l.buckets {
		if len(keepAfter(entry.hits, threshold)) == 0 {
			delete(l.buckets, key)
		}
	}
	if len(l.buckets) < maxTrackedKeys {
		return
	}
	for key := range l.buckets {
		delete(l.buckets, key)
		return
	}
}

func keepAfter(hits []time.Time, threshold time.Time) []time.Time {
	cut := 0
	for cut < len(hits) && !hits[cut].After(threshold) {
		cut++
	}
	return hits[cut:]
}
