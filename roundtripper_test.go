package httpretry

/*
func TestRetryRoundtripper(t *testing.T) {
	check := assert.New(t)

	mockRoundtripper := &MockRoundtripper{}

	retryRoundtripper := RetryRoundtripper{
		Next:             mockRoundtripper,
		MaxRetryCount:    3,
		ShouldRetry:      DefaultRetryPolicy,
		CalculateBackoff: DefaultBackoffPolicy,
	}

	tests := []struct {
		Description     string
		RequestIn       *http.Request
		ExpectedRetries int
	}{
		{
			Description:     "Should return OK response",
			RequestIn:       nil,
			ExpectedRetries: 0,
		},
	}

	for _, test := range tests {

	}
}

type MockRoundtripper struct {
	CallCount     int
	RoundTripFunc []func(req *http.Request) (*http.Response, error)
}

func (f *MockRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.RoundTripFunc == nil {
		panic("no roundtripper implementation")
	}
	f.CallCount++
	return f.RoundTripFunc(req)
}

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (rtf RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return rtf(req)
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
*/

/*
var (
	ShortBackoffPolicy = httpretry.ConstantBackoff(100*time.Millisecond, 0)

	OnlyErrorRetryPolicy   httpretry.RetryPolicy = func(statusCode int, err error) bool { return err != nil }
	ErrorAnd500RetryPolicy httpretry.RetryPolicy = func(statusCode int, err error) bool { return err != nil || statusCode == 500 }
)

func TestNewRetryClient(t *testing.T) {
	check := assert.New(t)

	client := httpretry.NewRetryClient(httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)
	retryRoundtripper, _ := client.Transport.(*httpretry.RetryRoundtripper)
	check.NotNil(retryRoundtripper.Next)
	check.Equal(3, retryRoundtripper.MaxRetryCount)
	check.NotNil(retryRoundtripper.CalculateBackoff)
	check.NotNil(retryRoundtripper.ShouldRetry)
}

func TestMakeRetryClient(t *testing.T) {
	check := assert.New(t)

	customHTTPClient := &http.Client{}
	client := httpretry.MakeRetryClient(customHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	check.Equal(customHTTPClient, client)
	check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)
	retryRoundtripper, _ := client.Transport.(*httpretry.RetryRoundtripper)
	check.NotNil(retryRoundtripper.Next)
	check.NotNil(retryRoundtripper.CalculateBackoff)
	check.NotNil(retryRoundtripper.ShouldRetry)
	check.Equal(3, retryRoundtripper.MaxRetryCount)
}

func TestSuccessfulGet(t *testing.T) {
	check := assert.New(t)

	mockRoundtripper := &MockRoundtripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return FakeResponse(req, 200, []byte("OK")), nil
		},
	}

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	res, err := client.Get("http://someurl.com")
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(1, mockRoundtripper.CallCount)
}

func TestSuccessfulGetWithRetry(t *testing.T) {
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
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

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
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

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
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

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
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	res, err := client.Post("http://someurl.com", "", bytes.NewReader(postBody))
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	check.Equal(3, callCount)
	check.Equal(postBody, receiveBody)
}

func TestNonRetryableIOReaderShouldBufferRetry(t *testing.T) {
	check := assert.New(t)

	var (
		callCount   = 0
		postBody    = []byte("postbody")
		receiveBody []byte
	)

	// use a pipe, since this generates a Pipereader, that should not be retriable (for now)
	r, w := io.Pipe()
	go func() {
		w.Write(postBody)
		w.Close()
	}()

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		receiveBodyTemp, _ := ioutil.ReadAll(req.Body)
		if receiveBody != nil {
			check.Equal(receiveBody, receiveBodyTemp)
		}
		receiveBody = receiveBodyTemp
		return nil, errors.New("some error")
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	res, err := client.Post("http://someurl.com", "", r)
	check.Contains(err.Error(), "some error")
	check.Nil(res)
	check.Equal(4, callCount)
	check.Equal(postBody, receiveBody)
}

func TestContextTimeoutCancelsRetry(t *testing.T) {
	check := assert.New(t)

	callCount := 0

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		return nil, errors.New("some error")
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))

	timeoutContext, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)

	req, _ := http.NewRequestWithContext(timeoutContext, "GET", "http://someurl.com", nil)

	res, err := client.Do(req)
	check.Contains(err.Error(), "context deadline exceeded")
	check.Nil(res)
	check.Equal(1, callCount)
}

func TestRetryOnStatusCode(t *testing.T) {
	check := assert.New(t)

	callCount := 0

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++

		if callCount == 3 {
			return FakeResponse(req, 200, []byte("ok")), nil
		}

		return FakeResponse(req, 500, []byte("error response")), nil
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}
	client := httpretry.MakeRetryClient(testHTTPClient,
		httpretry.WithBackoffPolicy(ShortBackoffPolicy),
		httpretry.WithRetryPolicy(func(statusCode int, err error) bool {
			if err != nil || statusCode == 500 {
				return true
			}
			return false
		}),
	)

	req, _ := http.NewRequest("GET", "http://someurl.com", nil)

	res, err := client.Do(req)
	check.Nil(err)
	check.NotNil(res)
	check.Equal(200, res.StatusCode)
	check.Equal(3, callCount)
}

func TestCancelingContextCancelsRetry(t *testing.T) {
	check := assert.New(t)

	callCount := 0
	ctx, cancel := context.WithCancel(context.Background())

	mockRoundtripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 2 {
			cancel()
		}
		return nil, errors.New("some error")
	})

	testHTTPClient := &http.Client{
		Transport: mockRoundtripper,
	}

	client := httpretry.MakeRetryClient(testHTTPClient, httpretry.WithBackoffPolicy(ShortBackoffPolicy))
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://someurl.com", nil)
	res, err := client.Do(req)

	check.Nil(res)
	check.Contains(err.Error(), "context canceled")
	check.Equal(2, callCount)
}

*/
