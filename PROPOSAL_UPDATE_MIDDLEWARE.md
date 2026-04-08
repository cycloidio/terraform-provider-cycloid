# Proposal: Update cycloid-cli dependency to fix API error handling

## Problem

The provider depends on `cycloid-cli v1.0.98-0.20260302211105-2991ca6d8bb3` (2026-03-02),
which uses a swagger-generated HTTP client. When the Cycloid API returns an error response
where `ErrorDetailsItem.errors.details` is a string instead of `[]string`, the swagger client
returns a raw JSON unmarshal error instead of the actual API error message:

```
json: cannot unmarshal string into Go struct field ErrorDetailsItem.errors.details of type []string
```

This makes debugging test failures and resource errors confusing — you see the
parsing error, not the actual API error.

## Root cause

The swagger-generated client tries to unmarshal the HTTP error body into
`models.ErrorPayload`. When the JSON shape doesn't match the generated struct,
the unmarshal fails and the raw body is lost.

## Solution

The CLI migrated to a custom `GenericRequest` HTTP client on 2026-03-16 (commit `1f51ceb7`)
which handles this gracefully:

1. Reads the raw response body
2. Tries to unmarshal into `ErrorPayload`
3. If that fails, falls back to the raw body text
4. Returns an `APIResponseError` with a clean message: `API error 404: The Environment was not found`

Update the `cycloid-cli` dependency to a recent develop commit to get this fix.

## Scope of changes

The middleware interface changed uniformly: every method now returns an additional
`*http.Response` value. Most changes are mechanical (`result, err :=` →
`result, _, err :=`).

### Mechanical changes (~50 call sites, 10 files)

| File | Calls | Notes |
|------|-------|-------|
| `provider/component_resource.go` | 12 | |
| `provider/credential_resource.go` | 7 | |
| `provider/credential_datasource.go` | 1 | |
| `provider/environment_resource.go` | 5 | |
| `provider/organization_resource.go` | 12 | |
| `provider/project_resource.go` | 6 | |
| `provider/team_resource.go` | 8 | |
| `provider/team_member_resource.go` | 5 | |
| `provider/stack_resource.go` | 1 | |
| `provider/test_dependencies.go` | 6 | |

### Non-mechanical changes (3 spots)

#### `CreateAndConfigureComponent` → `CreateOrUpdateComponent`

Method renamed in the new CLI. Parameters are identical (same types, same order).
2 call sites in `component_resource.go` (lines 192, 299).

#### `GenericRequest` signature change

Old positional args collapsed into a `Request` struct.
4 call sites in `organization_resource.go` (lines 149, 270, 395, 426).

```go
// Old:
m.GenericRequest("GET", &canonical, nil, nil, nil, licence, "organizations", canonical, "licence")

// New:
m.GenericRequest(middleware.Request{
    Method:       "GET",
    Organization: &canonical,
    Route:        []string{"organizations", canonical, "licence"},
}, licence)
```

#### `ListStackVersions` return type change

Returns `[]*middleware.StackVersion` instead of `[]*models.ServiceCatalogSourceVersion`.
Update `matchStackVersion` function signature in `component_resource.go` (line 586).
Fields used (`Type`, `Name`, `CommitHash`) are identical between the two types.

## Verification

```bash
go build ./...
go test -short ./provider/...
TF_ACC=1 go test -v -run TestAccProjectResource ./provider/...
```

## Execution

1. Create branch from main (after acceptance tests PR is merged)
2. `go get github.com/cycloidio/cycloid-cli@develop && go mod tidy`
3. Apply mechanical changes across all files
4. Fix the 3 non-mechanical spots
5. Fix imports
6. Verify build + unit tests + one acceptance test
7. Open PR
