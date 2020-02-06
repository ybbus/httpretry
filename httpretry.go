package httpretry

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// CheckRetryPolicy is used to decide if a request should be retried.
// This is done by examining the response status code and the error message of the last request.
// The statusCode may be 0 if there was no response available (e.g. in case of a request error).
type CheckRetryPolicy func(statusCode int, err error) bool

// BackoffPolicy is used to calculate the time to wait, before executing another request.
// The backoff may be calculated by taking the current number of retries into consideration.
type BackoffPolicy func(attemptNum int) time.Duration

var (
	// DefaultRetryPolicy will retry a request if an error occurred or the returned status code was >= 500
	// TODO: refine
	DefaultRetryPolicy CheckRetryPolicy = func(statusCode int, err error) bool {
		return err != nil || statusCode >= 500
	}

	// TODO: refine
	DefaultBackoffPolicy BackoffPolicy = LinearBackoff(1 * time.Second)

	// TODO: refine
	ConstantBackoff = func(wait time.Duration) BackoffPolicy {
		return func(attemptNum int) time.Duration {
			return wait
		}
	}

	LinearBackoff = func(initialWait time.Duration) BackoffPolicy {
		return func(attemptNum int) time.Duration {
			return time.Duration(attemptNum) * initialWait
		}
	}

	DefaultMaxRetryCount = 3
)

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

type RetryRoundtripper struct {
	Next          http.RoundTripper
	MaxRetryCount int
	RetryPolicy   CheckRetryPolicy
	BackoffPolicy BackoffPolicy
}

func (r *RetryRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp        *http.Response
		err         error
		dataBuffer  *bytes.Reader
		statusCode  int
		maxAttempts = r.MaxRetryCount + 1
	)

	for attemptCount := 1; attemptCount <= maxAttempts; attemptCount++ {
		resp = nil
		err = nil
		statusCode = 0

		// if request provides GetBody() we use it as Body,
		// because GetBody can be retrieved arbitrary times for retry
		if req.GetBody != nil {
			bodyReadCloser, _ := req.GetBody()
			req.Body = bodyReadCloser
		} else if req.Body != nil {

			// we need to store the complete body, since we need to reset it if a retry happens
			// but: not very efficient because:
			// a) huge stream data size will all be buffered completely in the memory
			//    imagine: 1GB stream data would work efficiently with io.Copy, but has to be buffered completely in memory
			// b) unnecessary if first attempt succeeds
			// a solution would be to at least support more types for GetBody()

			// store it for the first time
			if dataBuffer == nil {
				data, err := ioutil.ReadAll(req.Body)
				req.Body.Close()
				if err != nil {
					return nil, err
				}
				dataBuffer = bytes.NewReader(data)
				req.ContentLength = int64(dataBuffer.Len())
				req.Body = ioutil.NopCloser(dataBuffer)
			}

			// reset the request body
			dataBuffer.Seek(0, io.SeekStart)
		}

		resp, err = r.Next.RoundTrip(req)
		if resp != nil {
			statusCode = resp.StatusCode
		}

		if !r.RetryPolicy(statusCode, err) {
			return resp, err
		}

		backoff := r.BackoffPolicy(attemptCount)

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
		io.CopyN(ioutil.Discard, resp.Body, 16384)
		resp.Body.Close()
	}
}
