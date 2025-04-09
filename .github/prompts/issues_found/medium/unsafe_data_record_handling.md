# Title

Unsafe Data Record Type Handling and Conversion

## Problem

The data record resource implementation has several reliability and security concerns:

1. Unsafe type conversions without proper validation
2. String manipulation of JSON data instead of proper parsing
3. Missing input sanitization
4. Inefficient type handling with multiple conversions
5. Potential null reference issues

## Impact

**Severity: medium**

This issue impacts reliability and security:

- Type conversion errors could corrupt data
- JSON injection possible via malformed input
- Performance impact from inefficient conversions
- Potential panics from null pointer dereferences
- Memory inefficiency from redundant conversions

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/data_record/resource_data_record.go

## Code Issue

```go
func convertResourceModelToMap(columnsAsString *string) (mapColumns map[string]any, err error) {
    // Unsafe string manipulation of JSON
    replacedColumns := strings.ReplaceAll(*columnsAsString, `<null>`, `""`)
    columnsAsString = &replacedColumns

    // Multiple unnecessary conversions
    jsonColumns, err := json.Marshal(columnsAsString)
    if err != nil {
        return nil, err
    }
    unquotedJsonColumns, err := strconv.Unquote(string(jsonColumns))
    if err != nil {
        return nil, err
    }
    err = json.Unmarshal([]byte(unquotedJsonColumns), &mapColumns)
    // ...
}

func caseArrayOfAny(ctx context.Context, attrValue map[string]attr.Value, attrType map[string]attr.Type,
    apiClient *client, objectType map[string]attr.Type, key, environmentId, tableLogicalName, recordid string) error {
    // Unsafe type assertions without validation
    item, ok := rawItem.(map[string]any)
    if !ok {
        return errors.New("error asserting rawItem to map[string]any")
    }
    // ...
}

// Multiple small type conversion functions with duplicated logic
func caseBool(columnValue any, attrValue map[string]attr.Value, attrType map[string]attr.Type, key string) {
    value, ok := columnValue.(bool)
    if ok {
        attrValue[key] = types.BoolValue(value)
        attrType[key] = types.BoolType
    }
}
```

## Fix

Implement safe type handling and efficient conversions:

```go
// Define strong types for validation
type DataRecordValue struct {
    Type  string      `json:"type"`
    Value interface{} `json:"value"`
}

type DataRecordColumn struct {
    Name     string
    Type     string
    Required bool
    MaxLen   int
}

func validateDataRecordSchema(columns map[string]DataRecordColumn, data map[string]interface{}) error {
    for name, col := range columns {
        value, exists := data[name]
        if !exists {
            if col.Required {
                return fmt.Errorf("required column %s is missing", name)
            }
            continue
        }
        
        if err := validateDataType(value, col.Type); err != nil {
            return fmt.Errorf("invalid value for column %s: %w", name, err)
        }
        
        if col.MaxLen > 0 {
            if err := validateLength(value, col.MaxLen); err != nil {
                return fmt.Errorf("value too long for column %s: %w", name, err)
            }
        }
    }
    return nil
}

func convertResourceModelToMap(data []byte) (map[string]interface{}, error) {
    // Use a decoder for streaming large JSON
    decoder := json.NewDecoder(bytes.NewReader(data))
    decoder.UseNumber() // Preserve number precision
    
    var result map[string]interface{}
    if err := decoder.Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid JSON data: %w", err)
    }
    
    // Sanitize and validate values
    sanitized := make(map[string]interface{})
    for key, value := range result {
        clean, err := sanitizeValue(value)
        if err != nil {
            return nil, fmt.Errorf("invalid value for key %s: %w", key, err)
        }
        sanitized[key] = clean
    }
    
    return sanitized, nil
}

// Unified type conversion with validation
type TypeConverter struct {
    typeRegistry map[string]func(interface{}) (attr.Value, error)
}

func NewTypeConverter() *TypeConverter {
    return &TypeConverter{
        typeRegistry: map[string]func(interface{}) (attr.Value, error){
            "bool": func(v interface{}) (attr.Value, error) {
                switch val := v.(type) {
                case bool:
                    return types.BoolValue(val), nil
                case string:
                    b, err := strconv.ParseBool(val)
                    if err != nil {
                        return nil, err
                    }
                    return types.BoolValue(b), nil
                default:
                    return nil, fmt.Errorf("cannot convert %T to bool", v)
                }
            },
            "int": func(v interface{}) (attr.Value, error) {
                switch val := v.(type) {
                case json.Number:
                    i, err := val.Int64()
                    if err != nil {
                        return nil, err
                    }
                    return types.Int64Value(i), nil
                case float64:
                    return types.Int64Value(int64(val)), nil
                default:
                    return nil, fmt.Errorf("cannot convert %T to int", v)
                }
            },
            // Add other type conversions...
        },
    }
}

func (tc *TypeConverter) Convert(value interface{}, targetType string) (attr.Value, error) {
    converter, ok := tc.typeRegistry[targetType]
    if !ok {
        return nil, fmt.Errorf("unsupported type: %s", targetType)
    }
    return converter(value)
}

func (r *DataRecordResource) convertColumnsToState(ctx context.Context, data map[string]interface{}, schema map[string]DataRecordColumn) (*types.Object, error) {
    converter := NewTypeConverter()
    
    attrs := make(map[string]attr.Value)
    attrTypes := make(map[string]attr.Type)
    
    for key, value := range data {
        colSchema, ok := schema[key]
        if !ok {
            return nil, fmt.Errorf("unknown column: %s", key)
        }
        
        attrValue, err := converter.Convert(value, colSchema.Type)
        if err != nil {
            return nil, fmt.Errorf("conversion error for column %s: %w", key, err)
        }
        
        attrs[key] = attrValue
        attrTypes[key] = attrValue.Type(ctx)
    }
    
    return types.ObjectValue(attrTypes, attrs)
}
```

Changes needed:

1. Implement proper JSON parsing without string manipulation
2. Add strong typing and validation for data values
3. Create unified type conversion system
4. Add input sanitization
5. Implement value validation
6. Add schema validation
7. Add memory efficient processing
8. Add comprehensive error handling
9. Add performance optimizations
10. Add test coverage for edge cases
