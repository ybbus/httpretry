# httpretry
Enriches standard go http client with retry functionality using Roundtripper.

The advantage of this library is that it uses the default http.Client
Therefore you can provide it to any library that accepts the standard http.Client.
This give you the possibility to add resilience to a lot of http based go libraries (or just use it as standalone retryable http client.

## Information

This library is in alpha state and under heavy development.

It would help a lot if you give it a try and provide your feedback via issues.

### Quickstart

To get a standard http client with retry:

```golang
client := NewRetryableClient()
res, err := client.Get("https://someurl.com") // if not successful, will retry 3 times
```
**Important**: do not set / reset the client.Transport field after you created the retryable client.

If you need to customize your http.Client, do this before wrapping the Retryable Client around it:
```golang
// create and configure the client as you like (setting Transport, Timeouts etc)
customHttpClient := &http.Client{}

// then enrich with retry functionality
// this will return the exact same client
client := MakeRetryable(cumstomHttpClient)

// use the client as usual
res, err := client.Get("https://someurl.com") // if not successful, will retry 3 times

// in the rare case that you need to change Transport settings afterwards:
httpTransport := client.Transport.(*httpretry.RetryRoundtripper).Next.(*http.Transport)
httpTransport.MaxIdleConns = 5

```

### Customize retry settings

```golang
client := NewRetryableClient(
    // retry 5 times
    WithMaxRetryCount(5),
    // retry on status >= 500 or if err != nil
    WithRetryPolicy(func(statusCode int, err error) bool {
      return err != nil || statusCode >= 500
    }),
    // every retry should wait one more second
    WithBackoffPolicy(func(attemptNum int) time.Duration {
      return time.Duration(attemptNum) * 1 * time.Second
    }),
)

res, err := client.Get("https://someurl.com") // if not successful, will retry 3 times
```
