# Title

Inconsistent and Potentially Unsafe Logging Practices

## Problem

The logging implementation has several security and reliability concerns:

1. No central logging configuration
2. Inconsistent log level usage
3. Potential sensitive data exposure in debug logs
4. Missing log sanitization framework
5. No structured logging helpers

## Impact

**Severity: medium**

This issue impacts security and maintainability:

- Inconsistent logging could expose sensitive data
- Debug logs may contain PII or credentials
- No standardized sensitive data handling
- Difficult to maintain logging consistency
- Hard to filter sensitive information

## Location

Throughout codebase, particularly in:

- internal/api/*.go
- internal/services/*/*.go

## Code Issue

```go
// Various locations show inconsistent logging:
tflog.Debug(ctx, fmt.Sprintf("Token acquired (expire: %s): **********", tokenExpiry))
tflog.Debug(ctx, fmt.Sprintf("[GetTokenForScope] Getting token for scope: '%s'", strings.Join(scopes, ",")))
tflog.Debug(ctx, fmt.Sprintf("READ: %s with Id: %s", r.FullTypeName(), billing.Id))

// No consistent pattern for sensitive data
tflog.Debug(ctx, fmt.Sprintf("Field: %s", fieldInfo.Name))

// Raw value logging without sanitization
tflog.Debug(ctx, fmt.Sprintf("Skipping unknown field type %s", configuredFieldValue.Kind()))
```

## Fix

Implement centralized logging framework:

```go
package logging

import (
    "context"
    "encoding/json"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Sensitive data types
type SensitiveValue interface {
    Redact() string
}

type SensitiveString struct {
    value string
}

func (s *SensitiveString) Redact() string {
    if s == nil || s.value == "" {
        return ""
    }
    return "[REDACTED]"
}

// Log levels
type LogLevel int

const (
    LogTrace LogLevel = iota
    LogDebug
    LogInfo
    LogWarn
    LogError
)

// Structured log entry
type LogEntry struct {
    Level   LogLevel
    Message string
    Fields  map[string]interface{}
}

// Logger with sanitization
type SecureLogger struct {
    ctx context.Context
    sanitizers []SanitizeFunc
}

type SanitizeFunc func(interface{}) interface{}

func NewSecureLogger(ctx context.Context) *SecureLogger {
    return &SecureLogger{
        ctx: ctx,
        sanitizers: []SanitizeFunc{
            sanitizeCredentials,
            sanitizePII,
            sanitizeTokens,
        },
    }
}

func (l *SecureLogger) Debug(msg string, fields ...interface{}) {
    entry := l.sanitize(LogDebug, msg, fields...)
    tflog.Debug(l.ctx, entry.Message, entry.Fields)
}

func (l *SecureLogger) Info(msg string, fields ...interface{}) {
    entry := l.sanitize(LogInfo, msg, fields...)
    tflog.Info(l.ctx, entry.Message, entry.Fields)
}

func (l *SecureLogger) Error(msg string, err error, fields ...interface{}) {
    entry := l.sanitize(LogError, msg, fields...)
    entry.Fields["error"] = err.Error()
    tflog.Error(l.ctx, entry.Message, entry.Fields)
}

func (l *SecureLogger) sanitize(level LogLevel, msg string, fields ...interface{}) LogEntry {
    entry := LogEntry{
        Level:   level,
        Message: msg,
        Fields:  make(map[string]interface{}),
    }

    // Convert fields to map
    if len(fields)%2 == 0 {
        for i := 0; i < len(fields); i += 2 {
            key, ok := fields[i].(string)
            if !ok {
                continue
            }
            entry.Fields[key] = fields[i+1]
        }
    }

    // Apply all sanitizers
    for key, value := range entry.Fields {
        for _, sanitize := range l.sanitizers {
            entry.Fields[key] = sanitize(value)
        }
    }

    return entry
}

// Sanitizers
func sanitizeCredentials(value interface{}) interface{} {
    // Check common credential field names
    switch v := value.(type) {
    case string:
        if isCredentialField(v) {
            return "[REDACTED]"
        }
    }
    return value
}

func sanitizePII(value interface{}) interface{} {
    // Check for PII patterns (email, phone, etc)
    switch v := value.(type) {
    case string:
        if containsPII(v) {
            return "[REDACTED PII]"
        }
    }
    return value
}

func sanitizeTokens(value interface{}) interface{} {
    // Check for token patterns
    switch v := value.(type) {
    case string:
        if isTokenPattern(v) {
            return "[REDACTED TOKEN]"
        }
    }
    return value
}

// Helper functions
func isCredentialField(name string) bool {
    sensitiveFields := map[string]bool{
        "password": true,
        "secret":   true,
        "key":      true,
        "token":    true,
        "cert":     true,
    }
    return sensitiveFields[name]
}

func containsPII(value string) bool {
    // Check for email pattern
    if strings.Contains(value, "@") {
        return true
    }
    // Add other PII patterns
    return false
}

func isTokenPattern(value string) bool {
    // Check common token formats (JWT, OAuth, etc)
    if strings.HasPrefix(value, "ey") && strings.Count(value, ".") == 2 {
        return true
    }
    return false
}

// Usage example
func ExampleUsage(ctx context.Context) {
    logger := NewSecureLogger(ctx)
    
    // Safe logging
    logger.Debug("Processing request",
        "requestId", "123",
        "user", "john@example.com",  // Will be redacted
        "token", "eyJ0...",         // Will be redacted
    )
    
    // Error logging
    if err := doSomething(); err != nil {
        logger.Error("Operation failed", err,
            "operation", "create",
            "status", 500,
        )
    }
}
```

Changes needed:

1. Create central logging package
2. Add sensitive data detection
3. Add log sanitization
4. Add structured logging helpers
5. Add consistent log levels
6. Add PII detection
7. Add token pattern detection
8. Add logging documentation
9. Update existing log calls
10. Add log sanitization tests
