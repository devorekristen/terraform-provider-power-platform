# Title

Insufficient Application Package Installation Validation

## Problem

The application package installation resource lacks critical security validations:

1. No validation of application package source/authenticity
2. No version checking or compatibility verification
3. Missing rollback mechanism for failed installations
4. No validation of application dependencies
5. Update and Delete operations are no-ops without validation

## Impact

**Severity: critical**

This issue has severe security and reliability impacts:

- Potential installation of malicious or incompatible applications
- No way to verify package integrity before installation
- Silent failures in update/delete operations
- Possible system instability from unverified dependencies
- Lack of rollback could leave system in inconsistent state

## Location

File: /workspaces/terraform-provider-power-platform/internal/services/application/resource_environment_application_package_install.go

## Code Issue

```go
func (r *EnvironmentApplicationPackageInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // ...
    dvExits, err := r.ApplicationClient.DataverseExists(ctx, state.EnvironmentId.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(fmt.Sprintf("Client error when checking if Dataverse exists in environment '%s'", state.EnvironmentId.ValueString()), err.Error())
    }

    if !dvExits {
        resp.Diagnostics.AddError(fmt.Sprintf("No Dataverse exists in environment '%s'", state.EnvironmentId.ValueString()), "")
        return
    }

    // No package validation before installation
    applicationId, err := r.ApplicationClient.InstallApplicationInEnvironment(ctx, state.EnvironmentId.ValueString(), state.UniqueName.ValueString())
    // ...
}

func (r *EnvironmentApplicationPackageInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // ...
    tflog.Debug(ctx, "No application have been updated, as this is the expected behavior")
}

func (r *EnvironmentApplicationPackageInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // ...
    tflog.Debug(ctx, "No application have been uninstalled, as this is the expected behavior")
}
```

## Fix

Implement comprehensive package validation and lifecycle management:

```go
type ApplicationPackageMetadata struct {
    Version            string
    RequiredVersion    string
    Dependencies       []string
    ChecksumSHA256    string
    MinDataverseVersion string
}

func (r *EnvironmentApplicationPackageInstallResource) validatePackage(ctx context.Context, envId string, uniqueName string) (*ApplicationPackageMetadata, error) {
    // Get package metadata
    metadata, err := r.ApplicationClient.GetApplicationMetadata(ctx, uniqueName)
    if err != nil {
        return nil, fmt.Errorf("failed to get application metadata: %w", err)
    }

    // Verify package signature/checksum
    if err := r.verifyPackageIntegrity(ctx, uniqueName, metadata.ChecksumSHA256); err != nil {
        return nil, fmt.Errorf("package integrity check failed: %w", err)
    }

    // Check Dataverse compatibility
    dvVersion, err := r.ApplicationClient.GetDataverseVersion(ctx, envId)
    if err != nil {
        return nil, err
    }
    if !isCompatibleVersion(dvVersion, metadata.MinDataverseVersion) {
        return nil, fmt.Errorf("incompatible Dataverse version %s (required: %s)", dvVersion, metadata.MinDataverseVersion)
    }

    // Validate dependencies
    for _, dep := range metadata.Dependencies {
        if err := r.validateDependency(ctx, envId, dep); err != nil {
            return nil, fmt.Errorf("dependency validation failed: %w", err)
        }
    }

    return metadata, nil
}

func (r *EnvironmentApplicationPackageInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // ...existing environment validation...

    // Validate package before installation
    metadata, err := r.validatePackage(ctx, state.EnvironmentId.ValueString(), state.UniqueName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Package validation failed", err.Error())
        return
    }

    // Start installation with rollback support
    applicationId, err := r.ApplicationClient.InstallApplicationWithRollback(ctx, state.EnvironmentId.ValueString(), state.UniqueName.ValueString())
    if err != nil {
        if rbErr := r.rollbackInstallation(ctx, state.EnvironmentId.ValueString()); rbErr != nil {
            resp.Diagnostics.AddWarning("Rollback failed", rbErr.Error())
        }
        resp.Diagnostics.AddError("Installation failed", err.Error())
        return
    }
    // ...
}

func (r *EnvironmentApplicationPackageInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // ...
    // Validate the current installation
    currentState, err := r.ApplicationClient.GetApplicationState(ctx, plan.EnvironmentId.ValueString(), plan.UniqueName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to get application state", err.Error())
        return
    }

    if !currentState.IsHealthy {
        resp.Diagnostics.AddWarning("Application health check failed", 
            "The application is in an unhealthy state and may need reinstallation")
    }
    // ...
}

func (r *EnvironmentApplicationPackageInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // ... 
    // Verify it's safe to leave the application installed
    deps, err := r.ApplicationClient.GetDependentApplications(ctx, state.EnvironmentId.ValueString(), state.UniqueName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to check dependencies", err.Error())
        return
    }

    if len(deps) > 0 {
        resp.Diagnostics.AddWarning("Application has dependents",
            fmt.Sprintf("The following applications depend on this: %v", deps))
    }
    // ...
}
```

Changes needed:

1. Add package metadata validation
2. Implement package integrity verification
3. Add version compatibility checks
4. Add dependency validation
5. Implement proper update verification
6. Add installation rollback capability
7. Add health checks for installed applications
8. Improve error handling and diagnostics
9. Add comprehensive logging
