# Title

Inefficient and Potentially Unsafe Checksum Handling

## Problem

The sync attribute plan modifier has several minor issues in its checksum handling:

1. Using SHA256 but referring to MD5 in error message
2. No memory limit on checksum calculation
3. No caching of checksum values
4. Potential performance impact on large files
5. Inconsistent null/unknown value handling

## Impact

**Severity: low**

This issue impacts performance and maintainability:

- Incorrect error messages could confuse users
- Large files could cause memory pressure
- Redundant checksum calculations affect performance
- Inconsistent handling of null values
- Mixed hash algorithm references in code

## Location

File: /workspaces/terraform-provider-power-platform/internal/modifiers/sync_attribute_plan_modifier.go

## Code Issue

```go
func (d *syncAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
    // ...
    value, err := helpers.CalculateSHA256(settingsFile.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(fmt.Sprintf("Error calculating MD5 checksum for %s", d.syncAttribute), err.Error())
        return
    }

    if value == "" {
        resp.PlanValue = types.StringUnknown()
    } else {
        resp.PlanValue = types.StringValue(value)
    }
}
```

## Fix

Implement efficient and consistent checksum handling:

```go
const maxChecksumSize = 32 * 1024 * 1024 // 32MB limit for checksum calculation

type checksumCache struct {
    sync.RWMutex
    values map[string]string
}

var globalChecksumCache = &checksumCache{
    values: make(map[string]string),
}

func (d *syncAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
    var settingsFile types.String
    diags := req.Plan.GetAttribute(ctx, path.Root(d.syncAttribute), &settingsFile)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Consistent null/unknown handling
    if settingsFile.IsNull() || settingsFile.IsUnknown() {
        resp.PlanValue = types.StringNull()
        return
    }

    fileContent := settingsFile.ValueString()
    if len(fileContent) > maxChecksumSize {
        resp.Diagnostics.AddError(
            "File too large for checksum",
            fmt.Sprintf("File %s exceeds maximum size of %d bytes for checksum calculation", 
                d.syncAttribute, maxChecksumSize),
        )
        return
    }

    // Check cache first
    globalChecksumCache.RLock()
    if cachedValue, ok := globalChecksumCache.values[fileContent]; ok {
        globalChecksumCache.RUnlock()
        resp.PlanValue = types.StringValue(cachedValue)
        return
    }
    globalChecksumCache.RUnlock()

    // Calculate new checksum
    value, err := calculateFileChecksum(fileContent)
    if err != nil {
        resp.Diagnostics.AddError(
            fmt.Sprintf("Error calculating SHA256 checksum for %s", d.syncAttribute),
            err.Error(),
        )
        return
    }

    // Cache the result
    globalChecksumCache.Lock()
    globalChecksumCache.values[fileContent] = value
    globalChecksumCache.Unlock()

    resp.PlanValue = types.StringValue(value)
}

func calculateFileChecksum(content string) (string, error) {
    h := sha256.New()
    if _, err := io.WriteString(h, content); err != nil {
        return "", fmt.Errorf("failed to calculate checksum: %w", err)
    }
    return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Add cache cleanup on low memory
func init() {
    go func() {
        for {
            // Clear cache periodically or when memory pressure is high
            time.Sleep(5 * time.Minute)
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            
            if m.Alloc > (m.Sys * 80 / 100) { // Clear if using >80% of allocated memory
                globalChecksumCache.Lock()
                globalChecksumCache.values = make(map[string]string)
                globalChecksumCache.Unlock()
            }
        }
    }()
}
```

Changes needed:

1. Fix error message to reference correct hash algorithm
2. Add file size limit for checksum calculation
3. Implement checksum caching
4. Add consistent null/unknown handling
5. Add memory pressure handling
6. Add proper error messages
7. Add cache cleanup mechanism
8. Add performance monitoring
9. Add unit tests for edge cases
