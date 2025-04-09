# Title

Unsafe Tenant Settings Reflection Usage and Filtering

## Problem

The tenant settings implementation has several reliability and security concerns:

1. Unsafe use of reflection without proper type checking
2. Potential memory leaks from reflection usage
3. No validation of settings before applying
4. Silent failures in settings filtering
5. Complex and hard to maintain reflection-based filtering

## Impact

**Severity: medium**

This issue impacts reliability and security:

- Type errors could cause panics in production
- Memory leaks possible from reflection usage
- Settings validation gaps could allow invalid states
- Silent failures could mask configuration errors
- Hard to maintain and debug reflection code

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/tenant_settings/api_tenant_settings.go

## Code Issue

```go
func filterDto(ctx context.Context, configuredSettings any, backendSettings any) any {
    // Unsafe type comparisons
    configuredType := reflect.TypeOf(configuredSettings)
    backendType := reflect.TypeOf(backendSettings)
    if configuredType != backendType {
        return nil  // Silent failure
    }

    // Complex reflection usage without proper error handling
    visibleFields := reflect.VisibleFields(configuredType)
    for fieldIndex, fieldInfo := range visibleFields {
        configuredFieldValue := configuredValue.Field(fieldIndex)
        backendFieldValue := backendValue.Field(fieldIndex)
        outputField := reflect.ValueOf(output).Elem().Field(fieldIndex)

        if !configuredFieldValue.IsNil() && !backendFieldValue.IsNil() && backendFieldValue.IsValid() && outputField.CanSet() {
            // Complex type switching without validation
            if configuredFieldValue.Kind() == reflect.Pointer && configuredFieldValue.Elem().Kind() == reflect.Struct {
                // Recursive reflection without depth limits
                outputStruct := filterDto(ctx, configuredFieldValue.Elem().Interface(), backendFieldValue.Elem().Interface())
                outputField.Set(reflect.ValueOf(outputStruct))
            }
            // ...
        }
    }
}
```

## Fix

Implement type-safe settings handling:

```go
// Define strong types for settings validation
type SettingsValidator interface {
    Validate() error
    FilterFields(backend interface{}) error
}

// Type-safe settings struct
type TenantSettings struct {
    PowerPlatform *PowerPlatformSettings `json:"powerPlatform,omitempty"`
}

type PowerPlatformSettings struct {
    Governance *GovernanceSettings `json:"governance,omitempty"`
}

type GovernanceSettings struct {
    EnvironmentRoutingTargetSecurityGroupId   *string `json:"environmentRoutingTargetSecurityGroupId,omitempty"`
    EnvironmentRoutingTargetEnvironmentGroupId *string `json:"environmentRoutingTargetEnvironmentGroupId,omitempty"`
}

// Implement validation interface
func (s *TenantSettings) Validate() error {
    if s == nil {
        return errors.New("settings cannot be nil")
    }

    if s.PowerPlatform != nil {
        if err := s.PowerPlatform.Validate(); err != nil {
            return fmt.Errorf("invalid power platform settings: %w", err)
        }
    }
    return nil
}

// Type-safe filtering
func (s *TenantSettings) FilterFields(backend interface{}) error {
    backendSettings, ok := backend.(*TenantSettings)
    if !ok {
        return fmt.Errorf("expected *TenantSettings, got %T", backend)
    }

    // Create filtered copy
    filtered := &TenantSettings{}
    
    // Only copy configured fields
    if s.PowerPlatform != nil {
        filtered.PowerPlatform = &PowerPlatformSettings{}
        if err := s.PowerPlatform.FilterFields(backendSettings.PowerPlatform, filtered.PowerPlatform); err != nil {
            return fmt.Errorf("failed to filter power platform settings: %w", err)
        }
    }

    return nil
}

// Settings client with validation
type TenantSettingsClient struct {
    api *api.Client
}

func (c *TenantSettingsClient) UpdateSettings(ctx context.Context, settings *TenantSettings) (*TenantSettings, error) {
    // Validate before update
    if err := settings.Validate(); err != nil {
        return nil, fmt.Errorf("invalid settings: %w", err)
    }

    // Get current settings
    current, err := c.GetSettings(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get current settings: %w", err)
    }

    // Filter settings
    if err := settings.FilterFields(current); err != nil {
        return nil, fmt.Errorf("failed to filter settings: %w", err)
    }

    // Apply update with retry and proper error handling
    var result TenantSettings
    resp, err := c.api.Execute(ctx, nil, http.MethodPost, c.buildUpdateURL(), nil, settings, 
        []int{http.StatusOK}, &result)
    if err != nil {
        return nil, fmt.Errorf("failed to update settings: %w", err)
    }

    // Validate response
    if err := result.Validate(); err != nil {
        return nil, fmt.Errorf("invalid response from server: %w", err)
    }

    return &result, nil
}
```

Changes needed:

1. Replace reflection with type-safe structs
2. Add proper validation interfaces
3. Implement explicit type filtering
4. Add error handling for all operations
5. Add proper logging
6. Add retry handling
7. Add settings validation
8. Add test coverage
9. Document field dependencies
