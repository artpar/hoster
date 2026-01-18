# F009: Billing Integration

## Overview

Report deployment usage events to APIGate for billing purposes. APIGate handles payment processing, subscription management, and invoicing.

## User Stories

### US-1: As a customer, I want my deployment usage tracked for accurate billing

**Acceptance Criteria:**
- Usage events recorded when deployments are created, started, stopped, deleted
- Events include resource information (deployment ID, type, timestamp)
- Events are reliably delivered to APIGate

### US-2: As an operator, I want to enforce plan limits before resource creation

**Acceptance Criteria:**
- Plan limits checked before creating deployments
- Clear error message when limits exceeded
- No partial resource creation on limit failure

### US-3: As an operator, I want usage events batched for efficiency

**Acceptance Criteria:**
- Events stored locally before reporting
- Batch reporting on configurable interval
- Retry logic for failed deliveries

## Technical Specification

### Usage Event Types

| Event Type | Trigger | Billable |
|------------|---------|----------|
| `deployment_created` | POST /deployments | Yes - starts subscription |
| `deployment_started` | POST /deployments/:id/start | Yes - compute begins |
| `deployment_stopped` | POST /deployments/:id/stop | No - compute paused |
| `deployment_deleted` | DELETE /deployments/:id | No - ends subscription |

### Usage Event Structure

```go
// internal/core/domain/usage.go

type MeterEvent struct {
    ID           string            `json:"id"`
    UserID       string            `json:"user_id"`
    EventType    string            `json:"event_type"`
    ResourceID   string            `json:"resource_id"`
    ResourceType string            `json:"resource_type"`
    Quantity     int64             `json:"quantity"`      // Optional: for metered usage
    Metadata     map[string]string `json:"metadata"`
    Timestamp    time.Time         `json:"timestamp"`
    ReportedAt   *time.Time        `json:"reported_at"`   // Nil until reported
}
```

### Plan Limit Validation (Pure Core)

```go
// internal/core/limits/validation.go

type ValidationResult struct {
    Allowed bool
    Reason  string
}

// ValidateDeploymentCreation checks if user can create a deployment
func ValidateDeploymentCreation(
    limits auth.PlanLimits,
    currentDeployments int,
    requestedResources Resources,
) ValidationResult {
    if currentDeployments >= limits.MaxDeployments {
        return ValidationResult{
            Allowed: false,
            Reason:  fmt.Sprintf("deployment limit reached: %d/%d", currentDeployments, limits.MaxDeployments),
        }
    }

    // Check resource limits
    if requestedResources.CPUCores > limits.MaxCPUCores {
        return ValidationResult{
            Allowed: false,
            Reason:  fmt.Sprintf("CPU limit exceeded: %.1f/%.1f cores", requestedResources.CPUCores, limits.MaxCPUCores),
        }
    }

    // ... similar checks for memory, disk

    return ValidationResult{Allowed: true}
}

// Resources represents requested compute resources
type Resources struct {
    CPUCores float64
    MemoryMB int64
    DiskMB   int64
}
```

### Billing Client Interface (Shell)

```go
// internal/shell/billing/client.go

type Client interface {
    // MeterUsage reports a single usage event
    MeterUsage(ctx context.Context, event domain.MeterEvent) error

    // MeterUsageBatch reports multiple events at once
    MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error
}

type APIGateClient struct {
    baseURL    string
    httpClient *http.Client
    apiKey     string
}

func NewAPIGateClient(baseURL, apiKey string) *APIGateClient
```

### Usage Event Storage (Shell)

```go
// internal/shell/store/store.go (interface addition)

type Store interface {
    // ... existing methods

    // Usage events
    CreateUsageEvent(ctx context.Context, event domain.MeterEvent) error
    GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error)
    MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error
}
```

### Background Reporter

```go
// internal/shell/billing/reporter.go

type Reporter struct {
    store    store.Store
    client   Client
    interval time.Duration
    batchSize int
}

func (r *Reporter) Start(ctx context.Context) {
    ticker := time.NewTicker(r.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            r.reportBatch(ctx)
        }
    }
}

func (r *Reporter) reportBatch(ctx context.Context) error {
    events, err := r.store.GetUnreportedEvents(ctx, r.batchSize)
    if err != nil || len(events) == 0 {
        return err
    }

    if err := r.client.MeterUsageBatch(ctx, events); err != nil {
        slog.Error("failed to report usage events", "error", err)
        return err
    }

    ids := make([]string, len(events))
    for i, e := range events {
        ids[i] = e.ID
    }

    return r.store.MarkEventsReported(ctx, ids, time.Now())
}
```

### Integration Points

```go
// internal/shell/api/resources/deployment.go

func (r *DeploymentResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
    ctx := req.PlainRequest.Context()
    authCtx := auth.FromContext(ctx)

    // Get current deployment count
    count, err := r.store.CountDeploymentsByCustomer(ctx, authCtx.UserID)
    if err != nil {
        return nil, err
    }

    // Validate limits
    result := limits.ValidateDeploymentCreation(authCtx.PlanLimits, count, resources)
    if !result.Allowed {
        return nil, api2go.NewHTTPError(nil, result.Reason, http.StatusForbidden)
    }

    // Create deployment
    deployment, err := r.orchestrator.Deploy(ctx, ...)
    if err != nil {
        return nil, err
    }

    // Record usage event
    event := domain.MeterEvent{
        ID:           uuid.New().String(),
        UserID:       authCtx.UserID,
        EventType:    "deployment_created",
        ResourceID:   deployment.ID,
        ResourceType: "deployment",
        Metadata: map[string]string{
            "template_id": deployment.TemplateID,
            "plan_id":     authCtx.PlanID,
        },
        Timestamp: time.Now(),
    }
    if err := r.store.CreateUsageEvent(ctx, event); err != nil {
        slog.Error("failed to record usage event", "error", err)
        // Don't fail the request - event will be lost but deployment succeeded
    }

    return &Response{Res: deployment}, nil
}
```

### Database Migration

```sql
-- internal/shell/store/migrations/004_usage_events.up.sql

CREATE TABLE IF NOT EXISTS usage_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    quantity INTEGER DEFAULT 1,
    metadata TEXT, -- JSON
    timestamp TEXT NOT NULL,
    reported_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_usage_events_unreported
    ON usage_events(reported_at)
    WHERE reported_at IS NULL;

CREATE INDEX idx_usage_events_user_time
    ON usage_events(user_id, timestamp);
```

### Configuration

```yaml
billing:
  enabled: true
  apigate_url: http://apigate:8080
  api_key: ${APIGATE_API_KEY}
  report_interval: 60s
  batch_size: 100
  retry_attempts: 3
  retry_delay: 5s
```

## Test Cases

### Unit Tests (internal/core/limits/)

```go
// validation_test.go
func TestValidateDeploymentCreation_WithinLimits(t *testing.T)
func TestValidateDeploymentCreation_DeploymentLimitReached(t *testing.T)
func TestValidateDeploymentCreation_CPULimitExceeded(t *testing.T)
func TestValidateDeploymentCreation_MemoryLimitExceeded(t *testing.T)
func TestValidateDeploymentCreation_DiskLimitExceeded(t *testing.T)
func TestValidateDeploymentCreation_MultipleViolations(t *testing.T)
```

### Integration Tests (internal/shell/)

```go
// billing/client_test.go
func TestAPIGateClient_MeterUsage(t *testing.T)
func TestAPIGateClient_MeterUsageBatch(t *testing.T)
func TestAPIGateClient_RetryOnError(t *testing.T)

// billing/reporter_test.go
func TestReporter_BatchReporting(t *testing.T)
func TestReporter_MarkReported(t *testing.T)
func TestReporter_HandlesErrors(t *testing.T)

// store/usage_test.go
func TestStore_CreateUsageEvent(t *testing.T)
func TestStore_GetUnreportedEvents(t *testing.T)
func TestStore_MarkEventsReported(t *testing.T)
```

## Files to Create

- `internal/core/domain/usage.go` - MeterEvent type
- `internal/core/limits/validation.go` - Plan limit validation (pure)
- `internal/core/limits/validation_test.go` - Tests
- `internal/shell/billing/client.go` - APIGate billing client
- `internal/shell/billing/client_test.go` - Tests
- `internal/shell/billing/reporter.go` - Background event reporter
- `internal/shell/billing/reporter_test.go` - Tests
- `internal/shell/store/migrations/004_usage_events.up.sql`
- `internal/shell/store/migrations/004_usage_events.down.sql`

## Files to Modify

- `internal/shell/store/store.go` - Add usage event interface
- `internal/shell/store/sqlite.go` - Implement usage event storage
- `internal/shell/api/resources/deployment.go` - Record events on CRUD
- `cmd/hoster/config.go` - Add billing config
- `cmd/hoster/server.go` - Initialize billing client and reporter

## NOT Supported

- Payment processing (handled by APIGate)
- Subscription management (handled by APIGate)
- Invoice generation (handled by APIGate)
- Price calculation (handled by APIGate)
- Proration (handled by APIGate)
- Real-time usage queries (use APIGate API)
- Resource usage metering (CPU/memory usage over time)
- Bandwidth metering

## Dependencies

- F008: Authentication Integration (user context for events)
- ADR-005: APIGate Integration (billing API contract)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Event loss on crash | Events persisted to SQLite before reporting |
| APIGate unavailable | Batch with retry, events accumulate locally |
| Duplicate events | Idempotent event IDs, APIGate deduplication |
| Clock skew | Use server timestamp, APIGate records received time |
