# Title

Unsafe API Request Handling

## Problem

The REST API client implementation has several security and reliability concerns:

1. Panic used for control flow
2. No validation of user-provided URLs
3. Raw JSON strings passed without validation
4. Headers copied without sanitization
5. No request size limits

## Impact

**Severity: high**

This issue impacts security and reliability:

- Panics in production code could cause service disruption
- Injection vulnerabilities possible via unvalidated headers
- Potential DoS via large request bodies
- Possible SSRF via unvalidated URLs

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/rest/api_rest.go

## Code Issue

```go
func (client *client) ExecuteApiRequest(ctx context.Context, scope *string, url, method string, body *string, headers map[string]string, expectedStatusCodes []int) (*api.Response, error) {
    h := http.Header{}
    for k, v := range headers {
        h.Add(k, v)  // No header validation
    }

    if scope != nil {
        return client.Api.Execute(ctx, []string{*scope}, method, url, h, body, expectedStatusCodes, nil)
    }
    panic("scope or evironment_id must be provided")  // Using panic for control flow
}

func (client *client) SendOperation(ctx context.Context, operation *DataverseWebApiOperation) (types.Object, error) {
    url := operation.Url.ValueString()  // No URL validation
    method := operation.Method.ValueString()
    var body *string
    if operation.Body.ValueStringPointer() != nil {
        b := operation.Body.ValueString()  // No body validation/size limits
        body = &b
    }
    // ...
}
```

## Fix

Implement proper validation and error handling:

```go
const (
    maxRequestBodySize = 10 * 1024 * 1024  // 10MB
    maxHeaderSize = 8192  // 8KB
)

func validateRequestHeaders(headers map[string]string) error {
    for k, v := range headers {
        if len(k) > 256 || len(v) > maxHeaderSize {
            return fmt.Errorf("header too large: %s", k)
        }
        if !regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*+-.^_|~]+$`).MatchString(k) {
            return fmt.Errorf("invalid header name: %s", k)
        }
    }
    return nil
}

func validateRequestURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Validate scheme
    if u.Scheme != "https" {
        return fmt.Errorf("only HTTPS URLs are allowed")
    }
    
    // Additional domain validation as needed
    return nil
}

func (client *client) ExecuteApiRequest(ctx context.Context, scope *string, urlStr, method string, body *string, headers map[string]string, expectedStatusCodes []int) (*api.Response, error) {
    if scope == nil {
        return nil, fmt.Errorf("scope is required")
    }
    
    if err := validateRequestURL(urlStr); err != nil {
        return nil, err
    }
    
    if err := validateRequestHeaders(headers); err != nil {
        return nil, err
    }
    
    if body != nil && len(*body) > maxRequestBodySize {
        return nil, fmt.Errorf("request body exceeds maximum size of %d bytes", maxRequestBodySize)
    }
    
    // Validate method
    switch method {
    case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
        // Valid methods
    default:
        return nil, fmt.Errorf("unsupported HTTP method: %s", method)
    }
    
    h := http.Header{}
    for k, v := range headers {
        h.Add(k, v)
    }

    return client.Api.Execute(ctx, []string{*scope}, method, urlStr, h, body, expectedStatusCodes, nil)
}

func (client *client) SendOperation(ctx context.Context, operation *DataverseWebApiOperation) (types.Object, error) {
    // Validate operation
    if err := validateOperation(operation); err != nil {
        return types.ObjectUnknown(operationOutputType), err
    }

    // ... rest of implementation
}

func validateOperation(operation *DataverseWebApiOperation) error {
    if operation == nil {
        return fmt.Errorf("operation cannot be nil")
    }
    
    if operation.Body.ValueStringPointer() != nil {
        // Validate JSON if body is present
        if !json.Valid([]byte(operation.Body.ValueString())) {
            return fmt.Errorf("invalid JSON in request body")
        }
    }
    
    // Additional validation as needed
    return nil
}
```

Changes needed:

1. Replace panic with proper error handling
2. Add comprehensive URL validation
3. Implement request size limits
4. Add header validation and sanitization
5. Validate JSON request bodies
6. Add input validation for all operation parameters
7. Add unit tests for validation logic
