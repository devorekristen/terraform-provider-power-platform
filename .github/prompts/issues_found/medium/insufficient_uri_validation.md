# Title

Insufficient URI Validation and Construction

## Problem

The URI building functions lack input validation and could potentially generate invalid URIs if given malformed input. There's no validation of tenant/environment IDs before processing, and the string manipulation could fail with unexpected input lengths.

## Impact

**Severity: medium**

This issue impacts reliability and security:

- Malformed input could cause panics from string slicing
- Invalid URIs could be generated with unexpected input
- No validation of URI components before construction
- Potential for invalid host names due to missing character validation

## Location

File: /workspaces/terraform-provider-power-platform/internal/helpers/uri.go

## Code Issue

```go
func BuildEnvironmentHostUri(environmentId, powerPlatformUrl string) string {
    envId := strings.ReplaceAll(environmentId, "-", "")
    realm := string(envId[len(envId)-2:])  // Could panic with short strings
    envId = envId[:len(envId)-2]

    return fmt.Sprintf("%s.%s.environment.%s", envId, realm, powerPlatformUrl)
}
```

## Fix

Add proper validation and safer URI construction:

```go
func ValidateID(id string) error {
    if len(id) != 36 || !regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(id) {
        return fmt.Errorf("invalid ID format: %s", id)
    }
    return nil
}

func BuildEnvironmentHostUri(environmentId, powerPlatformUrl string) (string, error) {
    if err := ValidateID(environmentId); err != nil {
        return "", fmt.Errorf("invalid environment ID: %w", err)
    }
    
    if powerPlatformUrl == "" {
        return "", fmt.Errorf("powerPlatformUrl cannot be empty")
    }

    // Remove all hyphens and validate length
    envId := strings.ReplaceAll(environmentId, "-", "")
    if len(envId) < 2 {
        return "", fmt.Errorf("invalid environment ID length after processing")
    }

    // Extract realm and validate
    realm := string(envId[len(envId)-2:])
    if !regexp.MustCompile(`^[0-9a-fA-F]{2}$`).MatchString(realm) {
        return "", fmt.Errorf("invalid realm value: %s", realm)
    }

    // Get base ID and validate
    baseId := envId[:len(envId)-2]
    if !regexp.MustCompile(`^[0-9a-fA-F]+$`).MatchString(baseId) {
        return "", fmt.Errorf("invalid base ID value: %s", baseId)
    }

    // Validate final components are RFC compliant
    host := fmt.Sprintf("%s.%s.environment.%s", baseId, realm, powerPlatformUrl)
    if !regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(host) {
        return "", fmt.Errorf("generated host contains invalid characters: %s", host)
    }

    return host, nil
}
```

Changes needed:

1. Add proper input validation
2. Return errors instead of potentially invalid URIs
3. Validate all URI components
4. Add safety checks for string operations
5. Ensure generated URIs are RFC compliant
6. Add unit tests for edge cases
