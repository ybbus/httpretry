package httpretry_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/ybbus/httpretry"
	"net/http"
	"testing"
	"time"
)

func TestNewDefaultClient(t *testing.T) {
	check := assert.New(t)

	t.Run("should be created", func(t *testing.T) {
		client := httpretry.NewDefaultClient()
		check.NotNil(client)
		check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)

		roundTripper := client.Transport.(*httpretry.RetryRoundtripper)
		check.IsType(&http.Transport{}, roundTripper.Next)
		check.Equal(5, roundTripper.MaxRetryCount)
		check.NotNil(roundTripper.CalculateBackoff)
		check.NotNil(roundTripper.ShouldRetry)
	})

	t.Run("should set custom options", func(t *testing.T) {
		called := 0
		client := httpretry.NewDefaultClient(
			httpretry.WithMaxRetryCount(2),
			httpretry.WithRetryPolicy(func(statusCode int, err error) bool {
				called++
				return false
			}),
			httpretry.WithBackoffPolicy(func(attemptCount int) time.Duration {
				called++
				return 1 * time.Second
			}),
		)

		rt := client.Transport.(*httpretry.RetryRoundtripper)
		rt.CalculateBackoff(1)
		rt.ShouldRetry(200, nil)

		// check if both custom policies were called
		check.Equal(2, called)
		check.Equal(2, rt.MaxRetryCount)
	})

}

func TestNewCustomClient(t *testing.T) {
	check := assert.New(t)

	t.Run("should create custom client", func(t *testing.T) {
		customTransport := &http.Transport{}
		httpClient := &http.Client{
			Transport: customTransport,
		}

		client := httpretry.NewCustomClient(httpClient)
		check.Equal(httpClient, client)
		check.NotNil(client)
		check.IsType(&httpretry.RetryRoundtripper{}, client.Transport)

		roundTripper := client.Transport.(*httpretry.RetryRoundtripper)
		check.Equal(customTransport, roundTripper.Next)
		check.Equal(5, roundTripper.MaxRetryCount)
		check.NotNil(roundTripper.CalculateBackoff)
		check.NotNil(roundTripper.ShouldRetry)
	})

	t.Run("should panic if nil client provided", func(t *testing.T) {
		defer func() {
			check.Equal("client must not be nil", recover())
		}()

		httpretry.NewCustomClient(nil)
	})
}

func TestGetOriginalRoundtripper(t *testing.T) {
	check := assert.New(t)

	t.Run("should return Roundtripper", func(t *testing.T) {
		client := httpretry.NewDefaultClient()
		check.NotNil(httpretry.GetOriginalRoundtripper(client))
		check.Equal(http.DefaultTransport, httpretry.GetOriginalRoundtripper(client))

	})

	t.Run("should return nil if no Roundtripper set", func(t *testing.T) {
		customClient := &http.Client{}
		check.Nil(httpretry.GetOriginalRoundtripper(customClient))
	})

	t.Run("should return Roundtripper of custom client", func(t *testing.T) {
		customClient := &http.Client{
			Transport: http.DefaultTransport,
		}
		check.NotNil(httpretry.GetOriginalRoundtripper(customClient))
		check.Equal(http.DefaultTransport, httpretry.GetOriginalRoundtripper(customClient))
	})

	t.Run("should panic on nil parameter", func(t *testing.T) {
		defer func() {
			check.Equal("client must not be nil", recover())
		}()
		check.Nil(httpretry.GetOriginalRoundtripper(nil))
	})
}

func TestReplaceOriginalRoundtripper(t *testing.T) {
	check := assert.New(t)

	t.Run("should replace Roundtripper", func(t *testing.T) {
		newRoundtripper := &http.Transport{
			TLSHandshakeTimeout: 123 * time.Second,
		}
		client := httpretry.NewDefaultClient()
		err := httpretry.ReplaceOriginalRoundtripper(client, newRoundtripper)
		check.Equal(newRoundtripper, httpretry.GetOriginalRoundtripper(client))
		check.Nil(err)
	})

	t.Run("should return error if client was nil", func(t *testing.T) {
		defer func() {
			check.Equal("client must not be nil", recover())
		}()
		newRoundtripper := &http.Transport{
			TLSHandshakeTimeout: 123 * time.Second,
		}
		httpretry.ReplaceOriginalRoundtripper(nil, newRoundtripper)
	})

	t.Run("should replace roundtripper even if not a retryclient", func(t *testing.T) {
		newRoundtripper := &http.Transport{
			TLSHandshakeTimeout: 123 * time.Second,
		}

		customClient := &http.Client{}
		err := httpretry.ReplaceOriginalRoundtripper(customClient, newRoundtripper)
		check.Equal(newRoundtripper, customClient.Transport)
		check.Nil(err)
	})
}

func TestGetOriginalTransport(t *testing.T) {
	check := assert.New(t)

	t.Run("should get transport", func(t *testing.T) {
		client := httpretry.NewDefaultClient()
		transport, err := httpretry.GetOriginalTransport(client)
		check.NotNil(transport)
		check.Nil(err)
		check.Equal(http.DefaultTransport, transport)
		check.IsType(&http.Transport{}, transport)
	})

	t.Run("should return error if embedded roundtripper is not of type http.Transport", func(t *testing.T) {
		client := httpretry.NewCustomClient(&http.Client{Transport: &CustomRoundtripper{}})
		transport, err := httpretry.GetOriginalTransport(client)
		check.Nil(transport)
		check.Contains(err.Error(), "is not of type *http.Transport")
	})

	t.Run("should return error if roundtripper of standard client is not of type http.Transport", func(t *testing.T) {
		client := &http.Client{Transport: &CustomRoundtripper{}}
		transport, err := httpretry.GetOriginalTransport(client)
		check.Nil(transport)
		check.Contains(err.Error(), "is not of type *http.Transport")
	})

	t.Run("should return nil if no transport available", func(t *testing.T) {
		client := &http.Client{}
		transport, err := httpretry.GetOriginalTransport(client)
		check.Nil(transport)
		check.Nil(err)

	})

	t.Run("should get transport of standard client", func(t *testing.T) {
		client := &http.Client{
			Transport: &http.Transport{},
		}
		transport, err := httpretry.GetOriginalTransport(client)
		check.NotNil(transport)
		check.Nil(err)

	})

	t.Run("should return error if client was nil", func(t *testing.T) {
		defer func() {
			check.Equal("client must not be nil", recover())
		}()
		httpretry.GetOriginalTransport(nil)
	})
}

func TestModifyOriginalTransport(t *testing.T) {
	check := assert.New(t)

	t.Run("should change embedded transport", func(t *testing.T) {
		transport := &http.Transport{TLSHandshakeTimeout: 123 * time.Second}
		customClient := &http.Client{Transport: transport}
		client := httpretry.NewCustomClient(customClient)

		err := httpretry.ModifyOriginalTransport(client, func(t *http.Transport) {
			check.Equal(123*time.Second, t.TLSHandshakeTimeout)
			t.TLSHandshakeTimeout = 321 * time.Second
		})

		transport2, _ := httpretry.GetOriginalTransport(client)
		check.Equal(321*time.Second, transport2.TLSHandshakeTimeout)
		check.Nil(err)
	})

	t.Run("should change transport of standard client", func(t *testing.T) {
		transport := &http.Transport{TLSHandshakeTimeout: 123 * time.Second}
		customClient := &http.Client{Transport: transport}

		err := httpretry.ModifyOriginalTransport(customClient, func(t *http.Transport) {
			check.Equal(123*time.Second, t.TLSHandshakeTimeout)
			t.TLSHandshakeTimeout = 321 * time.Second
		})

		transport2, _ := httpretry.GetOriginalTransport(customClient)
		check.Equal(321*time.Second, transport2.TLSHandshakeTimeout)
		check.Nil(err)
	})

	t.Run("should return error if client was nil", func(t *testing.T) {
		defer func() {
			check.Equal("client must not be nil", recover())
		}()

		httpretry.ModifyOriginalTransport(nil, func(t *http.Transport) {
			check.Equal(123*time.Second, t.TLSHandshakeTimeout)
			t.TLSHandshakeTimeout = 321 * time.Second
		})
	})

	t.Run("should return error if not http.Transport type", func(t *testing.T) {
		customClient := &http.Client{Transport: &CustomRoundtripper{}}
		client := httpretry.NewCustomClient(customClient)

		err := httpretry.ModifyOriginalTransport(client, func(t *http.Transport) {
			check.Equal(123*time.Second, t.TLSHandshakeTimeout)
			t.TLSHandshakeTimeout = 321 * time.Second
		})

		check.Contains(err.Error(), "not of type *http.Transport")
	})

	t.Run("should return error if standard client not http.Transport type", func(t *testing.T) {
		customClient := &http.Client{Transport: &CustomRoundtripper{}}

		err := httpretry.ModifyOriginalTransport(customClient, func(t *http.Transport) {})

		check.Contains(err.Error(), "not of type *http.Transport")
	})

	t.Run("should return error if embedded transport was nil", func(t *testing.T) {
		client := &http.Client{Transport: &httpretry.RetryRoundtripper{}}

		err := httpretry.ModifyOriginalTransport(client, func(t *http.Transport) {})

		check.Contains(err.Error(), "embedded transport was nil")
	})

	t.Run("should return error if transport of standard client was nil", func(t *testing.T) {
		client := &http.Client{}

		err := httpretry.ModifyOriginalTransport(client, func(t *http.Transport) {})

		check.Contains(err.Error(), "transport was nil")
	})
}

type CustomRoundtripper struct {
}

func (c *CustomRoundtripper) RoundTrip(*http.Request) (*http.Response, error) {
	panic("implement me")
}
