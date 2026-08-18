package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// rate limiter tracks one window state per client. if user A and user B are both hitting the API, they need their own independent count and their own window boundary, user A maxing out their limit shouldn't affect user B at all.
// state for one client's current window; how many requests they've made, and which window (identified by its start timestamp) the count belongs to.
type window struct {
	count       int
	windowStart int64
}

// we need the window - the small, per-client unit. How many requests has this one specific person made, and in which window.
// nothing else, it doesn't know about limits, doesn't know about other clients, doesn't know about locking.

type FixedWindowLimiter struct {
	// holds the shared rules (limit, interval), and a collection of window(s) one per client, so it can look up; "whats the state for this specific right now?"
	mu       sync.Mutex
	windows  map[string]*window
	limit    int
	interval time.Duration
}

// windows is a map keyed by client identifier(IP, user-id), each pointing to that client's window state.
// limit and interval are the configured rule; 100 req per 60 seconds

// function below initializes the map
func NewFixedWindowLimiter(limit int, interval time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		windows:  make(map[string]*window),
		limit:    limit,
		interval: interval,
	}
}

// Allow reports whether the request for key is allowed under the current
// window, and how many requests remain that window if so

func (l *FixedWindowLimiter) Allow(ctx context.Context, key string) (bool, int) {
	// first thing that happens is to lock the mutext, and schedule the unlock to happen automatically when the function returns
	l.mu.Lock()
	defer l.mu.Unlock()

	// clock trick that makes it a fixed window rather than a per-client rolling one
	// truncate rounds a time down to the nearest multiple of the given duration, measured from Go's zero time, not from whenever this client's first request came in
	// so with a 60-second interval, truncate snaps every timestamp down to its minute boundary:12:40:37 and 12:40:58 both truncate to 12:40:00
	// unix converts it to an integer so its cheap to compare and store
	currentWindow := time.Now().Truncate(l.interval).Unix()

	// each request checks; does this client have a window at all yet, or is their stored window from an earlier time bucket than the one we're in right now?
	// if either is true, a fresh window struct is created, count defaults to 0 and stored back into the map, otherwise the existing window keeps accumulating

	w, exists := l.windows[key]

	if !exists || w.windowStart != currentWindow {
		w = &window{windowStart: currentWindow}
		l.windows[key] = w

	}

	if w.count >= l.limit {
		return false, 0
	}

	w.count++
	return true, l.limit - w.count

}
