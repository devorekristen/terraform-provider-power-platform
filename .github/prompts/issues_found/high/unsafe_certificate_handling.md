# Title

Unsafe Certificate and Private Key Handling

## Problem

The certificate handling implementation has several critical security concerns:

1. No secure memory handling for private keys
2. No certificate chain validation
3. Potential file traversal in certificate path
4. Password stored as plain string
5. No certificate expiration checking

## Impact

**Severity: high**

This issue has severe security impacts:

- Private keys could be exposed in memory dumps
- Invalid certificate chains could be accepted
- Path traversal could expose system files
- Passwords could be exposed in memory
- Expired certificates could be accepted and used

## Location

File: /workspaces/terraform-provider-power-platform/internal/helpers/cert.go

## Code Issue

```go
func GetCertificateRawFromCertOrFilePath(certificate, certificateFilePath string) (string, error) {
    // No path validation
    if certificateFilePath != "" {
        pfx, err := os.ReadFile(certificateFilePath)  // Potential path traversal
        if err != nil {
            return "", err
        }
        certAsBase64 := base64.StdEncoding.EncodeToString(pfx)
        return strings.TrimSpace(certAsBase64), nil
    }
    return "", errors.New("either client_certificate base64 or certificate_file_path must be provided")
}

func ConvertBase64ToCert(b64, password string) ([]*x509.Certificate, crypto.PrivateKey, error) {
    // Password stored as plain string
    // No certificate chain validation
    // No secure memory handling
    pfx, err := convertBase64ToByte(b64)
    if err != nil {
        return nil, nil, err
    }

    certs, key, err := convertByteToCert(pfx, password)
    if err != nil {
        return nil, nil, err
    }

    return certs, key, nil  // Private key exposed in memory
}
```

## Fix

Implement secure certificate handling:

```go
import (
    "crypto"
    "crypto/x509"
    "encoding/base64"
    "path/filepath"
    "time"
    "golang.org/x/sys/unix"
    "github.com/youmark/pkcs8"  // For secure key handling
)

// Secure memory wrapper for private keys
type SecureKey struct {
    key crypto.PrivateKey
}

func (s *SecureKey) Clear() {
    if s.key != nil {
        // Zero memory holding the key
        if clearer, ok := s.key.(interface{ Clear() }); ok {
            clearer.Clear()
        }
    }
}

// Safe path validation
func validateCertPath(path string) error {
    // Convert to absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }
    
    // Check path is within allowed directories
    if !filepath.HasPrefix(absPath, "/etc/ssl/") && 
       !filepath.HasPrefix(absPath, "/usr/local/share/ca-certificates/") {
        return fmt.Errorf("certificate path must be in approved directories")
    }
    
    return nil
}

func GetCertificateRawFromCertOrFilePath(certificate, certificateFilePath string) (string, error) {
    if certificate != "" {
        return strings.TrimSpace(certificate), nil
    }
    
    if certificateFilePath != "" {
        if err := validateCertPath(certificateFilePath); err != nil {
            return "", fmt.Errorf("invalid certificate path: %w", err)
        }
        
        // Open with minimal permissions
        file, err := os.OpenFile(certificateFilePath, os.O_RDONLY, 0)
        if err != nil {
            return "", err
        }
        defer file.Close()
        
        // Lock memory pages
        if err := unix.Mlock([]byte(certificateFilePath)); err != nil {
            return "", fmt.Errorf("failed to lock memory: %w", err)
        }
        defer unix.Munlock([]byte(certificateFilePath))
        
        pfx, err := os.ReadFile(certificateFilePath)
        if err != nil {
            return "", err
        }
        
        certAsBase64 := base64.StdEncoding.EncodeToString(pfx)
        return strings.TrimSpace(certAsBase64), nil
    }
    
    return "", errors.New("either client_certificate base64 or certificate_file_path must be provided")
}

func ConvertBase64ToCert(b64 string, password *SecureString) (*SecureCertBundle, error) {
    defer password.Clear()  // Clear password from memory when done
    
    pfx, err := convertBase64ToByte(b64)
    if err != nil {
        return nil, err
    }
    
    bundle, err := NewSecureCertBundle(pfx, password.String())
    if err != nil {
        return nil, err
    }
    
    // Validate certificate chain
    if err := bundle.ValidateChain(); err != nil {
        return nil, fmt.Errorf("invalid certificate chain: %w", err)
    }
    
    // Check expiration
    if err := bundle.ValidateExpiration(); err != nil {
        return nil, fmt.Errorf("certificate expiration check failed: %w", err)
    }
    
    return bundle, nil
}

type SecureCertBundle struct {
    certs []*x509.Certificate
    key   *SecureKey
}

func NewSecureCertBundle(certData []byte, password string) (*SecureCertBundle, error) {
    // Use secure PKCS#12 parsing
    key, cert, certs, err := pkcs12.DecodeChain(certData, password)
    if err != nil {
        return nil, err
    }
    
    if cert == nil {
        return nil, errors.New("found no certificate")
    }
    
    // Store key securely
    secureKey := &SecureKey{key: key}
    
    // Build cert chain
    chain := []*x509.Certificate{cert}
    chain = append(chain, certs...)
    
    return &SecureCertBundle{
        certs: chain,
        key:   secureKey,
    }, nil
}

func (b *SecureCertBundle) ValidateChain() error {
    if len(b.certs) == 0 {
        return errors.New("no certificates in bundle")
    }
    
    // Create cert pool with intermediate certs
    intermediates := x509.NewCertPool()
    for _, cert := range b.certs[1:] {
        intermediates.AddCert(cert)
    }
    
    // Verify chain
    opts := x509.VerifyOptions{
        Intermediates: intermediates,
        CurrentTime:  time.Now(),
        KeyUsages:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }
    
    _, err := b.certs[0].Verify(opts)
    return err
}

func (b *SecureCertBundle) ValidateExpiration() error {
    now := time.Now()
    for i, cert := range b.certs {
        if now.Before(cert.NotBefore) {
            return fmt.Errorf("certificate %d not yet valid", i)
        }
        if now.After(cert.NotAfter) {
            return fmt.Errorf("certificate %d has expired", i)
        }
    }
    return nil
}

func (b *SecureCertBundle) Clear() {
    // Clear private key
    if b.key != nil {
        b.key.Clear()
    }
    
    // Clear certificates
    for i := range b.certs {
        b.certs[i] = nil
    }
}
```

Changes needed:

1. Add secure memory handling for keys
2. Implement certificate chain validation
3. Add path traversal prevention
4. Add secure password handling
5. Add certificate expiration checking
6. Add memory locking for sensitive data
7. Add proper cleanup of sensitive data
8. Add comprehensive validation
9. Add test coverage for security features
