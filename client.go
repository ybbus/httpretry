package httpretry

import (
	"net/http"
)

const ()

func NewRetryableClient(opts ...Option) *http.Client {
	return MakeRetryable(&http.Client{}, opts...)
}

func MakeRetryable(client *http.Client, opts ...Option) *http.Client {
	if client == nil {
		panic("client must not be nil")
	}

	nextRoundtripper := client.Transport
	if nextRoundtripper == nil {
		nextRoundtripper = http.DefaultTransport
	}

	// set defaults
	retryRoundtripper := &RetryRoundtripper{
		Next:          nextRoundtripper,
		MaxRetryCount: DefaultMaxRetryCount,
		RetryPolicy:   DefaultRetryPolicy,
		BackoffPolicy: DefaultBackoffPolicy,
	}

	// overwrite defaults with user provided configuration
	for _, o := range opts {
		o(retryRoundtripper)
	}

	client.Transport = retryRoundtripper

	return client
}
