# Title

UUID Type Safety Improvements Needed

## Problem

The UUID type implementation, while functional, has some type safety and efficiency concerns:

1. No canonical format enforcement
2. Redundant UUID parsing in validation functions
3. Missing helper methods for common operations
4. Type alias could lead to confusion

## Impact

**Severity: low**

The issues impact code maintainability and efficiency:

- Multiple UUID parsing operations during validation
- No guaranteed canonical format for string representation
- Potential confusion around UUID vs UUIDValue type alias
- Missing convenience methods for UUID manipulation

## Location

File: /workspaces/terraform-provider-power-platform/internal/customtypes/uuid.go
File: /workspaces/terraform-provider-power-platform/internal/customtypes/uuid_value.go

## Code Issue

```go
// Type alias could be confusing
type UUID = UUIDValue

// Multiple parsing operations
func (v UUIDValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
    // ...
    oldUUID, err := uuid.ParseUUID(v.ValueString())
    // ...
    newUUID, err := uuid.ParseUUID(newValue.ValueString())
    // ...
}

func (v UUIDValue) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
    // ...
    if _, err := uuid.ParseUUID(v.ValueString()); err != nil {
        // ...
    }
}
```

## Fix

Implement improved type safety and efficiency:

```go
// Separate type and alias names
type UUIDValue struct {
    basetypes.StringValue
    parsed *[16]byte  // Cache parsed value
}

// Clear type alias
type UUID interface {
    IsValid() bool
    String() string
    Bytes() [16]byte
}

func (v *UUIDValue) ensureParsed() error {
    if v.parsed != nil {
        return nil
    }
    
    parsed, err := uuid.ParseUUID(v.ValueString())
    if err != nil {
        return err
    }
    
    v.parsed = &parsed
    return nil
}

func (v *UUIDValue) Normalize() error {
    if err := v.ensureParsed(); err != nil {
        return err
    }
    // Store canonical string representation
    v.StringValue = basetypes.NewStringValue(uuid.FormatUUID(*v.parsed))
    return nil
}

func (v *UUIDValue) Equal(o attr.Value) bool {
    other, ok := o.(*UUIDValue)
    if !ok {
        return false
    }
    
    if err := v.ensureParsed(); err != nil {
        return false
    }
    if err := other.ensureParsed(); err != nil {
        return false
    }
    
    return bytes.Equal(v.parsed[:], other.parsed[:])
}

// Add convenience methods
func (v *UUIDValue) IsNil() bool {
    return v == nil || v.IsNull()
}

func (v *UUIDValue) Version() (int, error) {
    if err := v.ensureParsed(); err != nil {
        return 0, err
    }
    return int((v.parsed[6] & 0xF0) >> 4), nil
}

func NewUUIDFromBytes(b [16]byte) *UUIDValue {
    return &UUIDValue{
        StringValue: basetypes.NewStringValue(uuid.FormatUUID(b)),
        parsed:      &b,
    }
}
```

Changes needed:

1. Cache parsed UUID value to avoid repeated parsing
2. Enforce canonical string format
3. Add clear distinction between types and aliases
4. Add convenience methods for common operations
5. Use pointer receivers for better efficiency
6. Add more comprehensive validation
7. Add tests for new functionality
