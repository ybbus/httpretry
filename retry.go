package httpretry

// RetryPolicy is used to decide if a request should be retried.
//
// This is done by examining the response status code and the error message of the last request.
//
// The statusCode may be 0 if there was no response available (e.g. in case of a request error).
type RetryPolicy func(statusCode int, err error) bool

var (
	// DefaultRetryPolicy will retry a request if an error occurred or the returned status code was >= 500
	// TODO: refine
	DefaultRetryPolicy RetryPolicy = func(statusCode int, err error) bool {
		return err != nil || statusCode >= 500 || statusCode == 0
	}
)
