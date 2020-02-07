package httpretry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffPolicy is used to calculate the time to wait, before executing another request.
//
// The backoff can be calculated by taking the current number of retries into consideration.
type BackoffPolicy func(attemptCount int) time.Duration

var (
	// DefaultBackoffPolicy uses ExponentialBackoff with 1 second minWait, 30 seconds max wait and 100ms max jitter
	DefaultBackoffPolicy BackoffPolicy = ExponentialBackoff(1*time.Second, 30*time.Second, 100*time.Millisecond)

	// ConstantBackoff waits for the exact same duration after a failed retry.
	//
	// maxJitter is a random interval [0, maxJitter) added to the constant backoff.
	//
	// Example: constantWait = 5 * time.Second, backoff will be: 5 seconds + [0, maxJitter).
	ConstantBackoff = func(constantWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if constantWait.Milliseconds() < 0 {
			constantWait = 0
		}
		if maxJitter.Milliseconds() < 0 {
			maxJitter = 0
		}

		return func(attemptCount int) time.Duration {
			return constantWait + randJitter(maxJitter)
		}
	}

	// LinearBackoff increases the backoff time by multiplying the minWait duration by the number of retries.
	//
	// maxJitter is a random interval [0, maxJitter) added to the linear backoff.
	//
	// if maxWait > 0, this will set an upper bound of the maximum time to wait between to requests.
	//
	// Example: minWait = 2 * time.Seconds, backoff will be: 2, 4, 6, 8, ... + [0, maxJitter).
	LinearBackoff = func(minWait time.Duration, maxWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if minWait.Milliseconds() < 0 {
			minWait = 0
		}
		if maxJitter.Milliseconds() < 0 {
			maxJitter = 0
		}
		if maxWait < minWait {
			maxWait = 0
		}
		return func(attemptCount int) time.Duration {
			nextWait := time.Duration(attemptCount+1)*minWait + randJitter(maxJitter)
			if maxWait > 0 {
				return minDuration(nextWait, maxWait)
			}
			return nextWait
		}
	}

	// ExponentialBackoff increases the backoff exponentially by multiplying the minWait with 2^attemptCount
	//
	// maxJitter is a random interval [0, maxJitter) added to the exponential backoff.
	//
	// if maxWait > 0, this will set an upper bound of the maximum time to wait between to requests.
	//
	// Example minWait = 1 * time.Seconds, backoff will be: 1, 2, 4, 8, 16, ... + [0, maxJitter)
	ExponentialBackoff = func(minWait time.Duration, maxWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if minWait.Milliseconds() < 0 {
			minWait = 0
		}
		if maxJitter.Milliseconds() < 0 {
			maxJitter = 0
		}
		if maxWait < minWait {
			maxWait = 0
		}
		return func(attemptCount int) time.Duration {
			nextWait := time.Duration(math.Pow(2, float64(attemptCount)))*minWait + randJitter(maxJitter)
			if maxWait > 0 {
				return minDuration(nextWait, maxWait)
			}
			return nextWait
		}
	}
)

// minDuration returns the minimum of two durations
func minDuration(duration1 time.Duration, duration2 time.Duration) time.Duration {
	if duration1.Milliseconds() < duration2.Milliseconds() {
		return duration1
	}
	return duration2
}

// randJitter returns a random duration in the interval [0, maxJitter)
func randJitter(maxJitter time.Duration) time.Duration {
	maxJitterMs := maxJitter.Milliseconds()
	if maxJitterMs == 0 {
		return time.Duration(0)
	}

	return time.Duration(rand.Intn(int(maxJitterMs))) * time.Millisecond
}
