package httpretry

import (
	"context"
	"net/http"
	"time"
)

type Option func(*RetryRoundtripper)

type RetryPolicy func(resp *http.Response, err error) (bool, error)

type BackoffPolicy func(attemptNum int, resp *http.Response, err error) time.Duration

var (
	// TODO: refine
	DefaultRetryPolicy = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		return err != nil || resp.StatusCode >= 500, err
	}

	// TODO: refine
	DefaultBackoffPolicy = func(attemptNum int, resp *http.Response) time.Duration {
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
)

func WithRetryCount(retryCount int) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.RetryCount = retryCount
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
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
		Next:       http.DefaultTransport,
		RetryCount: 3,
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
	RetryCount    int
	MinWait       time.Duration
	MaxWait       time.Duration
	RetryPolicy   RetryPolicy
	BackoffPolicy BackoffPolicy
}

func (r *RetryRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for count := 0; count <= r.RetryCount; count++ {
		// TODO: this does not work in all cases, since http.NewRequest does not recognize all body type / has a fallback
		// prepare the body, so the actual body will not be consumed

		// TODO: body may only have partially been read, drain and renew, otherwise we could start in the middle or without body at all
		if req.GetBody != nil {
			bodyReadCloser, _ := req.GetBody()
			req.Body = bodyReadCloser
		}

		resp, err = r.Next.RoundTrip(req)

		// TODO: if false and noerror -> successful
		// TODO: if false and error failed, but should not retry
		// TODO: if true and no error -> just go on
		retry, _ := r.RetryPolicy(resp, err)

		if retry {
			// TODO: close response body if exists (may happen even if err != nil)
			// TODO: set resp to nil, since we start over, otherwise we man have a response set when retries are over

			// TODO: also check for context canceled
			time.Sleep(r.BackoffPolicy(count+1, resp, err))
			continue
		}

		// TODO: what to do with retry error?

		break
	}

	// TODO: if we land here, retries are all over, if resp is not nil it should be the response of the last failed retry
	// TODO: if resp is nil, at least err is not nil.

	// TODO: what if response is not nil and err is nil (e.g. if a retry check was on some status code?) should we return an error or the actual response?

	// TODO: if error, close body and set resp to nil, or better do not touch response?
	return resp, err

}
