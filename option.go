package httpretry

type Option func(*RetryRoundtripper)

func WithMaxRetryCount(maxRetryCount int) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.MaxRetryCount = maxRetryCount
	}
}

func WithRetryPolicy(retryPolicy CheckRetryPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.RetryPolicy = retryPolicy
	}
}

func WithBackoffPolicy(backoffPolicy BackoffPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.BackoffPolicy = backoffPolicy
	}
}
