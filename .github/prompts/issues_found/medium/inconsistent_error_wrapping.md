# Title

Inconsistent Error Wrapping Patterns

## Problem

The codebase has inconsistent error wrapping patterns. While there's a custom `ProviderError` type with wrapping functionality, the implementation could lead to confusion when unwrapping errors and determining error types. The `Unwrap` function behaves differently from Go's standard error unwrapping patterns.

## Impact

**Severity: medium**

This issue impacts error handling reliability and debuggability:

- Inconsistent error wrapping makes error handling less predictable
- Custom unwrap behavior may break standard error handling patterns
- Some error context could be lost due to inconsistent wrapping

## Location

File: /workspaces/terraform-provider-power-platform/internal/customerrors/provider_error.go

## Code Issue

```go
func Unwrap(err error) error {
    if e, ok := err.(ProviderError); ok {
        return errors.Unwrap(e.Err)
    }
    return errors.Unwrap(err)
}

func WrapIntoProviderError(err error, errorCode ErrorCode, msg string) error {
    if err == nil {
        return ProviderError{
            Err:       fmt.Errorf("%s", msg),
            ErrorCode: errorCode,
        }
    }
    return ProviderError{
        Err:       fmt.Errorf("%s: [%w]", msg, err),
        ErrorCode: errorCode,
    }
}
```

## Fix

Implement standard error handling patterns:

```go
type ProviderError struct {
    ErrorCode ErrorCode
    Message   string
    cause     error
}

func (e *ProviderError) Error() string {
    if e.cause == nil {
        return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
    }
    return fmt.Sprintf("%s: %s: %v", e.ErrorCode, e.Message, e.cause)
}

func (e *ProviderError) Unwrap() error {
    return e.cause
}

func WrapIntoProviderError(err error, errorCode ErrorCode, msg string) error {
    return &ProviderError{
        ErrorCode: errorCode,
        Message:   msg,
        cause:     err,
    }
}

// Use errors.Is and errors.As for error checking:
// if errors.Is(err, ErrNotFound) { ... }
// var perr *ProviderError
// if errors.As(err, &perr) { ... }
```

Changes:

1. Implement standard `Unwrap()` method
2. Use pointer receiver for error interface
3. Separate message from underlying error
4. Follow Go 1.13+ error wrapping conventions
