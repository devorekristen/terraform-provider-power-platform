# Title

Unsafe API Retry Mechanism

## Problem

The API client's retry mechanism has potential security and reliability issues:

1. No maximum retry limit which could lead to infinite loops
2. Uses math/rand without proper seeding
3. Retries on 401 Unauthorized without token refresh
4. No exponential backoff for retries

## Impact

**Severity: critical**

This issue has severe impacts:

- Infinite retries could cause resource exhaustion
- Predictable retry timing due to unseeded random
- Potential credential exposure through unnecessary retries
- Possible service degradation from aggressive retry patterns

## Location

File: /workspaces/terraform-provider-power-platform/internal/api/client.go

## Code Issue

```go
var retryableStatusCodes = []int{
    http.StatusUnauthorized,        // 401 is retryable because the token may have expired.
    // ...
}

func DefaultRetryAfter() time.Duration {
    return time.Duration((rand.Intn(10) + 10)) * time.Second
}

func (client *Client) Execute(ctx context.Context, scopes []string, method, url string, headers http.Header, body any, acceptableStatusCodes []int, responseObj any) (*Response, error) {
    // ...
    for {  // Infinite retry loop
        // ...
        isRetryable := array.Contains(retryableStatusCodes, resp.HttpResponse.StatusCode)
        if !isRetryable {
            return resp, customerrors.NewUnexpectedHttpStatusCodeError(acceptableStatusCodes, resp.HttpResponse.StatusCode, resp.HttpResponse.Status, resp.BodyAsBytes)
        }
        waitFor := retryAfter(ctx, resp.HttpResponse)
        // ...
    }
}
```

## Fix

Implement a secure retry mechanism:

```go
type RetryConfig struct {
    MaxRetries      int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
    RandomizationFactor float64
}

func init() {
    // Seed the random number generator
    rand.Seed(time.Now().UnixNano())
}

func (client *Client) Execute(ctx context.Context, scopes []string, method, url string, headers http.Header, body any, acceptableStatusCodes []int, responseObj any) (*Response, error) {
    retries := 0
    retryConfig := client.getRetryConfig()
    
    for {
        if retries >= retryConfig.MaxRetries {
            return nil, fmt.Errorf("exceeded maximum retries (%d)", retryConfig.MaxRetries)
        }

        token, err := client.BaseAuth.GetTokenForScopes(ctx, scopes)
        if err != nil {
            return nil, err
        }

        resp, err := client.doRequest(/* ... */)
        
        // Handle 401 differently - try token refresh first
        if resp.StatusCode == http.StatusUnauthorized {
            if err := client.BaseAuth.RefreshToken(ctx); err != nil {
                return nil, err
            }
            continue
        }

        if !isRetryableError(resp.StatusCode) {
            return resp, err
        }

        // Calculate backoff with jitter
        backoff := retryConfig.InitialInterval * time.Duration(math.Pow(retryConfig.Multiplier, float64(retries)))
        if backoff > retryConfig.MaxInterval {
            backoff = retryConfig.MaxInterval
        }
        jitter := rand.Float64() * retryConfig.RandomizationFactor
        backoff = time.Duration(float64(backoff) * (1 + jitter))

        retries++
        
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(backoff):
            continue
        }
    }
}
```

Changes needed:

1. Add maximum retry limit
2. Implement exponential backoff with jitter
3. Properly seed random number generator
4. Handle 401s with token refresh
5. Add retry metrics/logging
6. Make retry configuration configurable
