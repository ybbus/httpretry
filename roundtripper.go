package httpretry

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// RetryRoundtripper is the roundtripper that will wrap around the actual http.Transport roundtripper
// to enrich the http client with retry functionality.
type RetryRoundtripper struct {
	Next             http.RoundTripper
	MaxRetryCount    int
	ShouldRetry      RetryPolicy
	CalculateBackoff BackoffPolicy
}

// RoundTrip implements the actual roundtripper interface (http.RoundTripper).
func (r *RetryRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp         *http.Response
		err          error
		dataBuffer   *bytes.Reader
		statusCode   int
		attemptCount = 1
		maxAttempts  = r.MaxRetryCount + 1
	)

	for {
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

		if !r.ShouldRetry(statusCode, err) {
			return resp, err
		}

		// no need to wait if we do not have retries left
		attemptCount++
		if attemptCount >= maxAttempts {
			break
		}

		backoff := r.CalculateBackoff(attemptCount)

		// we won't need the response anymore, drain (up to a maximum) and close it
		drainAndCloseBody(resp, 16384)

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			// context was canceled, return context error
			return nil, req.Context().Err()
		}
	}

	// no more attempts, return the last response / error
	return resp, err
}

func drainAndCloseBody(resp *http.Response, maxBytes int64) {
	if resp != nil {
		io.CopyN(ioutil.Discard, resp.Body, maxBytes)
		resp.Body.Close()
	}
}
