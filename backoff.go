package httpretry

import (
	"math"
	"time"
)

// BackoffPolicy is used to calculate the time to wait, before executing another request.
// The backoff can be calculated by taking the current number of retries into consideration.
type BackoffPolicy func(attemptCount int) time.Duration

var (
	// DefaultBackoffPolicy uses ExponentialBackoff(1 * time.Second)
	DefaultBackoffPolicy BackoffPolicy = ExponentialBackoff(1 * time.Second)

	// ConstantBackoff waits for the exact same duration after a failed retry.
	// If you set constantWait to 5 * time.Second, each retry will be triggered after 5 seconds.
	ConstantBackoff = func(constantWait time.Duration) BackoffPolicy {
		return func(attemptCount int) time.Duration {
			return constantWait
		}
	}

	// LinearBackoff increases the backoff time by multiplying the initial wait duration by the number of retries.
	// With an initialWait of 2 * time.Seconds, backoff durations will be: 2, 4, 6, 8, ...
	LinearBackoff = func(initialWait time.Duration) BackoffPolicy {
		return func(attemptCount int) time.Duration {
			return time.Duration(attemptCount) * initialWait
		}
	}

	// ExponentialBackoff increases the backoff exponentially by multiplying the initialWait with 2^attemptCount
	// With an initialWait of 1 * time.Seconds, backoff durations will be: 1, 2, 4, 8, 16, ...
	ExponentialBackoff = func(initialWait time.Duration) BackoffPolicy {
		return func(attemptCount int) time.Duration {
			return time.Duration(math.Pow(2, float64(attemptCount))) * initialWait
		}
	}
)
