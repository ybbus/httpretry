package httpretry

import (
	"errors"
	"net/http"
)

const (
	DefaultMaxRetryCount = 3
)

// NewRetryClient returns a standard http client with embedded retry functionality.
//
// You should not replace the client.Transport field, otherwise you will lose the retry functionality.
//
// If you need to set / change the client.Transport field you have to options:
//
// 1. create your own http client and use MakeRetryClient() function to enrich the client with retry functionality.
//   client := &http.Client{}
//   client.Transport = &http.Transport{ ... }
//   retryClient := MakeRetryClient(client)
// 2. retrieve the actual Roundtripper (casting checks omitted)
//   httpTransport := retryClient.Transport.(*httpretry.RetryRoundtripper).Next.(*http.Transport)
//   httpTransport.MaxIdleConns = 5
func NewRetryClient(opts ...Option) *http.Client {
	return MakeRetryClient(&http.Client{}, opts...)
}

func MakeRetryClient(client *http.Client, opts ...Option) *http.Client {
	// TODO: should client be returned? since it is no no client?
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

func ReplaceRoundtripper(client *http.Client, roundtripper http.RoundTripper) error {
	if client == nil {
		return errors.New("client must not be nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		r.Next = roundtripper
		return nil
	default: // also catches Transport == nil
		client.Transport = roundtripper
		return nil
	}
}

func ModifyEmbeddedTransport(client *http.Client, f func(transport *http.Transport)) error {
	if client == nil {
		return errors.New("cannot modify *http.Transport if client is nil")
	}

	switch r := client.Transport.(type) {
	case *http.Transport:
		f(r)
		return nil
	case *RetryRoundtripper:
		switch t := r.Next.(type) {
		case nil:
			return errors.New("embedded transport was nil")
		case *http.Transport:
			f(t)
			return nil
		default:
			return errors.New("embedded roundtripper is not of type *http.Transport")
		}
	case nil:
		return errors.New("transport was nil")
	default:
		return errors.New("transport is not of type *http.Transport")
	}
}

// TODO: should we return error if client is not enriched with retry?
func GetEmbeddedRoundtripper(client *http.Client) (http.RoundTripper, error) {
	if client == nil {
		return nil, errors.New("cannot get *http.Transport if client is nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		return r.Next, nil
	default: // also catches Transport == nil
		return client.Transport, nil
	}
}

// TODO: should we return error if client is not enriched with retry?
func GetEmbeddedTransport(client *http.Client) (*http.Transport, error) {
	if client == nil {
		return nil, errors.New("cannot get *http.Transport if client is nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		switch t := r.Next.(type) {
		case *http.Transport:
			return t, nil
		case nil:
			return nil, nil
		default:
			return nil, errors.New("embedded roundtripper is not of type *http.Transport")
		}
	case *http.Transport:
		return r, nil
	case nil:
		return nil, nil
	default: // also catches Transport == nil
		return nil, errors.New("roundtripper is not of type *http.Transport")
	}
}
