# httpretry (alpha state)
Enriches the standard go http client with retry functionality using a wrapper around the Roundtripper interface.

The advantage of this library is that it makes use of the default http.Client.
This means you can provide it to any library that accepts the go standard http.Client.
This give you the possibility to add resilience to a lot of http based go libraries with just a single line of code.
Of course it can also be used as standalone http client in your own projects.

## Information

This library is in alpha state and under heavy development.

It would help a lot if you give it a try and provide your feedback via issues / PRs.

### Quickstart

To get a standard http client with retry functionality:

```golang
client := NewClient()
// use this as usual when working with http.Client
```
This single line of code returns a default http.Client that uses an exponential backoff and sends up to 5 retries if the request was not successful.
Requests will be retried if the error seems to be temporary or the requests returns a status code that may change on retry (e.g. GetwayTimeout).

### Modify / customize the Roundtripper (http.Transport)
Since httpretry wraps the actual Roundtripper of the http.Client, you should not try to replace / modify the client.Transport field after creation.

You either configure the http.Client upfront and then "make" it retryable like in this code:
```golang
customHttpClient := &http.Client{}
customHttpClient.Transport = &http.Transport{...}

retryClient := MakeRetryable(cumstomHttpClient)
```

or you use one of the available helper functions to gain access to the underlying Roundtripper / http.Transport:

```golang
// replaces the embedded roundtripper
ReplaceRoundtripper(client *http.Client, roundtripper http.RoundTripper)

// modifies the embedded http.Transport by providing a function that receives the client.Transport as parameter
// (returns an error if the embedded Roundtripper is not of type http.Transport)
ModifyEmbeddedTransport(client *http.Client, f func(transport *http.Transport)) error

// returns the embedded Roundtripper
GetEmbeddedRoundtripper(client *http.Client) (http.RoundTripper, error)

// returns the embedded Roundtripper as http.Transport if it is of that type
GetEmbeddedTransport(client *http.Client) (*http.Transport, error)
```

### Customize retry settings

You may provide your own Backoff- and RetryPolicy.

```golang
client := NewRetryClient(
    // retry up to 5 times
    WithMaxRetryCount(5),
    // retry on status >= 500, if err != nil, or if response was nil (status == 0)
    WithRetryPolicy(func(statusCode int, err error) bool {
      return err != nil || statusCode >= 500 || statusCode == 0
    }),
    // every retry should wait one more second
    WithBackoffPolicy(func(attemptNum int) time.Duration {
      return time.Duration(attemptNum+1) * 1 * time.Second
    }),
)
```
