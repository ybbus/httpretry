package httpretry

import (
	"net"
	"net/http"
	"net/url"
)

// RetryPolicy is used to decide if a request should be retried.
//
// This is done by examining the response status code and the error message of the last request.
//
// The statusCode may be 0 if there was no response available (e.g. in case of a request error).
type RetryPolicy func(statusCode int, err error) bool

var DefaultRetryPolicy RetryPolicy = func(statusCode int, err error) bool {
	// retry if error is temporary, if not we will do further checks
	t, ok := err.(interface{ Temporary() bool })
	if ok && t.Temporary() {
		return true
	}

	switch e := err.(type) {
	case *url.Error:
		if urlErrorRetry(e) {
			return true
		}
	case nil: // no error, continue
	default: // we should always retry unknown errors
		return true
	}

	switch statusCode {
	case // status codes that should be retried
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusLocked,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	case 0: // means we did not get a response. we need to retry
		return true
	default: // on all other status codes we should not retry
		return false
	}
}

func urlErrorRetry(urlError *url.Error) bool {
	// parse errors should net be retried
	if urlError.Op == "parse" {
		return false
	}

	// dial errors may be retried
	switch e := urlError.Err.(type) {
	case *net.OpError:
		// TODO: all of them?
		return e.Op == "dial"
	}

	// TODO: something else that should be retried?
	// all other errors should be retried
	return true
}
