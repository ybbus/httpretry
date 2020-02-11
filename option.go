package httpretry

// Option is a function type to modify the RetryRoundtripper configuration
type Option func(*RetryRoundtripper)

// WithMaxRetryCount sets the maximum number of retries if an http request was not successful.
func WithMaxRetryCount(maxRetryCount int) Option {
	if maxRetryCount < 0 {
		maxRetryCount = 0
	}
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.MaxRetryCount = maxRetryCount
	}
}

// WithRetryPolicy sets the user defined retry policy.
func WithRetryPolicy(retryPolicy RetryPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.ShouldRetry = retryPolicy
	}
}

// WithBackoffPolicy sets the user defined backoff policy.
func WithBackoffPolicy(backoffPolicy BackoffPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.CalculateBackoff = backoffPolicy
	}
}
