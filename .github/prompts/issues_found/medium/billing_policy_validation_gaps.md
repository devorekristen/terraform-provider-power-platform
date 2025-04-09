# Title

Insufficient Billing Policy Validation and State Management

## Problem

The billing policy implementation has several security and reliability concerns:

1. No validation of subscription ID format
2. Missing billing instrument state validation
3. Incomplete status transition validation
4. No validation of location against allowed regions
5. Possible race conditions in state updates

## Impact

**Severity: medium**

This issue impacts reliability and security:

- Invalid subscription IDs could cause runtime errors
- Incorrect state transitions could leave policies in invalid states
- Location validation gaps could allow deployment to unsupported regions
- Race conditions could lead to inconsistent state
- Missing validations could allow invalid configurations

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/licensing/resource_billing_policy.go

## Code Issue

```go
func (r *BillingPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // No validation of subscription ID format
    // No validation of resource group existence
    // No validation of location
    billingPolicyToCreate := billingPolicyCreateDto{
        BillingInstrument: BillingInstrumentDto{
            ResourceGroup:  plan.BillingInstrument.ResourceGroup.ValueString(),
            SubscriptionId: plan.BillingInstrument.SubscriptionId.ValueString(),
        },
        Location: plan.Location.ValueString(),
        Name:     plan.Name.ValueString(),
    }
}

func (r *BillingPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // No validation of status transitions
    // No state locking for updates
    if plan.Name.ValueString() != state.Name.ValueString() ||
        plan.Status.ValueString() != state.Status.ValueString() {
        policyToUpdate := BillingPolicyUpdateDto{
            Name:   plan.Name.ValueString(),
            Status: plan.Status.ValueString(),
        }
        // ...
    }
}
```

## Fix

Implement comprehensive validation and state management:

```go
// Define allowed transitions and validations
type BillingPolicyValidator struct {
    allowedRegions map[string]bool
    statusTransitions map[string][]string
}

func NewBillingPolicyValidator() *BillingPolicyValidator {
    return &BillingPolicyValidator{
        allowedRegions: map[string]bool{
            "westus": true,
            "eastus": true,
            "westeurope": true,
            // Add other supported regions
        },
        statusTransitions: map[string][]string{
            "": {"Enabled"},
            "Enabled": {"Disabled"},
            "Disabled": {"Enabled"},
        },
    }
}

func (v *BillingPolicyValidator) ValidateCreate(ctx context.Context, policy *billingPolicyCreateDto) error {
    if err := v.validateLocation(policy.Location); err != nil {
        return fmt.Errorf("invalid location: %w", err)
    }
    
    if err := v.validateSubscriptionId(policy.BillingInstrument.SubscriptionId); err != nil {
        return fmt.Errorf("invalid subscription ID: %w", err)
    }
    
    if err := v.validateResourceGroup(ctx, policy.BillingInstrument.SubscriptionId, 
        policy.BillingInstrument.ResourceGroup); err != nil {
        return fmt.Errorf("invalid resource group: %w", err)
    }
    
    if err := v.validateStatus("", policy.Status); err != nil {
        return fmt.Errorf("invalid initial status: %w", err)
    }
    
    return nil
}

func (v *BillingPolicyValidator) validateLocation(location string) error {
    if location == "" {
        return errors.New("location cannot be empty")
    }
    
    if !v.allowedRegions[strings.ToLower(location)] {
        return fmt.Errorf("location %s is not supported", location)
    }
    
    return nil
}

func (v *BillingPolicyValidator) validateSubscriptionId(subId string) error {
    if subId == "" {
        return errors.New("subscription ID cannot be empty")
    }
    
    if !regexp.MustCompile(`^[0-9a-fA-F]{8}-([0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}$`).MatchString(subId) {
        return fmt.Errorf("invalid subscription ID format: %s", subId)
    }
    
    return nil
}

func (v *BillingPolicyValidator) validateResourceGroup(ctx context.Context, subId, resourceGroup string) error {
    // Call Azure API to verify resource group exists
    exists, err := checkResourceGroupExists(ctx, subId, resourceGroup)
    if err != nil {
        return fmt.Errorf("failed to verify resource group: %w", err)
    }
    
    if !exists {
        return fmt.Errorf("resource group %s does not exist in subscription %s", resourceGroup, subId)
    }
    
    return nil
}

func (v *BillingPolicyValidator) validateStatus(currentStatus, newStatus string) error {
    if newStatus == "" {
        return errors.New("status cannot be empty")
    }
    
    allowedTransitions, ok := v.statusTransitions[currentStatus]
    if !ok {
        return fmt.Errorf("unknown current status: %s", currentStatus)
    }
    
    for _, allowed := range allowedTransitions {
        if allowed == newStatus {
            return nil
        }
    }
    
    return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
}

// State management with locking
type BillingPolicyState struct {
    sync.RWMutex
    policy *BillingPolicyResourceModel
}

func (s *BillingPolicyState) Update(ctx context.Context, updateFn func(*BillingPolicyResourceModel) error) error {
    s.Lock()
    defer s.Unlock()
    
    if err := updateFn(s.policy); err != nil {
        return err
    }
    
    return nil
}

// Resource implementation
func (r *BillingPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
    defer exitContext()

    var plan *BillingPolicyResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Validate before creation
    validator := NewBillingPolicyValidator()
    policy := &billingPolicyCreateDto{
        BillingInstrument: BillingInstrumentDto{
            ResourceGroup:  plan.BillingInstrument.ResourceGroup.ValueString(),
            SubscriptionId: plan.BillingInstrument.SubscriptionId.ValueString(),
        },
        Location: plan.Location.ValueString(),
        Name:     plan.Name.ValueString(),
        Status:   plan.Status.ValueString(),
    }
    
    if err := validator.ValidateCreate(ctx, policy); err != nil {
        resp.Diagnostics.AddError("Invalid billing policy configuration", err.Error())
        return
    }

    // Create policy with retry
    var createdPolicy *BillingPolicyDto
    err := retry.Do(func() error {
        var err error
        createdPolicy, err = r.LicensingClient.CreateBillingPolicy(ctx, *policy)
        return err
    }, retry.OnRetry(func(n uint, err error) {
        tflog.Info(ctx, fmt.Sprintf("Retrying creation after error: %v", err))
    }))
    
    if err != nil {
        resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %s", r.FullTypeName()), err.Error())
        return
    }

    // Update state atomically
    state := &BillingPolicyState{policy: plan}
    if err := state.Update(ctx, func(p *BillingPolicyResourceModel) error {
        p.Id = types.StringValue(createdPolicy.Id)
        p.Name = types.StringValue(createdPolicy.Name)
        p.Location = types.StringValue(createdPolicy.Location)
        p.Status = types.StringValue(createdPolicy.Status)
        p.BillingInstrument.Id = types.StringValue(createdPolicy.BillingInstrument.Id)
        p.BillingInstrument.ResourceGroup = types.StringValue(createdPolicy.BillingInstrument.ResourceGroup)
        p.BillingInstrument.SubscriptionId = types.StringValue(createdPolicy.BillingInstrument.SubscriptionId)
        return nil
    }); err != nil {
        resp.Diagnostics.AddError("Failed to update state", err.Error())
        return
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, state.policy)...)
}
```

Changes needed:

1. Add subscription ID validation
2. Add resource group validation
3. Add location validation
4. Add status transition validation
5. Add state locking
6. Add retry handling
7. Add validation interfaces
8. Add comprehensive tests
9. Add proper error messages
10. Document validation rules
