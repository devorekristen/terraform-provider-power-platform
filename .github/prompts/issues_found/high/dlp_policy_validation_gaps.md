# Title

DLP Policy Validation and Security Gaps

## Problem

The Data Loss Prevention policy implementation has several security concerns:

1. Insufficient validation of custom connector patterns
2. No validation of endpoint rule ordering conflicts
3. No validation of policy overlaps/conflicts
4. Incomplete action rule validation
5. Missing environment existence validation

## Impact

**Severity: high**

This issue impacts data security and policy enforcement:

- Malformed custom connector patterns could bypass security
- Conflicting rules could lead to unintended data exposure
- Invalid environments could cause policy enforcement gaps
- Action rule conflicts could create security holes
- Missing validations could allow policy bypass

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/dlp_policy/resource_dlp_policy.go

## Code Issue

```go
func (r DataLossPreventionPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    // ...
    // Only basic action rule validation, missing many security checks
    for _, c := range connectors {
        if (c.DefaultActionRuleBehavior != "" && len(c.ActionRules) == 0) || (c.DefaultActionRuleBehavior == "" && len(c.ActionRules) > 0) {
            resp.Diagnostics.AddAttributeError(
                path.Empty(),
                "Incorrect attribute Configuration",
                "Expected 'default_action_rule_behavior' to be empty if 'action_rules' are empty.",
            )
        }
    }
}

func convertToDlpCustomConnectorUrlPatternsDefinition(ctx context.Context, diags diag.Diagnostics, patterns types.Set) []dlpConnectorUrlPatternsDefinitionDto {
    // No pattern validation or security checks
    // ...
}
```

## Fix

Implement comprehensive policy validation:

```go
// Add new types for validation
type PolicyValidationContext struct {
    ExistingPolicies []dlpPolicyModelDto
    Environments     map[string]bool
    ConnectorTypes   map[string]ConnectorMetadata
}

type ConnectorMetadata struct {
    Type          string
    Capabilities  []string
    RequiredRules []string
}

func (r *DataLossPreventionPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config *dataLossPreventionPolicyResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    
    // Create validation context
    validationCtx, err := r.buildValidationContext(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Failed to build validation context", err.Error())
        return
    }
    
    // Validate environments exist
    if err := r.validateEnvironments(ctx, config.Environments, validationCtx); err != nil {
        resp.Diagnostics.AddError("Environment validation failed", err.Error())
    }
    
    // Validate custom connector patterns
    if err := r.validateCustomConnectorPatterns(ctx, config.CustomConnectorsPatterns); err != nil {
        resp.Diagnostics.AddError("Custom connector pattern validation failed", err.Error())
    }
    
    // Validate connector groups
    if err := r.validateConnectorGroups(ctx, config, validationCtx); err != nil {
        resp.Diagnostics.AddError("Connector group validation failed", err.Error())
    }
    
    // Check for policy conflicts
    if err := r.validatePolicyConflicts(ctx, config, validationCtx); err != nil {
        resp.Diagnostics.AddError("Policy conflict detected", err.Error())
    }
}

func (r *DataLossPreventionPolicyResource) validateCustomConnectorPatterns(ctx context.Context, patterns types.Set) error {
    seen := make(map[string]int)
    for _, pattern := range patterns.Elements() {
        // Validate pattern syntax
        if err := validateUrlPattern(pattern.HostUrlPattern); err != nil {
            return fmt.Errorf("invalid URL pattern: %w", err)
        }
        
        // Check for duplicates/overlaps
        if existing, exists := seen[pattern.HostUrlPattern]; exists {
            return fmt.Errorf("overlapping URL patterns at positions %d and %d", existing, pattern.Order)
        }
        
        // Validate order
        if pattern.Order < 0 {
            return fmt.Errorf("pattern order must be positive")
        }
        seen[pattern.HostUrlPattern] = pattern.Order
    }
    return nil
}

func (r *DataLossPreventionPolicyResource) validateConnectorGroups(ctx context.Context, config *dataLossPreventionPolicyResourceModel, validationCtx *PolicyValidationContext) error {
    // Validate each connector group
    groups := []struct {
        name      string
        connectors types.Set
    }{
        {"business", config.BusinessGeneralConnectors},
        {"non-business", config.NonBusinessConfidentialConnectors},
        {"blocked", config.BlockedConnectors},
    }
    
    for _, group := range groups {
        if err := r.validateConnectorGroup(ctx, group.name, group.connectors, validationCtx); err != nil {
            return fmt.Errorf("invalid %s connector group: %w", group.name, err)
        }
    }
    
    // Check for connector conflicts between groups
    if err := r.validateConnectorGroupConflicts(groups); err != nil {
        return err
    }
    
    return nil
}

func (r *DataLossPreventionPolicyResource) validateActionRules(ctx context.Context, connector dlpConnectorModelDto, metadata ConnectorMetadata) error {
    // Validate rule ordering
    seen := make(map[string]bool)
    for _, rule := range connector.ActionRules {
        // Check for duplicate actions
        if seen[rule.ActionId] {
            return fmt.Errorf("duplicate action rule: %s", rule.ActionId)
        }
        seen[rule.ActionId] = true
        
        // Validate action exists for connector
        if !containsAction(metadata.Capabilities, rule.ActionId) {
            return fmt.Errorf("action %s not supported by connector", rule.ActionId)
        }
        
        // Check required rules are present
        for _, required := range metadata.RequiredRules {
            if !seen[required] {
                return fmt.Errorf("missing required rule: %s", required)
            }
        }
    }
    return nil
}

func (r *DataLossPreventionPolicyResource) validatePolicyConflicts(ctx context.Context, config *dataLossPreventionPolicyResourceModel, validationCtx *PolicyValidationContext) error {
    // Check for overlapping environment assignments
    for _, existing := range validationCtx.ExistingPolicies {
        if existing.Name == config.Id.ValueString() {
            continue // Skip self
        }
        
        if hasEnvironmentOverlap(existing.Environments, config.Environments) {
            return fmt.Errorf("policy %s has overlapping environments with existing policy %s", 
                config.DisplayName.ValueString(), existing.DisplayName)
        }
    }
    return nil
}
```

Changes needed:

1. Add comprehensive URL pattern validation
2. Implement endpoint rule conflict detection
3. Add environment existence validation
4. Add connector capability validation
5. Implement policy overlap detection
6. Add action rule dependency checking
7. Add test cases for validation logic
8. Improve error messages with clear remediation steps
9. Add policy impact analysis
