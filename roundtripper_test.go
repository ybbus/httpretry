package httpretry

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRetryRoundtripperSimple(t *testing.T) {
	check := assert.New(t)

	mockRoundtripper := &MockRoundtripper{}
	mockBackoffPolicy := &MockBackoffPolicy{}
	mockRetryPolicy := &MockRetryPolicy{}

	retryRoundtripper := RetryRoundtripper{
		MaxRetryCount:    3,
		Next:             mockRoundtripper,
		ShouldRetry:      mockRetryPolicy.ShouldRetry,
		CalculateBackoff: mockBackoffPolicy.CalculateBackoff,
	}

	reset := func() {
		mockRoundtripper.reset()
		mockBackoffPolicy.reset()
		mockRetryPolicy.reset()
		retryRoundtripper.MaxRetryCount = 3
		retryRoundtripper.CalculateBackoff = mockBackoffPolicy.CalculateBackoff
		retryRoundtripper.ShouldRetry = mockRetryPolicy.ShouldRetry
		retryRoundtripper.Next = mockRoundtripper
	}

	t.Run("should return response if first call was successful", func(t *testing.T) {
		reset()
		req, _ := http.NewRequest("GET", "https://my-super-nonexisting-url.asd", nil)
		res, err := retryRoundtripper.RoundTrip(req)

		check.Equal(1, mockRetryPolicy.CallCount)
		check.Equal(0, mockBackoffPolicy.CallCount)
		check.Equal(1, mockRoundtripper.CallCount)
		check.True(responseContains(t, res, "OK"))
		check.NoError(err)
	})

	t.Run("should retry one time if second call was successful", func(t *testing.T) {
		reset()
		mockRoundtripper.RoundTripFunc = func(req *http.Request, called int) (response *http.Response, e error) {
			switch called {
			case 1:
				return FakeResponse(req, 500, []byte("error")), nil
			default:
				return FakeResponse(req, 200, []byte("ok")), nil
			}
		}

		req, _ := http.NewRequest("GET", "https://my-super-nonexisting-url.asd", nil)
		res, err := retryRoundtripper.RoundTrip(req)

		check.Equal(2, mockRetryPolicy.CallCount)
		check.Equal(1, mockBackoffPolicy.CallCount)
		check.Equal(2, mockRoundtripper.CallCount)
		check.True(responseContains(t, res, "ok"))
		check.Equal(200, res.StatusCode)
		check.NoError(err)
	})

	t.Run("should give up after retries are over", func(t *testing.T) {
		reset()
		retryRoundtripper.MaxRetryCount = 2
		mockRoundtripper.RoundTripFunc = func(req *http.Request, called int) (response *http.Response, e error) {
			switch called {
			case 1, 2:
				return FakeResponse(req, 500, []byte("error")), nil
			case 3:
				return FakeResponse(req, 500, []byte("finished")), nil
			}
			panic("panic")
		}

		req, _ := http.NewRequest("GET", "https://my-super-nonexisting-url.asd", nil)
		res, err := retryRoundtripper.RoundTrip(req)

		check.Equal(3, mockRetryPolicy.CallCount)
		check.Equal(2, mockBackoffPolicy.CallCount)
		check.Equal(3, mockRoundtripper.CallCount)
		check.True(responseContains(t, res, "finished"))
		check.Equal(500, res.StatusCode)
		check.NoError(err)
	})

	t.Run("should give up if context was canceled", func(t *testing.T) {
		reset()
		retryRoundtripper.MaxRetryCount = 2
		ctx, cancel := context.WithCancel(context.Background())
		mockRoundtripper.RoundTripFunc = func(req *http.Request, called int) (response *http.Response, e error) {
			cancel()
			return FakeResponse(req, 500, []byte("error")), nil
		}

		req, _ := http.NewRequestWithContext(ctx, "GET", "https://my-super-nonexisting-url.asd", nil)
		res, err := retryRoundtripper.RoundTrip(req)

		check.Equal(1, mockRetryPolicy.CallCount)
		check.Equal(1, mockBackoffPolicy.CallCount)
		check.Equal(1, mockRoundtripper.CallCount)
		check.Nil(res)
		check.Contains(err.Error(), "context canceled")
	})
}

func TestRetryRoundtripperWithBody(t *testing.T) {
	check := assert.New(t)

	mockRoundtripper := &MockRoundtripper{}
	mockBackoffPolicy := &MockBackoffPolicy{}
	mockRetryPolicy := &MockRetryPolicy{}

	retryRoundtripper := RetryRoundtripper{
		MaxRetryCount:    3,
		Next:             mockRoundtripper,
		ShouldRetry:      mockRetryPolicy.ShouldRetry,
		CalculateBackoff: mockBackoffPolicy.CalculateBackoff,
	}

	reset := func() {
		mockRoundtripper.reset()
		mockBackoffPolicy.reset()
		mockRetryPolicy.reset()
		retryRoundtripper.MaxRetryCount = 3
		retryRoundtripper.CalculateBackoff = mockBackoffPolicy.CalculateBackoff
		retryRoundtripper.ShouldRetry = mockRetryPolicy.ShouldRetry
		retryRoundtripper.Next = mockRoundtripper
	}

	t.Run("should return response if first call was successful", func(t *testing.T) {
		reset()
		check.True(true)
	})
}

func responseContains(t *testing.T, res *http.Response, substring string) bool {
	t.Helper()
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("could not read body: ", err.Error())
	}
	return strings.Contains(string(d), substring)
}

type MockRoundtripper struct {
	CallCount     int
	RoundTripFunc func(req *http.Request, called int) (*http.Response, error)
}

func (f *MockRoundtripper) reset() {
	f.CallCount = 0
	f.RoundTripFunc = nil
}

func (f *MockRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.CallCount++
	if f.RoundTripFunc != nil {
		return f.RoundTripFunc(req, f.CallCount)
	}

	return FakeResponse(req, 200, []byte("OK")), nil
}

type MockBackoffPolicy struct {
	CallCount   int
	BackoffFunc func(attempt int) time.Duration
}

func (mbp *MockBackoffPolicy) reset() {
	mbp.CallCount = 0
	mbp.BackoffFunc = nil
}

func (mbp *MockBackoffPolicy) CalculateBackoff(attempt int) time.Duration {
	mbp.CallCount++
	if mbp.BackoffFunc != nil {
		return mbp.BackoffFunc(attempt)
	}
	return 10 * time.Millisecond
}

type MockRetryPolicy struct {
	CallCount int
	RetryFunc func(statusCode int, err error) bool
}

func (mrp *MockRetryPolicy) reset() {
	mrp.CallCount = 0
	mrp.RetryFunc = nil
}

func (mrp *MockRetryPolicy) ShouldRetry(statusCode int, err error) bool {
	mrp.CallCount++
	if mrp.RetryFunc != nil {
		return mrp.RetryFunc(statusCode, err)
	}

	// simple default retry
	return statusCode == 500 || statusCode == 0 || err != nil
}

func FakeResponse(req *http.Request, code int, body []byte) *http.Response {
	codemap := map[int]string{
		200: "200 OK",
	}

	var bodyReadCloser io.ReadCloser
	var contentLength int64 = -1

	if len(body) != 0 {
		bodyReadCloser = ioutil.NopCloser(bytes.NewReader(body))
		contentLength = int64(len(body))
	}

	return &http.Response{
		Status:        codemap[code],
		StatusCode:    code,
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		ProtoMinor:    0,
		Uncompressed:  true,
		ContentLength: contentLength,
		Body:          bodyReadCloser,
		Request:       req,
	}
}
