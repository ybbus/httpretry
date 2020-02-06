package httpretry

import (
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Option func(*RetryRoundtripper)

// TODO: should be possible to check on response body and reset after checking
type CheckRetryPolicy func(resp *http.Response, err error) bool

type BackoffPolicy func(attemptNum int, resp *http.Response, err error) time.Duration

var (
	// TODO: refine
	DefaultRetryPolicy = func(resp *http.Response, err error) bool {
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

	noRetry := false
	maxAttempts := r.MaxRetryCount + 1
	for attemptCount := 1; attemptCount <= maxAttempts; attemptCount++ {
		resp = nil
		err = nil

		// TODO: this does not work in all cases, since http.NewRequest does not recognize all body types / has no fallback
		// TODO: body may only have partially been red -> drain and renew, otherwise we could start in the middle or without body at all
		if req.GetBody != nil {
			// TODO: we should use our own body function here to support more bodies and maybe have a fallback for unknown bodies
			bodyReadCloser, _ := req.GetBody()
			req.Body = bodyReadCloser
		} else if req.Body != nil {
			noRetry = true
		}

		resp, err = r.Next.RoundTrip(req)

		// TODO: because of the used io.Reader, we may not retry, can we do better? maybe io.TeeReader and RetryBufferSize
		// TODO: check if request was completely red, and if not read the rest in

		// TODO: should we be able to access (consume) the resp body here?
		if noRetry || !r.RetryPolicy(resp, err) {
			// TODO: for "noRetry" we need a logger here, since this info should not be propagated in the err message, or should it?
			return resp, err
		}

		// TODO: should we be able to access (consume) the resp body here?
		backoff := r.BackoffPolicy(attemptCount, resp, err)

		// wo won't need the response anymore, drain (4096kb) and close it
		drainAndCloseBody(resp)

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
			continue
		case <-req.Context().Done():
			// context was canceled, return context error
			return nil, req.Context().Err()
		}
	}

	// no more attempts, return the last response / error
	return resp, err
}

func drainAndCloseBody(resp *http.Response) {
	if resp != nil {
		// TODO: can this block?
		io.CopyN(ioutil.Discard, resp.Body, 4096)
		resp.Body.Close()
	}
}
