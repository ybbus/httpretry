package httpretry

import (
	"crypto/x509"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

type MyTemporaryError struct {
	IsTemp bool
}

func (e *MyTemporaryError) Error() string {
	return "my error"
}

func (e *MyTemporaryError) Temporary() bool {
	return e.IsTemp
}

func TestDefaultRetryPolicy(t *testing.T) {
	check := assert.New(t)

	tests := []struct {
		Description  string
		StatusCodeIn int
		ErrorIn      error
		Expect       bool
	}{
		{
			Description:  "Should not retry on OK requests",
			StatusCodeIn: http.StatusOK,
			ErrorIn:      nil,
			Expect:       false,
		},
		{
			Description:  "Should not retry on BadRequest",
			StatusCodeIn: http.StatusBadRequest,
			ErrorIn:      nil,
			Expect:       false,
		},
		{
			Description:  "Should retry on InternalError",
			StatusCodeIn: http.StatusInternalServerError,
			ErrorIn:      nil,
			Expect:       true,
		},
		{
			Description:  "Should retry on Temporary error",
			StatusCodeIn: http.StatusInternalServerError,
			ErrorIn:      &MyTemporaryError{IsTemp: true},
			Expect:       true,
		},
		{
			Description:  "Should not retry on URL parse error",
			StatusCodeIn: 0,
			ErrorIn: &url.Error{
				Op:  "parse",
				URL: "",
			},
			Expect: false,
		},
		{
			Description:  "Should not retry on Certificate error",
			StatusCodeIn: 0,
			ErrorIn: &url.Error{
				Op:  "Get",
				URL: "https://some-non-existing-url.com",
				Err: x509.UnknownAuthorityError{},
			},
			Expect: false,
		},
		{
			// should not happen
			Description:  "Should retry if there was no error but also no response",
			StatusCodeIn: 0,
			ErrorIn:      nil,
			Expect:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.Description, func(t *testing.T) {
			shouldRetry := DefaultRetryPolicy(test.StatusCodeIn, test.ErrorIn)
			check.Equal(test.Expect, shouldRetry)
		})
	}

}
