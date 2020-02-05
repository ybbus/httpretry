package httpretry_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/ybbus/httpmockserver"
	"github.com/ybbus/httpretry"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

func TestNewDefaultClient(t *testing.T) {
	check := assert.New(t)

	client := httpretry.NewDefaultClient()

	check.IsType(&httpretry.RetryRoundtripper{}, client.Transport, "client.Transport must be of type *RetryRoundtripper")
	retryRoundtripper, ok := client.Transport.(*httpretry.RetryRoundtripper)
	check.True(ok, "client.Transport must be of type RetryRoundtripper")
	check.NotNil(retryRoundtripper.Next, "RetryRoundtripper must wrap another Roundtripper")
	check.Equal(httpretry.DefaultRetryCount, retryRoundtripper.RetryCount, "retrycount must use default value when not set")
}

func TestWithRetryCount(t *testing.T) {
	check := assert.New(t)

	retryCount := 5
	client := httpretry.NewDefaultClient(httpretry.WithRetryCount(retryCount))

	retryRoundtripper, _ := client.Transport.(*httpretry.RetryRoundtripper)
	check.Equal(retryCount, retryRoundtripper.RetryCount)
}

func TestConnErrorNoEndpoint(t *testing.T) {
	check := assert.New(t)

	// TODO: add 100 ms linear backoff here

	retryCount := 3
	client := httpretry.NewClient(
		&http.Client{},
		httpretry.WithRetryCount(retryCount),
	)

	res, err := client.Get("http://0.0.0.0:1234")

	check.Contains(err.Error(), "No connection could be made")
	check.Nil(res)
}

func TestConnErrorNoDNS(t *testing.T) {
	check := assert.New(t)

	// TODO: add 100 ms linear backoff here

	retryCount := 3
	client := httpretry.NewClient(
		&http.Client{},
		httpretry.WithRetryCount(retryCount),
	)

	res, err := client.Get("http://fqdn-should-not-exist.hopefully:1234")

	check.Contains(err.Error(), "no such host")
	check.Nil(res)
}

func TestConnClosed(t *testing.T) {
	check := assert.New(t)

	// TODO: add 100 ms linear backoff here

	retryCount := 3
	client := httpretry.NewDefaultClient(httpretry.WithRetryCount(retryCount))

	l, err := net.Listen("tcp", ":0")
	defer l.Close()

	if err != nil {
		panic(err)
	}
	connectionCount := 0
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			connectionCount++
			conn.Close()
		}
	}()

	res, err := client.Get("http://" + l.Addr().String())

	check.Contains(err.Error(), "EOF")
	check.Nil(res)
	check.Equal(retryCount+1, connectionCount)
}

func TestFirstRequestSuccessfull(t *testing.T) {
	check := assert.New(t)

	mockServer := httpmockserver.New(t)
	defer mockServer.Shutdown()

	mockServer.EXPECT().Get("/").Times(1).Response(200).StringBody("ok")

	client := httpretry.NewDefaultClient()

	res, err := client.Get(mockServer.URL())
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	d, _ := ioutil.ReadAll(res.Body)
	check.Equal("ok", string(d))

	mockServer.Finish()
}

/*func TestSecondRequestSuccessfull(t *testing.T) {
	check := assert.New(t)

	mockServer := httpmockserver.New(t)
	defer mockServer.Shutdown()

	mockServer.EXPECT().Get("/").Times(1).Response(200).StringBody("ok")

	client := httpretry.NewDefaultClient()

	res, err := client.Get(mockServer.URL())
	check.Nil(err)
	check.Equal(200, res.StatusCode)
	d, _ := ioutil.ReadAll(res.Body)
	check.Equal("ok", string(d))

	mockServer.Finish()
}
*/
// TODO: test with ssl (secure disabled, custom cert)
// TODO: test server times out request not possible with mockServer atm
