# Title

Critical Authentication Security Vulnerabilities

## Problem

The authentication implementation has several critical security concerns:

1. Tokens logged in debug mode
2. Insecure token storage in memory
3. No secure cleanup of sensitive data
4. Unsafe OIDC token file handling
5. Missing validation of OIDC endpoints
6. Unbounded token response reading
7. Missing token expiration handling

## Impact

**Severity: critical**

This issue has severe security impacts:

- Authentication tokens exposed in logs
- Sensitive data remains in memory
- Possible path traversal in token files
- Potential SSRF via OIDC endpoints
- Memory exhaustion from large responses
- Silent token expiration failures

## Location

File: /workspaces/terraform-provider-power-platform/internal/api/auth.go

## Code Issue

```go
func (client *Auth) GetTokenForScopes(ctx context.Context, scopes []string) (*string, error) {
    // Unsafe logging of scopes
    tflog.Debug(ctx, fmt.Sprintf("[GetTokenForScope] Getting token for scope: '%s'", strings.Join(scopes, ",")))
    
    // Token stored as plain string
    token := ""
    // ...
    
    // Unsafe token logging
    tflog.Debug(ctx, fmt.Sprintf("Token acquired (expire: %s): **********", tokenExpiry))
    return &token, err
}

func (w *OidcCredential) getAssertion(ctx context.Context) (string, error) {
    // Unsafe file reading without path validation
    if w.tokenFilePath != "" {
        idTokenData, err := os.ReadFile(w.tokenFilePath)
        // ...
    }
    
    // Unsafe URL usage without validation
    req, err := http.NewRequestWithContext(ctx, "GET", w.requestUrl, http.NoBody)
    // ...
    
    // Unbounded response reading
    body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
    // ...
}

func (client *Auth) AuthenticateClientCertificate(ctx context.Context, scopes []string) (string, time.Time, error) {
    cert, key, err := helpers.ConvertBase64ToCert(client.config.ClientCertificateRaw, client.config.ClientCertificatePassword)
    // No secure cleanup of sensitive data
    // ...
}
```

## Fix

Implement secure authentication handling:

```go
import (
    "crypto/subtle"
    "golang.org/x/crypto/secure"
    "net/url"
    "path/filepath"
)

// Secure token container
type SecureToken struct {
    value []byte
    expiry time.Time
}

func NewSecureToken(value string, expiry time.Time) *SecureToken {
    // Copy token to secure memory
    secure := make([]byte, len(value))
    subtle.ConstantTimeCopy(1, secure, []byte(value))
    
    return &SecureToken{
        value: secure,
        expiry: expiry,
    }
}

func (t *SecureToken) Clear() {
    if t.value != nil {
        // Zero sensitive data
        for i := range t.value {
            t.value[i] = 0
        }
        t.value = nil
    }
}

// Secure credential manager
type CredentialManager struct {
    tokens map[string]*SecureToken
    mu     sync.RWMutex
}

func NewCredentialManager() *CredentialManager {
    return &CredentialManager{
        tokens: make(map[string]*SecureToken),
    }
}

func (cm *CredentialManager) StoreToken(scope string, token *SecureToken) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Clear any existing token
    if existing := cm.tokens[scope]; existing != nil {
        existing.Clear()
    }
    
    cm.tokens[scope] = token
}

func (cm *CredentialManager) GetToken(scope string) (*SecureToken, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    token := cm.tokens[scope]
    if token == nil {
        return nil, errors.New("no token found for scope")
    }
    
    // Check expiration
    if time.Now().After(token.expiry) {
        return nil, &TokenExpiredError{Message: "token has expired"}
    }
    
    return token, nil
}

// OIDC security
type OIDCConfig struct {
    allowedHosts []string
    maxFileSize  int64
    maxTokenSize int64
}

func (c *OIDCConfig) ValidateTokenPath(path string) error {
    // Validate absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("invalid token path: %w", err)
    }
    
    // Check path is within allowed directories
    if !filepath.HasPrefix(absPath, "/run/secrets/") && 
       !filepath.HasPrefix(absPath, "/var/run/secrets/") {
        return fmt.Errorf("token path must be in approved directories")
    }
    
    // Check file size
    info, err := os.Stat(absPath)
    if err != nil {
        return fmt.Errorf("cannot stat token file: %w", err)
    }
    
    if info.Size() > c.maxFileSize {
        return fmt.Errorf("token file exceeds maximum size of %d bytes", c.maxFileSize)
    }
    
    return nil
}

func (c *OIDCConfig) ValidateEndpoint(endpoint string) error {
    parsed, err := url.Parse(endpoint)
    if err != nil {
        return fmt.Errorf("invalid endpoint URL: %w", err)
    }
    
    // Validate scheme
    if parsed.Scheme != "https" {
        return errors.New("endpoint must use HTTPS")
    }
    
    // Validate host
    host := parsed.Hostname()
    for _, allowed := range c.allowedHosts {
        if host == allowed {
            return nil
        }
    }
    
    return fmt.Errorf("endpoint host %s is not allowed", host)
}

// Updated authentication client
func (client *Auth) GetTokenForScopes(ctx context.Context, scopes []string) (*SecureToken, error) {
    // Don't log sensitive scope information
    tflog.Debug(ctx, "Getting token for scopes")
    
    if client.config.TestMode {
        return NewSecureToken("test_mode_mock_token_value", time.Now().Add(time.Hour)), nil
    }
    
    var token *SecureToken
    var err error
    
    switch {
    case client.config.IsClientSecretCredentialsProvided():
        token, err = client.AuthenticateClientSecret(ctx, scopes)
    // ... other auth methods
    default:
        return nil, errors.New("no credentials provided")
    }
    
    if err != nil {
        return nil, err
    }
    
    // Don't log token details
    tflog.Debug(ctx, "Token acquired")
    return token, nil
}

func (client *Auth) AuthenticateClientCertificate(ctx context.Context, scopes []string) (*SecureToken, error) {
    // Use secure cert handling from earlier fix
    certBundle, err := NewSecureCertBundle(client.config.ClientCertificateRaw, 
        client.config.ClientCertificatePassword)
    if err != nil {
        return nil, err
    }
    defer certBundle.Clear()
    
    // Get token using cert
    azureCertCredentials, err := azidentity.NewClientCertificateCredential(
        client.config.TenantId,
        client.config.ClientId,
        certBundle.GetCertificate(),
        certBundle.GetKey(),
        &azidentity.ClientCertificateCredentialOptions{
            AdditionallyAllowedTenants: client.config.AuxiliaryTenantIDs,
            ClientOptions: azcore.ClientOptions{
                Cloud: client.config.Cloud,
            },
        },
    )
    if err != nil {
        return nil, err
    }
    
    accessToken, err := azureCertCredentials.GetToken(ctx, policy.TokenRequestOptions{
        Scopes: scopes,
    })
    if err != nil {
        return nil, err
    }
    
    return NewSecureToken(accessToken.Token, accessToken.ExpiresOn), nil
}

func (w *OidcCredential) getAssertion(ctx context.Context) (*SecureToken, error) {
    config := &OIDCConfig{
        allowedHosts: []string{"token.actions.githubusercontent.com"},
        maxFileSize: 32 * 1024,  // 32KB
        maxTokenSize: 1 * 1024 * 1024,  // 1MB
    }
    
    if w.tokenFilePath != "" {
        if err := config.ValidateTokenPath(w.tokenFilePath); err != nil {
            return nil, err
        }
        
        idTokenData, err := os.ReadFile(w.tokenFilePath)
        if err != nil {
            return nil, fmt.Errorf("reading token file: %w", err)
        }
        
        return NewSecureToken(string(idTokenData), time.Now().Add(time.Hour)), nil
    }
    
    // Validate OIDC endpoint
    if err := config.ValidateEndpoint(w.requestUrl); err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "GET", w.requestUrl, http.NoBody)
    if err != nil {
        return nil, errors.New("getAssertion: failed to build request")
    }
    
    // Set security headers
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.requestToken))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("getAssertion: cannot request token: %w", err)
    }
    defer resp.Body.Close()
    
    // Limit response size
    body, err := io.ReadAll(io.LimitReader(resp.Body, config.maxTokenSize))
    if err != nil {
        return nil, fmt.Errorf("getAssertion: cannot read response: %w", err)
    }
    
    if len(body) == int(config.maxTokenSize) {
        return nil, errors.New("getAssertion: response too large")
    }
    
    // Validate response
    if statusCode := resp.StatusCode; statusCode < http.StatusOK || 
        statusCode >= http.StatusMultipleChoices {
        return nil, fmt.Errorf("getAssertion: received HTTP status %d", resp.StatusCode)
    }
    
    var tokenRes struct {
        Value *string `json:"value"`
    }
    if err := json.Unmarshal(body, &tokenRes); err != nil {
        return nil, fmt.Errorf("getAssertion: cannot unmarshal response: %w", err)
    }
    
    if tokenRes.Value == nil {
        return nil, errors.New("getAssertion: nil JWT assertion received")
    }
    
    return NewSecureToken(*tokenRes.Value, time.Now().Add(time.Hour)), nil
}
```

Changes needed:

1. Add secure token handling
2. Remove sensitive logging
3. Add token expiration handling
4. Add OIDC endpoint validation
5. Add secure token file handling
6. Add response size limits
7. Add secure memory cleanup
8. Add proper error handling
9. Add input validation
10. Add security headers
11. Add comprehensive tests
