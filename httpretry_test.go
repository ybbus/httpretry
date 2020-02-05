package httpretry_test

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/ybbus/httpretry"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

var TestBackoffPolicy = func(attemptNum int, resp *http.Response, err error) time.Duration {
	return 100 * time.Millisecond
}

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (rtf RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return rtf(req)
}

func TestNewDefaultClient(t *testing.T) {
	check := assert.New(t)

	client := httpretry.NewDefaultClient(httpretry.WithBackoffPolicy(TestBackoffPolicy))

	check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)
	retryRoundtripper, _ := client.Transport.(*httpretry.RetryRoundtripper)
	check.NotNil(retryRoundtripper.Next)
	check.NotNil(retryRoundtripper.BackoffPolicy)
	check.NotNil(retryRoundtripper.RetryPolicy)
}

func TestNewClient(t *testing.T) {
	check := assert.New(t)

	customHTTPClient := &http.Client{}
	client := httpretry.NewClient(customHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)
	retryRoundtripper, _ := client.Transport.(*httpretry.RetryRoundtripper)
	check.NotNil(retryRoundtripper.Next)
	check.NotNil(retryRoundtripper.BackoffPolicy)
	check.NotNil(retryRoundtripper.RetryPolicy)
}

func TestSuccessfulGet(t *testing.T) {
	check := assert.New(t)

	callCount := 0
	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		return FakeResponse(req, 200, []byte("OK")), nil
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.NewClient(testHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	res, err := client.Get("http://someurl.com")
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(1, callCount)
}

func TestSuccessfulGetOneRetry(t *testing.T) {
	check := assert.New(t)

	callCount := 0
	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		switch callCount {
		case 1:
			return nil, errors.New("some error")
		case 2:
			return FakeResponse(req, 200, []byte("OK")), nil
		default:
			t.Fatal("unexpected call")
		}
		return FakeResponse(req, 200, []byte("OK")), nil
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.NewClient(testHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	res, err := client.Get("http://someurl.com")
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(2, callCount)
}

func TestGiveUpGet(t *testing.T) {
	check := assert.New(t)

	callCount := 0
	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		return nil, errors.New("some error")
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.NewClient(testHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	res, err := client.Get("http://someurl.com")
	check.Nil(res)
	check.Contains(err.Error(), "some error")
	check.Equal(4, callCount)
}

func TestSuccessfulPostSimpleBytes(t *testing.T) {
	check := assert.New(t)

	callCount := 0
	postBody := []byte("postbody")
	var receiveBody []byte

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		receiveBody, _ = ioutil.ReadAll(req.Body)
		return FakeResponse(req, 200, []byte("OK")), nil
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.NewClient(testHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	res, err := client.Post("http://someurl.com", "", bytes.NewReader(postBody))
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(1, callCount)
	check.Equal(postBody, receiveBody)
}

func TestSuccessfulPostSimpleBytesRetry(t *testing.T) {
	check := assert.New(t)

	var (
		callCount   = 0
		postBody    = []byte("postbody")
		receiveBody []byte
	)

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		receiveBodyTemp, _ := ioutil.ReadAll(req.Body)
		if receiveBody != nil {
			check.Equal(receiveBody, receiveBodyTemp)
		}
		receiveBody = receiveBodyTemp

		if callCount <= 2 {
			return nil, errors.New("some error")
		}
		return FakeResponse(req, 200, []byte("OK")), nil
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.NewClient(testHTTPClient, httpretry.WithBackoffPolicy(TestBackoffPolicy))

	res, err := client.Post("http://someurl.com", "", bytes.NewReader(postBody))
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(3, callCount)
	check.Equal(postBody, receiveBody)
}

// TODO: test with ssl (secure disabled, custom cert)
// TODO: test server times out request not possible with mockServer atm

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
