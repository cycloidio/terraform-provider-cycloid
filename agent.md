# Agent Development Notes

This file contains important notes and context for the AI agent working on this terraform provider.

## API Naming Conventions

From the cycloid API or middleware, you can map these names:

- `service_catalog` == `stack`
- `service_catalog_source` == `catalog_repository`

The former are the legacy API names, the latter are the customer-facing naming conventions.

## Field Mapping Notes

When working with the cycloid-cli models, keep in mind:

- Component fields may use internal API names that differ from Terraform resource field names
- Service catalog references are accessed via `component.ServiceCatalog.Ref`
- Stack versions can be of type "tag", "branch", or commit hash
- Environment and Project names are nested objects that need safe dereferencing

## Implementation Patterns

- Follow the same patterns as `TeamToModel` when implementing `*ToModel` functions
- Use `types.StringPointerValue()` for pointer fields and `types.StringValue()` for direct string fields
- Always handle nil cases for nested objects
- Set default boolean values for fields that don't exist in the API model but are required in Terraform schema

## Error Handling Guidelines

- **Avoid Panics**: Never use `panic()` in production code. Always return proper error diagnostics.
- **Use Diagnostics**: Return `diag.Diagnostics` with descriptive error messages for Terraform framework operations.
- **Clear Messages**: Provide specific error messages that explain what went wrong and what was expected.
- **Graceful Failures**: Handle unexpected input types gracefully instead of crashing the provider.

**Example:**
```go
// Bad: panic(litter.Sdump("Incorrect type", valueType))

// Good: 
return nil, diag.Diagnostics{
    diag.NewErrorDiagnostic(
        "Failed to convert dynamic value to variables",
        fmt.Sprintf("Unsupported value type: %T. Expected map[string]interface{}", valueType),
    ),
}
```

## Code Style Guidelines

- **No Comments**: Avoid adding comments to code. The code should be self-explanatory.
- **Descriptive Names**: Use clear, descriptive variable and function names.
- **Minimal Documentation**: Keep code concise without explanatory comments.

## Terraform Naming Conventions

By convention, when referring to the name of an entity in a terraform attribute, we speak of the canonical.

The only exception is when the resource is about the said entity, we refer then to it canonical attribute by the name `canonical`.

**Example:**
- For the `team_resource` we refer to its organization canonical as `organization` 
- But the canonical of the team itself is named `canonical`
