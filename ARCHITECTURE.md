# ARTESCA Terraform Provider - Technical Architecture Document

## Executive Summary

This document explains the architectural decisions, design patterns, and implementation strategies used in the ARTESCA Terraform Provider. The primary goals are:

1. **Maintainability** - Easy to understand and modify
2. **Extensibility** - Simple to add new API calls and resources
3. **Reliability** - Consistent error handling and state management
4. **Simplicity** - Reduce complexity while maintaining DRY principles

## Table of Contents

1. [Architecture Overview](#architecture-overview) — high-level structure, three API surfaces, directory layout
2. [Design Principles](#design-principles) — separation of concerns, DRY, context-first, overlay reads
3. [Client Architecture](#client-architecture) — Management, OIDC, IAM, and S3 clients
4. [Resource Pattern](#resource-pattern) — standard structure, CRUD pattern, resource summary
5. [Adding New API Calls](#adding-new-api-calls) — step-by-step guide, decision tree
6. [Error Handling Strategy](#error-handling-strategy) — wrapping, user-facing messages, not-found
7. [State Management](#state-management) — drift detection, sensitive fields, state-only reads
8. [Security Patterns](#security-patterns) — credentials, sensitivity, env vars, TLS
9. [Testing Strategy](#testing-strategy) — acceptance tests, manual verification
10. [Future Extensibility](#future-extensibility) — adding resources, data sources, client enhancements

---

## Architecture Overview

### High-Level Structure

```
┌───────────────────────────────────────────────────────────────┐
│                    Terraform / OpenTofu Core                  │
│            (Handles plan, apply, destroy lifecycle)           │
└─────────────────────────────┬─────────────────────────────────┘
                              │
                              │ Plugin Protocol (gRPC)
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                  ARTESCA Terraform Provider                  │
│                                                              │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐     │
│   │   Provider   │   │  Resources   │   │ Data Sources │     │
│   │ (provider.go)│   │  (11 total)  │   │   (future)   │     │
│   └──────┬───────┘   └──────┬───────┘   └──────────────┘     │
│          │                  │                                │
│          ▼                  ▼                                │
│   ┌───────────────────────────────────────────────────────┐  │
│   │            Client Layer (ProviderClients)             │  │
│   │                                                       │  │
│   │   ┌────────────────┐  ┌───────────┐  ┌───────────┐    │  │
│   │   │ManagementClient│  │ IAMClient │  │ S3Client  │    │  │
│   │   │  (OIDC Token)  │  │ (AWS SigV4│  │ (AWS SigV4│    │  │
│   │   └───────┬────────┘  └─────┬─────┘  └─────┬─────┘    │  │
│   │           │                 │              │          │  │
│   │   ┌───────┴────────┐        │              │          │  │
│   │   │ OIDCTokenSource│  ┌─────┴──────────────┘─┐        │  │
│   │   │ (Token Cache)  │  │  sigv4.go (shared)   │        │  │
│   │   └────────────────┘  └──────────────────────┘        │  │
│   └───────────┼───────────────────┼──────────┼────────────┘  │
│               │                   │          │               │
└───────────────┼───────────────────┼──────────┼───────────────┘
                │                   │          │
                ▼                   ▼          ▼
     ┌──────────────────┐  ┌────────────┐  ┌──────────┐
     │  Management API  │  │IAM (Vault) │  │  S3 API  │
     │  X-Auth-Token    │  │ AWS SigV4  │  │ AWS SigV4│
     │  JSON / REST     │  │ XML / Form │  │ XML/REST │
     └──────────────────┘  └────────────┘  └──────────┘
```

### Key Components

| Component | Purpose | Files |
|-----------|---------|-------|
| **Provider** | Configuration, auth, client setup | `internal/provider/provider.go` |
| **Management Client** | OIDC-authenticated REST API | `internal/client/management*.go` |
| **OIDC Token Source** | Token fetch, cache, refresh | `internal/client/oidc.go` |
| **IAM Client** | AWS SigV4 signed API calls | `internal/client/iam.go` |
| **S3 Client** | AWS SigV4 signed S3 operations | `internal/client/s3.go`, `s3_lifecycle.go` |
| **SigV4 Helpers** | Shared signing primitives | `internal/client/sigv4.go` |
| **ProviderClients** | Bundles all three clients | `internal/client/provider_clients.go` |
| **Resources** | Terraform resource implementations | `internal/resources/*/` |

### Three API Surfaces

ARTESCA exposes three distinct APIs that the provider interacts with:

| API | Auth | Format | Purpose |
|-----|------|--------|---------|
| **Management API** | OIDC Bearer (custom `X-Authentication-Token` header) | JSON/REST | Accounts, locations, endpoints, replication, workflows |
| **IAM (Vault) API** | AWS SigV4 per-account credentials | XML/Form-encoded | Users, policies, access keys within accounts |
| **S3 API** | AWS SigV4 per-account credentials | XML/REST | Bucket creation, deletion, lifecycle configuration |

The IAM and S3 clients share SigV4 signing primitives extracted into `sigv4.go`.

### Directory Structure

```
terraform-provider-scality-artesca/
├── main.go                              # Provider server entry point
├── go.mod                               # Module: github.com/scality/terraform-provider-scality-artesca
├── Makefile                             # build, install, test, testacc, lint, fmt
├── .github/workflows/
│   ├── test.yml                         # Unit tests + build matrix (self-hosted)
│   ├── integration.yml                  # Integration tests against real ARTESCA (self-hosted)
│   └── release.yml                      # Tagged release builds
├── tests/integration/
│   ├── main.tf                          # 12-resource lifecycle config
│   ├── run.sh                           # Local test runner (build + apply + plan + destroy)
│   ├── integration.tfvars.example       # RING S3 variable template
│   └── README.md                        # Maintainer guide
├── examples/
│   ├── provider/main.tf                 # Provider config with OIDC
│   └── resources/
│       ├── artesca_account/main.tf      # Account + AWS provider integration
│       └── artesca_location/main.tf     # AWS S3 and RING location examples
├── internal/
│   ├── client/
│   │   ├── provider_clients.go          # ProviderClients bundle (Management + IAM + S3)
│   │   ├── management.go               # Management API base client + overlay
│   │   ├── management_account.go        # Account CRUD operations
│   │   ├── management_endpoint.go       # Endpoint CRUD operations
│   │   ├── management_location.go       # Location CRUD operations
│   │   ├── management_replication.go    # Replication stream operations
│   │   ├── management_workflow.go       # Workflow CRUD operations
│   │   ├── management_types.go          # Shared Management API data types
│   │   ├── oidc.go                      # OIDC token lifecycle
│   │   ├── iam.go                       # IAM SigV4 client
│   │   ├── s3.go                        # S3 SigV4 client (buckets)
│   │   ├── s3_lifecycle.go              # S3 lifecycle configuration
│   │   └── sigv4.go                     # Shared AWS SigV4 signing primitives
│   ├── provider/
│   │   └── provider.go                  # Schema, Configure, resource registration
│   ├── validators/                      # Custom schema validators (BucketName, etc.)
│   └── resources/
│       ├── account/                     # artesca_account
│       ├── bucket/                      # artesca_bucket
│       ├── endpoint/                    # artesca_endpoint
│       ├── location/                    # artesca_location
│       ├── replication/                 # artesca_replication
│       ├── user/                        # artesca_user
│       ├── user_access_key/             # artesca_user_access_key
│       ├── user_policy/                 # artesca_user_policy
│       ├── workflow_expiration/         # artesca_bucket_workflow_expiration
│       ├── workflow_replication/        # artesca_bucket_workflow_replication
│       └── workflow_transition/         # artesca_bucket_workflow_transition
```

Each resource directory typically contains:
- `model.go` — Terraform schema model structs with `tfsdk` tags
- `resource.go` — CRUD implementation and conversion helpers
- `schema_test.go` — Schema validation tests (optional)

The `bucket` resource is an exception: the model is defined inline in `resource.go` since the schema is small enough not to warrant a separate file.

---

## Design Principles

### 1. Separation of Concerns

Decision: Split Management, IAM, and S3 clients into separate files with a shared bundle.

Reasoning:
- Different authentication mechanisms (OIDC token vs AWS SigV4)
- Different wire formats (JSON vs XML)
- Different API patterns (REST vs form-encoded actions vs path-style S3)
- Independent evolution of each API surface

```
management*.go  → Management API (accounts, locations, endpoints, replication, workflows)
iam.go          → IAM API (users, policies, access keys)
s3.go           → S3 API (buckets, lifecycle)
sigv4.go        → AWS SigV4 signing (shared by IAM and S3 clients)
oidc.go         → OIDC token lifecycle (shared by Management client)
```

The `ProviderClients` struct bundles all three clients into a single value passed through `resp.ResourceData`, so each resource extracts only the client it needs:

```go
// Management API resources:
r.client = providerData.Management

// IAM resources:
r.iamClient = providerData.IAM

// S3 resources:
r.s3Client = providerData.S3
```

Benefits: Clear responsibility boundaries, no coupling between API types, SigV4 helpers shared without duplication.

---

### 2. DRY Principle with Pragmatism

Decision: Apply DRY where it reduces complexity, not religiously.

#### Where We Applied DRY

**1. Shared Constants**

```go
// iam.go — IAM / AWS SigV4 constants
const (
    iamAPIVersion   = "2010-05-08"
    iamAWSService   = "iam"
    iamHTTPTimeout  = 30 * time.Second
    contentTypeForm = "application/x-www-form-urlencoded"
)

// management.go — Management API constants
const (
    managementAPIPath     = "/api/v1"
    managementHTTPTimeout = 60 * time.Second
    contentTypeJSON       = "application/json"
)

// oidc.go — OIDC constants
const (
    oidcHTTPTimeout    = 30 * time.Second
    oidcTokenPreExpiry = 30 * time.Second
)
```

Reasoning: Changes once, benefits everywhere. Self-documenting names replace magic values.

**2. IAM Helper Method**

```go
func (c *IAMClient) doSignedRequest(ctx context.Context, accessKey, secretKey string, params url.Values) ([]byte, error)
```

The helper sets `Version` automatically and handles SigV4 signing, HTTP execution, and XML error parsing. All 9 IAM operations use this single method.

**3. Management Helper Method**

```go
func (c *ManagementClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error)
```

Handles JSON marshaling, token injection via `X-Authentication-Token`, and HTTP execution. All management operations use this.

#### Where We Didn't Apply DRY

**Management and IAM HTTP helpers are kept separate.**

```go
// Management: OIDC token header, JSON body
req.Header.Set("X-Authentication-Token", token)
req.Header.Set("Content-Type", contentTypeJSON)

// IAM: AWS SigV4 authorization, form-encoded body
req.Header.Set("Authorization", authHeader)
req.Header.Set("Content-Type", contentTypeForm)
```

These are different enough that a shared abstraction would add complexity without reducing it.

Rule of Thumb:
- Duplicate when abstractions would be more complex than the duplication
- Abstract when the pattern is identical and changes together

---

### 3. Context-First Design

Decision: Every public API method accepts `context.Context` as the first parameter.

```go
func (c *Client) MethodName(ctx context.Context, params...) (result, error)
```

Reasoning:
1. **Cancellation** — Terraform provides context to resources; propagating it through enables proper lifecycle management
2. **Timeouts** — HTTP clients respect context deadlines
3. **Logging** — `tflog` uses context for structured logging
4. **Future-proofing** — Distributed tracing, OpenTelemetry integration

Flow:
```
Terraform → Resource.Create(ctx) → Client.CreateAccount(ctx) → HTTP Request (with context)
                                                                         ↓
                                                        Respects timeout/cancellation
```

---

### 4. Constants Over Magic Values

Decision: All configuration values are package-level constants.

Before (magic values):
```go
params.Set("Version", "2010-05-08")
region := "us-east-1"
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
fullURL := fmt.Sprintf("%s/api/v1%s", c.BaseURL, path)
```

After (constants):
```go
params.Set("Version", iamAPIVersion)
region := c.region  // configurable via provider attribute
req.Header.Set("Content-Type", contentTypeForm)
fullURL := fmt.Sprintf("%s%s%s", c.BaseURL, managementAPIPath, path)
```

Benefits:

| Benefit | Example |
|---------|---------|
| Single source of truth | Update API version in one place |
| Self-documenting | `oidcTokenPreExpiry` explains purpose |
| Type safety | Compiler catches typos |
| Configurability | `iam_region` moved from constant to provider attribute |

The IAM region (`us-east-1`) was deliberately made configurable rather than a constant, since deployments may use different regions for SigV4 signing.

---

### 5. Overlay Read Pattern

Decision: No per-resource GET endpoints — reads go through a single overlay endpoint.

ARTESCA's management API does not provide individual GET endpoints for resources. Instead, `GET /config/overlay/view/{instanceId}` returns the entire configuration. Individual resource reads extract their data from the overlay:

```go
func (c *ManagementClient) GetLocation(ctx context.Context, name string) (*Location, error) {
    overlay, err := c.GetOverlay(ctx)
    if err != nil {
        return nil, err
    }
    loc, ok := overlay.Locations[name]
    if !ok {
        return nil, nil  // not found
    }
    return &loc, nil
}
```

This pattern is used for accounts, locations, endpoints, and replication streams. Workflow resources have no read endpoint at all — they preserve state from the last known write.

---

## Client Architecture

### Management Client (`management.go`)

#### Design

```go
type ManagementClient struct {
    BaseURL     string
    InstanceID  string
    TokenSource *OIDCTokenSource
    HTTPClient  *http.Client
}
```

Simple struct with clear dependencies. The `TokenSource` handles token lifecycle automatically — the management client just calls `TokenSource.Token(ctx)` on every request.

#### Helper Method Pattern

```go
// Private helper — handles HTTP mechanics
func (c *ManagementClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
    // 1. Marshal body to JSON
    // 2. Build full URL with managementAPIPath prefix
    // 3. Get token from OIDCTokenSource
    // 4. Set headers (X-Authentication-Token, Content-Type, Accept)
    // 5. Execute request
    // 6. Return body, status code, error
}

// Public methods — handle business logic and status code interpretation
func (c *ManagementClient) CreateLocation(ctx context.Context, loc *Location) (*Location, error) {
    body, status, err := c.doRequest(ctx, http.MethodPost, path, loc)
    if status != http.StatusCreated && status != http.StatusOK {
        return nil, fmt.Errorf("create location failed (status %d): %s", status, string(body))
    }
    // Parse and return
}
```

The helper returns status codes to callers because different operations interpret the same code differently — a 404 means "not found" in Read (remove from state) but is an error in Create.

---

### OIDC Token Source (`oidc.go`)

#### Token Lifecycle

```go
type OIDCTokenSource struct {
    tokenURL   string
    clientID   string
    scope      string
    username   string
    password   string
    httpClient *http.Client

    mu          sync.Mutex
    cachedToken string
    tokenExpiry time.Time
}
```

Key design decisions:

1. **Thread-safe** — `sync.Mutex` protects cached token since Terraform may call resources in parallel
2. **Pre-expiry refresh** — Refreshes `oidcTokenPreExpiry` (30s) before actual expiry to prevent mid-request failures
3. **Lazy initialization** — First call to `Token(ctx)` fetches; subsequent calls return cache
4. **Configurable scope** — The `scope` field (default `openid`) is sent in the token request, allowing different deployments to request additional claims
5. **JWT introspection** — `InstanceIDs(ctx)` decodes the JWT payload to extract custom `instanceIds` claim for auto-discovery

```go
func (s *OIDCTokenSource) Token(ctx context.Context) (string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.cachedToken != "" && time.Now().Before(s.tokenExpiry.Add(-oidcTokenPreExpiry)) {
        return s.cachedToken, nil  // return cached
    }
    // Fetch new token via password grant...
}
```

---

### IAM Client (`iam.go`)

#### Design

```go
type IAMClient struct {
    endpoint   string
    region     string
    httpClient *http.Client
}
```

The `region` field is configurable via the provider's `iam_region` attribute (defaults to `us-east-1`).

#### SigV4 Signing

Both the IAM and S3 clients use AWS Signature Version 4. The signing primitives (`sha256Hex`, `hmacSHA256`, `getSignatureKey`) are extracted into `sigv4.go` and shared between both clients.

```go
// sigv4.go — shared by IAM (service="iam") and S3 (service="s3")
func sha256Hex(data []byte) string { ... }
func hmacSHA256(key, data []byte) []byte { ... }
func getSignatureKey(secretKey, datestamp, region, service string) []byte { ... }
```

Each client has its own `doSignedRequest` method that uses these primitives with its own service name, headers, and payload format.

The IAM helper handles API version injection, signing, and XML error parsing. All 9 IAM operations (CreateUser, GetUser, DeleteUser, PutUserPolicy, GetUserPolicy, DeleteUserPolicy, CreateAccessKey, ListAccessKeys, DeleteAccessKey) are thin wrappers around this helper.

The low-level signing primitives (`sha256Hex`, `hmacSHA256`, `getSignatureKey`) live in `sigv4.go` and are shared between the IAM and S3 clients.

#### Error Handling

IAM errors come as XML with structured error codes:

```go
if resp.StatusCode >= 400 {
    var iamErr iamErrorResponse
    if xmlErr := xml.Unmarshal(respBody, &iamErr); xmlErr == nil && iamErr.Error.Code != "" {
        return nil, fmt.Errorf("%s: %s", iamErr.Error.Code, iamErr.Error.Message)
    }
    return nil, fmt.Errorf("IAM request failed (status %d): %s", resp.StatusCode, string(respBody))
}
```

Callers check for specific error codes like `NoSuchEntity` to distinguish "not found" from real errors:

```go
if strings.Contains(err.Error(), "NoSuchEntity") {
    return nil, nil  // resource doesn't exist
}
```

---

### S3 Client (`s3.go`, `s3_lifecycle.go`)

#### Design

```go
type S3Client struct {
    endpoint   string
    region     string
    httpClient *http.Client
}
```

The S3 client performs path-style bucket operations (PUT, HEAD, DELETE on `/{bucket}`) and lifecycle configuration. It uses the shared `sigv4.go` helpers with `service="s3"` and the additional `X-Amz-Content-Sha256` header required by S3.

S3 operations include retry logic for location propagation: after a location is created in the management API, S3 may take time to recognize it. The client retries `CreateBucket` when it receives an `InvalidLocationConstraint` error.

#### Endpoint Discovery

The S3 endpoint can be configured explicitly via the `s3_endpoint` provider attribute or `ARTESCA_S3_ENDPOINT` env var. If not set, it is derived automatically from the management endpoint:

```go
func DeriveS3Endpoint(managementEndpoint string) (string, error) {
    // "https://management.artesca.example.com" → "https://s3.artesca.example.com"
}
```

#### Operations

| Method | HTTP | Purpose |
|--------|------|---------|
| `CreateBucket` | `PUT /{bucket}` | Creates bucket with optional LocationConstraint XML body |
| `HeadBucket` | `HEAD /{bucket}` | Returns `(true, nil)` if exists, `(false, nil)` for 404 |
| `DeleteBucket` | `DELETE /{bucket}` | Idempotent — ignores NoSuchBucket (404) |

---

### Why Different Error Patterns Per Client?

The Management client returns `(body, statusCode, error)` and lets callers decide. The IAM and S3 clients handle status codes internally and return `(body, error)` or typed results.

This is intentional:
- **Management API** uses HTTP status codes as the primary error signal. Different operations need different status handling (201 vs 200 for create, 204 vs 200 for delete).
- **IAM API** uses structured XML error codes inside the response body. The status code is secondary — `NoSuchEntity` is the meaningful signal, not 404.
- **S3 API** also uses structured XML error codes (`BucketAlreadyOwnedByYou`, `NoSuchBucket`, `InvalidLocationConstraint`), handled internally with specific retry and tolerance logic.

---

## Resource Pattern

### Standard Resource Structure

Every resource follows this exact pattern:

```
internal/resources/<name>/
├── model.go      # Data model with tfsdk tags
└── resource.go   # Schema, Configure, CRUD, conversion helpers
```

#### Model (`model.go`)

```go
type LocationResourceModel struct {
    Name         types.String `tfsdk:"name"`
    LocationType types.String `tfsdk:"location_type"`
    ObjectID     types.String `tfsdk:"object_id"`
    Details      *LocationDetailsModel `tfsdk:"details"`
}
```

Models map directly to the Terraform schema via `tfsdk` struct tags. No business logic lives here.

#### Resource (`resource.go`)

Every resource implements these methods in order:

```go
// 1. Type assertion
var _ resource.Resource = &LocationResource{}

// 2. Struct holds client reference
type LocationResource struct {
    client *client.ManagementClient
}

// 3. Constructor
func NewLocationResource() resource.Resource { return &LocationResource{} }

// 4. Metadata — sets type name
func (r *LocationResource) Metadata(...)

// 5. Schema — defines attributes
func (r *LocationResource) Schema(...)

// 6. Configure — extracts client from ProviderClients
func (r *LocationResource) Configure(...) {
    providerData, ok := req.ProviderData.(*client.ProviderClients)
    r.client = providerData.Management
}

// 7. CRUD operations
func (r *LocationResource) Create(...)
func (r *LocationResource) Read(...)
func (r *LocationResource) Update(...)
func (r *LocationResource) Delete(...)

// 8. Optional: ImportState
func (r *LocationResource) ImportState(...)

// 9. Conversion helpers (private)
func modelToAPILocation(...)
func apiLocationToModel(...)
```

#### CRUD Pattern

```go
func (r *LocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // a. Read plan data
    var plan LocationResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // b. Convert to API type and call client
    apiLoc := modelToAPILocation(ctx, &plan)
    created, err := r.client.CreateLocation(ctx, apiLoc)
    if err != nil {
        resp.Diagnostics.AddError("Error creating location", err.Error())
        return
    }

    // c. Convert response back and update state
    apiLocationToModel(ctx, created, &plan)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### Resource Summary

| Resource | Client | CRUD | Import | Notes |
|----------|--------|------|--------|-------|
| `artesca_account` | Management | CR_D | By name | Keys auto-generated |
| `artesca_bucket` | S3 | CR_D | By name | All fields ForceNew, retries for location propagation |
| `artesca_endpoint` | Management | CR_D | By hostname | Immutable (both fields ForceNew) |
| `artesca_location` | Management | CRUD | By name | 20+ detail fields, sensitive field preservation |
| `artesca_replication` | Management | CRUD | By stream_id | Nested source/destination/locations blocks |
| `artesca_user` | IAM | CR_D | By username | Uses account credentials for auth |
| `artesca_user_access_key` | IAM | CR_D | No | Secret key only available at creation |
| `artesca_user_policy` | IAM | CRUD | No | Policy document is the only updatable field |
| `artesca_bucket_workflow_expiration` | Management | CUD* | No | *Read preserves state (no API read) |
| `artesca_bucket_workflow_replication` | Management | CUD* | No | *Read preserves state (no API read) |
| `artesca_bucket_workflow_transition` | Management | CUD* | No | *Read preserves state (no API read) |

---

## Adding New API Calls

### Step-by-Step Guide

#### For Management API Resources

**Step 1: Add client methods to `management_<domain>.go`**

```go
func (c *ManagementClient) CreateWidget(ctx context.Context, w *Widget) (*Widget, error) {
    path := fmt.Sprintf("/config/%s/widget", c.InstanceID)
    body, status, err := c.doRequest(ctx, http.MethodPost, path, w)
    if err != nil {
        return nil, err
    }
    if status != http.StatusCreated && status != http.StatusOK {
        return nil, fmt.Errorf("create widget failed (status %d): %s", status, string(body))
    }

    var created Widget
    if err := json.Unmarshal(body, &created); err != nil {
        return nil, fmt.Errorf("parsing create widget response: %w", err)
    }
    return &created, nil
}
```

The `doRequest` helper handles token injection, JSON marshaling, and HTTP execution.

**Step 2: Create resource files**

```
internal/resources/widget/
├── model.go
└── resource.go
```

Follow the standard resource pattern (see [Resource Pattern](#resource-pattern)).

**Step 3: Register in `provider.go`**

```go
import widget "github.com/scality/terraform-provider-scality-artesca/internal/resources/widget"

func (p *ArtescaProvider) Resources(...) []func() resource.Resource {
    return []func() resource.Resource{
        // ...existing resources...
        widget.NewWidgetResource,
    }
}
```

#### For IAM API Resources

Same pattern, but use `iam.go` and the `doSignedRequest` helper:

```go
func (c *IAMClient) CreateWidget(ctx context.Context, accessKey, secretKey, name string) (*iamWidget, error) {
    params := url.Values{
        "Action": {"CreateWidget"},
        "Name":   {name},
    }
    // Version is set automatically by doSignedRequest

    body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
    if err != nil {
        return nil, fmt.Errorf("create widget: %w", err)
    }

    var resp createWidgetResponse
    if err := xml.Unmarshal(body, &resp); err != nil {
        return nil, fmt.Errorf("parsing create widget response: %w", err)
    }
    return &resp.Result.Widget, nil
}
```

### Complexity Decision Tree

```
Is this an API call?
├─ Yes
│  ├─ Management API (OIDC token)?
│  │  └─ Add to management_<domain>.go, use doRequest helper
│  │     → Returns (body, statusCode, error)
│  │     → Caller handles status code interpretation
│  │
│  ├─ IAM API (SigV4, service="iam")?
│  │  └─ Add to iam.go, use doSignedRequest helper
│  │     → Returns (body, error)
│  │     → Helper handles status codes and XML errors
│  │
│  └─ S3 API (SigV4, service="s3")?
│     └─ Add to s3.go or s3_lifecycle.go, use doSignedS3Request helper
│        → Uses shared sigv4.go primitives
│        → Path-style operations on /{bucket}
│
└─ No
   └─ Is this a Terraform resource?
      ├─ Read-only → Create data source in internal/resources/
      └─ Mutable → Create resource with model.go + resource.go
```

---

## Error Handling Strategy

### Principles

1. **Wrap errors with context** using `%w`
2. **Meaningful messages** that help users solve problems
3. **Distinguish "not found" from real errors** — return `nil` for missing resources
4. **Never swallow errors**

### Error Wrapping

```go
// Good: Wrap with context, preserve error chain
if err != nil {
    return nil, fmt.Errorf("create location failed: %w", err)
}

// Bad: Lose error chain
if err != nil {
    return nil, fmt.Errorf("create location failed: %s", err)
}
```

Using `%w` preserves the error chain so callers can use `errors.Is()` and `errors.As()` to inspect root causes.

### User-Facing Error Messages

In resources, errors go through diagnostics with a summary and detail:

```go
resp.Diagnostics.AddError(
    "Error creating location",   // Summary — shown bold
    err.Error(),                  // Detail — includes wrapped context
)
```

The provider's Configure method provides especially actionable messages:

```go
resp.Diagnostics.AddError(
    "OIDC Authentication Failed",
    fmt.Sprintf("Failed to obtain initial OIDC token: %s\n\nPlease verify your oidc_url, username, and password.", err),
)
```

### Not-Found Handling

Resources that read from the overlay return `nil` when not found. The resource's Read method then removes the resource from state:

```go
loc, err := r.client.GetLocation(ctx, state.Name.ValueString())
if loc == nil {
    resp.State.RemoveResource(ctx)  // triggers recreation on next apply
    return
}
```

IAM resources check for `NoSuchEntity` in the XML error:

```go
if strings.Contains(err.Error(), "NoSuchEntity") {
    return nil, nil
}
```

---

## State Management

### Drift Detection

On every `plan` or `apply`, Terraform calls `Read()` for each resource. The Read method checks the real infrastructure state against what Terraform believes:

```go
func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state AccountResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    account, err := r.client.GetAccount(ctx, state.Name.ValueString())
    if account == nil {
        resp.State.RemoveResource(ctx)  // deleted outside Terraform
        return
    }

    // Update state with current values
    state.ARN = types.StringValue(account.ARN)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### Sensitive Fields Preservation

Some API responses redact sensitive fields (secret keys, passwords). To prevent false drift, these are preserved from state.

For fields that are empty in the API response (like account credentials):

```go
if account.AccessKey == "" {
    // Keep existing state value
} else {
    state.AccessKey = types.StringValue(account.AccessKey)
}
```

For fields where the API returns a redacted non-empty value (like location secret keys), the plan value must be unconditionally preserved:

```go
planSecretKey := plan.Details.SecretKey
// ... API call ...
if !planSecretKey.IsNull() && !planSecretKey.IsUnknown() {
    plan.Details.SecretKey = planSecretKey
}
```

Combined with the `UseStateForUnknown` plan modifier:

```go
"secret_key": schema.StringAttribute{
    Computed:  true,
    Sensitive: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

Server-default fields (e.g., location `region` defaults to `us-east-1`) use Optional+Computed with `UseStateForUnknown` to avoid null-vs-value drift.

### Write-Only Fields

The `artesca_user_access_key` resource's `secret_access_key` is only available at creation time. The IAM API never returns it again. `UseStateForUnknown` preserves the value from the initial creation in all subsequent plans.

### State-Only Read

Workflow resources (`expiration` and `transition`) have no read API endpoint. Their Read method simply preserves the current state:

```go
func (r *WorkflowExpirationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WorkflowExpirationResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    // No API call — preserve state as-is
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

---

## Security Patterns

### 1. Credential Handling

Never log credentials:

```go
// Good: Log operation context, not secrets
tflog.Debug(ctx, "Creating IAM user", map[string]any{
    "username": plan.Username.ValueString(),
})

// Bad: Would expose credentials in logs
tflog.Debug(ctx, "Creating IAM user", map[string]any{
    "access_key": plan.AccountAccessKey.ValueString(), // NEVER
})
```

### 2. Schema-Level Sensitivity

All credential fields are marked `Sensitive: true`:

```go
"account_access_key": schema.StringAttribute{
    Required:  true,
    Sensitive: true,  // masked in terraform show/plan output
},
```

This covers: provider password, account access/secret keys, location access/secret keys and passwords, user account credentials, and generated access keys.

### 3. Environment Variable Support

All provider configuration supports environment variable overrides:

```bash
export ARTESCA_MANAGEMENT_ENDPOINT="https://management.artesca.example.com"
export ARTESCA_OIDC_URL="https://10.0.0.1:8443"
export ARTESCA_USERNAME="admin"
export ARTESCA_PASSWORD="secret"
export ARTESCA_INSECURE_SKIP_VERIFY="true"
export ARTESCA_IAM_REGION="us-east-1"
export ARTESCA_S3_ENDPOINT="https://s3.artesca.example.com"
export ARTESCA_OIDC_SCOPE="openid"
```

The S3 endpoint is optional — if not set, it is derived from the management endpoint by replacing `management.` with `s3.` in the hostname.

The provider reads config first, falls back to environment:

```go
func envOrValue(val types.String, envVar string) string {
    if !val.IsNull() && !val.IsUnknown() {
        return val.ValueString()
    }
    return os.Getenv(envVar)
}
```

Benefits: credentials are not committed to version control, CI/CD integration is simpler, and follows 12-factor app principles.

### 4. TLS Configuration

TLS verification is enabled by default. The `insecure_skip_verify` option exists for development environments but should never be used in production:

```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: insecureSkipVerify,
    },
}
```

---

## Testing Strategy

### Unit Tests

All client packages have httptest-based unit tests that verify request signing, HTTP method/path, error handling, and response parsing without a real ARTESCA instance:

```bash
go test ./... -v
```

### Integration Tests

A full 12-resource create→plan→destroy lifecycle test runs against a real ARTESCA instance. See `tests/integration/README.md` for setup.

```bash
tests/integration/run.sh
```

The integration test creates all resource types, verifies no drift on a second plan, then destroys everything. It runs on self-hosted GitHub Actions runners via `.github/workflows/integration.yml`.

### Acceptance Testing

Terraform plugin acceptance tests run against a real ARTESCA instance with `TF_ACC=1`:

```go
func TestAccArtescaAccount_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccArtescaAccountConfig("test-account"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("artesca_account.test", "name", "test-account"),
                    resource.TestCheckResourceAttrSet("artesca_account.test", "access_key"),
                ),
            },
        },
    })
}
```

### Makefile Targets

```makefile
test:     go test ./... -v              # Unit tests
testacc:  TF_ACC=1 go test ./... -v     # Acceptance tests (requires real instance)
lint:     golangci-lint run ./...        # Static analysis
fmt:      gofmt -w .                     # Format
```

---

## Future Extensibility

### Adding More Resources

Pattern is established — for each new resource:
1. Add client methods to `management.go` or `iam.go`
2. Create `internal/resources/<name>/model.go` and `resource.go`
3. Register in `provider.go`

### Data Sources

Currently no data sources are implemented. Future candidates:

```go
func (p *ArtescaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // data.artesca_account — look up existing account
        // data.artesca_config_overlay — full config overlay
    }
}
```

### Client Enhancements

Adding features to the helper methods benefits all operations automatically:

```go
// Future: Add retry logic to doRequest
func (c *ManagementClient) doRequest(...) ([]byte, int, error) {
    for attempt := 0; attempt < maxRetries; attempt++ {
        body, status, err := c.executeRequest(...)
        if err == nil && status < 500 {
            return body, status, nil
        }
        time.Sleep(backoff(attempt))
    }
    return nil, 0, fmt.Errorf("max retries exceeded: %w", lastErr)
}
// All management operations get retry automatically.
```

---

## Complexity Management Rules

### When to Abstract

Extract when:
- Used 3+ times
- Changes together always
- Clear single responsibility
- Reduces complexity

Don't extract when:
- Used only twice
- Needs many parameters
- Changes independently
- Abstraction is more complex than duplication

### Complexity Budget

Target:
- Client method: < 50 lines
- Resource method: < 80 lines
- Helper function: < 30 lines

If exceeded:
1. Can you extract a helper?
2. Can you simplify the logic?
3. Is this inherently complex? (document well)

### Code Review Checklist

When adding new code:

- [ ] Uses `context.Context` as first parameter
- [ ] Constants defined for magic values
- [ ] Errors wrapped with `%w`
- [ ] Sensitive data marked in schema
- [ ] HTTP requests use `NewRequestWithContext`
- [ ] Follows existing patterns (Management helper or IAM helper)
- [ ] No credentials logged
- [ ] Resource follows model.go + resource.go structure

---

## Summary

### Key Takeaways

1. **Three API surfaces, three clients** — Management (OIDC/JSON), IAM (SigV4/XML), and S3 (SigV4/XML), bundled in ProviderClients
2. **Shared SigV4 signing** — `sigv4.go` provides HMAC/SHA256 primitives used by both IAM and S3 clients
3. **DRY with pragmatism** — Helper methods where patterns are identical; separate when abstraction adds complexity
4. **Context-first** — All public methods accept context for cancellation, timeouts, and logging
5. **Constants over magic values** — No hardcoded strings; configurable where appropriate (e.g., `iam_region`)
6. **Overlay read pattern** — Single GET for all management resources, extract what's needed
7. **Sensitive field preservation** — UseStateForUnknown + plan value preservation prevents false drift from redacted API responses
8. **Consistent resource structure** — Every resource is model.go + resource.go with identical CRUD patterns

Design Philosophy:

> "Make it work, make it right, make it fast" — in that order.

The goal is code that a new developer can understand in 30 minutes and confidently modify.

---

Document Version: 1.1
Last Updated: 2026-04-24
Status: Living Document (update as architecture evolves)
