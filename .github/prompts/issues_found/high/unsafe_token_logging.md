# Title

Potentially Unsafe Token Logging

## Problem

The authentication code logs tokens in debug mode with partial masking, but the token value itself is passed around as a plain string. This could lead to accidental exposure of sensitive credentials in log files or memory dumps.

## Impact

**Severity: high**

This issue impacts security of authentication tokens:

- Debug logs could contain sensitive token information
- Tokens stored as plain strings can be exposed in memory dumps
- No secure string/credential type handling for sensitive values

## Location

File: /workspaces/terraform-provider-power-platform/internal/api/auth.go

## Code Issue

```go
func (client *Auth) GetTokenForScopes(ctx context.Context, scopes []string) (*string, error) {
    // ...
    tflog.Debug(ctx, fmt.Sprintf("Token acquired (expire: %s): **********", tokenExpiry))
    return &token, err.mcp_servers
}
```

## Fix

Implement secure string handling and improved logging:

```go
type SecureToken struct {
    value   *string
    expires time.Time
}

func (t *SecureToken) String() string {
    return "[REDACTED]"
}

func (client *Auth) GetTokenForScopes(ctx context.Context, scopes []string) (*SecureToken, error) {
    // ...
    return &SecureToken{
        value:   &token,
        expires: tokenExpiry,
    }, nil
}
```

Additionally:

1. Use a secure string type that zeros memory when freed
2. Avoid logging tokens even in debug mode
3. Add structured logging with proper redaction of sensitive values
