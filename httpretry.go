package httpretry

import (
	"context"
	"net/http"
	"time"
)

type Option func(*RetryRoundtripper)

type CheckRetryPolicy func(ctx context.Context, resp *http.Response, err error) bool

type BackoffPolicy func(attemptNum int, resp *http.Response, err error) time.Duration

var (
	// TODO: refine
	DefaultRetryPolicy = func(ctx context.Context, resp *http.Response, err error) bool {
		return err != nil || resp.StatusCode >= 500
	}

	// TODO: refine
	DefaultBackoffPolicy = func(attemptNum int, resp *http.Response, err error) time.Duration {
		return 1 * time.Second
	}

	// TODO: refine
	ConstantBackoff = func(waitTime time.Duration) BackoffPolicy {
		return func(attemptNum int, resp *http.Response, err error) time.Duration {
			return waitTime
		}
	}

	LinearBackoff = func(waitTime time.Duration) BackoffPolicy {
		return func(attemptNum int, resp *http.Response, err error) time.Duration {
			return time.Duration(attemptNum) * waitTime
		}
	}

	DefaultMaxRetryCount = 3
)

func WithMaxRetryCount(retryCount int) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.MaxRetryCount = retryCount
	}
}

func WithRetryPolicy(policy CheckRetryPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.RetryPolicy = policy
	}
}

func WithBackoffPolicy(policy BackoffPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.BackoffPolicy = policy
	}
}

// TODO: new may be the wrong term, since we reuse the client
func NewDefaultClient(opts ...Option) *http.Client {
	return NewClient(&http.Client{}, opts...)
}

// TODO: new may be the wrong term, since we reuse the client
func NewClient(client *http.Client, opts ...Option) *http.Client {
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

const (
	DefaultRetryCount = 3
)

type RetryRoundtripper struct {
	Next          http.RoundTripper
	MaxRetryCount int
	RetryPolicy   CheckRetryPolicy
	BackoffPolicy BackoffPolicy
}

func (r *RetryRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for attemptCount := 0; attemptCount <= r.MaxRetryCount; attemptCount++ {
		resp = nil
		err = nil

		// TODO: this does not work in all cases, since http.NewRequest does not recognize all body types / has no fallback
		// TODO: body may only have partially been red -> drain and renew, otherwise we could start in the middle or without body at all
		if req.GetBody != nil {
			// TODO: we should use our own body function here to support more bodies and maybe have a fallback for unknown bodies
			bodyReadCloser, _ := req.GetBody()
			req.Body = bodyReadCloser
		}

		resp, err = r.Next.RoundTrip(req)

		if !r.RetryPolicy(req.Context(), resp, err) {
			break
		}

		if resp != nil {
			// TODO: drain the body first?
			resp.Body.Close()
		}

		// TODO: also check for context canceled
		time.Sleep(r.BackoffPolicy(attemptCount, resp, err))
	}

	return resp, err
}
