# CLAUDE.md - Picchu Kubernetes Operator

This document provides context and guidance for working on the Picchu codebase.

## Overview

**Picchu** is a Kubernetes operator that manages progressive deployments (canary releases, traffic shifting) for applications across multiple Kubernetes clusters. It uses Istio service mesh for traffic management and integrates with Prometheus/Datadog for SLO-based monitoring and automatic rollback.

Started in February 2019 by Bob Corsaro, Picchu has evolved over 7 years with 765+ commits from 18 contributors.

## Core Concepts

### Key Abstractions

| Concept | Description |
|---------|-------------|
| **Revision** | Defines what to deploy: container specs, targets, scaling config, release settings |
| **ReleaseManager** | Auto-created to track release state per app+target combination |
| **Cluster** | Represents a target Kubernetes cluster for deployments |
| **Incarnation** | Internal concept: Revision + Target + Status (not a CRD) |
| **Plan** | Encapsulates Kubernetes resource sync logic with `Apply()` method |

### Data Flow

```
1. User creates Revision CR in delivery cluster
                    │
                    ▼
2. RevisionReconciler creates ReleaseManager CRs (one per target)
                    │
                    ▼
3. ReleaseManagerReconciler:
   - Gets enabled Clusters for fleet
   - Creates PlanAppliers (one per cluster)
   - Creates Observers (one per cluster)
   - Builds IncarnationCollection from Revisions
                    │
                    ▼
4. ResourceSyncer coordinates operations:
   - Sync namespace, service account, RBAC
   - Tick incarnation state machines
   - Observe cluster state (ReplicaSets)
   - Sync application (Service, Istio resources)
   - Sync monitoring (ServiceMonitors, SLO rules)
   - Garbage collection
                    │
                    ▼
5. Incarnation state machine progresses:
   deploying -> deployed -> [testing] -> [canarying] ->
   pendingrelease -> releasing -> released
                    │
                    ▼
6. Traffic weights calculated and applied via VirtualService
                    │
                    ▼
7. Prometheus/Datadog SLO alerts queried for automatic rollback
```

### State Machine

Deployment lifecycle states:
```
created -> deploying -> deployed -> [pendingtest -> testing -> tested] ->
  [canarying -> canaried] -> pendingrelease -> releasing -> released ->
  retiring -> retired -> deleting -> deleted

With Datadog monitoring:
  [canaryingDatadog -> canariedDatadog]

Failure paths:
  failing -> failed
  timingout
```

## Architecture

### Project Structure

```
picchu/
├── api/v1alpha1/           # CRD type definitions
│   ├── revision_types.go   # Revision CRD spec
│   ├── releasemanager_types.go
│   ├── cluster_types.go
│   └── apis/               # Scheme registration
├── controllers/            # Controller implementations
│   ├── revision_controller.go      # Handles Revision CRs
│   ├── releasemanager_controller.go # Main orchestration
│   ├── cluster_controller.go       # Cluster CR handling
│   ├── incarnation.go              # Deployed revision with state
│   ├── syncer.go                   # Resource synchronization
│   ├── state.go                    # State machine
│   ├── scaling.go                  # Traffic scaling strategies
│   ├── plan/               # Reconciliation plans for remote clusters
│   │   ├── syncRevision.go         # ReplicaSet, configs, PDB
│   │   ├── syncApp.go              # Service, VirtualService, DestinationRule
│   │   ├── scaleRevision.go        # HPA, WPA, KEDA
│   │   └── syncSLORules.go         # PrometheusRules, ServiceLevels
│   ├── observe/            # Cluster state observation
│   ├── scaling/            # Linear/geometric scaling strategies
│   ├── schedule/           # Release schedule enforcement
│   └── utils/              # Shared utilities, remote client
├── plan/                   # Core plan abstraction
│   ├── common.go           # CreateOrUpdate, Plan interface
│   └── applier.go          # Single/concurrent cluster appliers
├── prometheus/             # Prometheus API client for SLO alerts
├── slack/                  # Slack notifications
├── sentry/                 # Sentry integration
├── client/                 # Generated clientset
├── mocks/                  # Test mocks
├── hack/                   # Build scripts
├── config/                 # Kustomize configs, CRD manifests
└── main.go                 # Entry point
```

### Key Components

1. **Controllers** (`controllers/`)
   - `RevisionReconciler`: Creates ReleaseManagers, mirrors configs
   - `ReleaseManagerReconciler`: Main orchestration, applies plans to clusters
   - `ClusterReconciler`: Manages cluster connectivity

2. **Plans** (`controllers/plan/`)
   - All implement `plan.Plan` with `Apply(ctx, client, cluster, log) error`
   - `SyncRevision`: Creates ReplicaSet, ConfigMaps, ExternalSecrets, PDB
   - `SyncApp`: Creates Service, DestinationRule, VirtualService, Sidecar
   - `ScaleRevision`: Creates HPA, WPA, or KEDA ScaledObject

3. **Observers** (`controllers/observe/`)
   - `ClusterObserver`: Single cluster state
   - `ConcurrentObserver`: Multi-cluster parallel observation

4. **Scaling Strategies** (`controllers/scaling/`)
   - `Linear`: Fixed increment traffic ramping
   - `Geometric`: Exponential (doubling) traffic ramping

### External Integrations

| Integration | Purpose | Package |
|-------------|---------|---------|
| **Istio** | Traffic routing (VirtualService, DestinationRule, Sidecar) | `istio.io/client-go` |
| **Prometheus Operator** | ServiceMonitor, PrometheusRule CRDs | `prometheus-operator/prometheus-operator` |
| **Sloth** | SLO generation (PrometheusServiceLevel) | `github.com/slok/sloth` |
| **Datadog** | Metrics and monitoring | `DataDog/datadog-api-client-go` |
| **KEDA** | Event-driven autoscaling | `kedacore/keda` |
| **External Secrets** | Secrets synchronization | `external-secrets/external-secrets` |
| **Slack** | Release notifications | `slack-go/slack` |

## Development

### Prerequisites

- Go 1.24+ (see `.tool-versions`)
- Access to a Kubernetes cluster
- kubectl configured

### Common Commands

```bash
make build          # Build binary
make test           # Run tests
make manifests      # Regenerate CRD YAMLs
make generate       # Regenerate deepcopy code
make docker-build   # Build container image
```

### Running Locally

```bash
# Run against local kubeconfig
go run main.go

# With specific flags
go run main.go \
  --metrics-addr=:8080 \
  --enable-leader-election=false \
  --concurrent-revisions=20 \
  --concurrent-release-managers=50
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./controllers/...

# Run with verbose output
go test -v ./controllers/plan/...
```

Tests use Ginkgo/Gomega BDD framework. Mocks are in `mocks/`, `plan/mocks/`, `prometheus/mocks/`.

## Key Files for Common Tasks

| Task | Key Files |
|------|-----------|
| Add CRD field | `api/v1alpha1/*_types.go`, then `make manifests generate` |
| Modify state machine | `controllers/state.go` |
| Change scaling behavior | `controllers/scaling.go`, `controllers/scaling/*.go` |
| Modify traffic routing | `controllers/plan/syncApp.go` |
| Add new K8s resource type | `plan/common.go` (CreateOrUpdate switch) |
| Change deployment sync | `controllers/plan/syncRevision.go` |
| Modify SLO/alerting | `controllers/plan/syncSLORules.go`, `prometheus/api.go` |
| Garbage collection | `controllers/garbagecollector/` |
| Add new controller | `main.go` (registration) |

## Important Patterns

### Plan Pattern

All Kubernetes resource operations are encapsulated in Plan structs:

```go
type SyncRevision struct {
    Revision   *picchu.Revision
    Namespace  string
    // ...
}

func (p *SyncRevision) Apply(ctx context.Context, cli client.Client, cluster *picchu.Cluster, log logr.Logger) error {
    // Use plan.CreateOrUpdate for all resources
}
```

### CreateOrUpdate Pattern

From `plan/README.md`: Don't assume resources exist. Use `controllerutil.CreateOrUpdate` for all resources with complete specs (no simple edits).

```go
// Good: Complete resource definition
plan.CreateOrUpdate(ctx, cli, &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{Name: "myservice", Namespace: ns},
    Spec: corev1.ServiceSpec{
        // Complete spec
    },
})
```

### Observer Pattern

Cluster state observation through `Observer` interface for tracking ReplicaSet status.

### State Handler Pattern

Each deployment state has a handler function returning the next state:

```go
func (s *Deployment) tickDeploying() (State, error) {
    if s.IsDeployed() {
        return StateDeployed, nil
    }
    return StateDeploying, nil
}
```

## Controller-Runtime Configuration

### Registered Controllers

| Controller | Resource | Notes |
|------------|----------|-------|
| `ClusterReconciler` | Cluster | Manages cluster connectivity |
| `ReleaseManagerReconciler` | ReleaseManager | Main orchestration |
| `ClusterSecretsReconciler` | ClusterSecrets | Secrets for clusters |
| `RevisionReconciler` | Revision | Creates ReleaseManagers |

### Concurrency Settings

- `--concurrent-revisions`: Max parallel Revision reconciles (default: 20)
- `--concurrent-release-managers`: Max parallel ReleaseManager reconciles (default: 50)

### Key Metrics

- `picchu_git_create_latency` - Time from git commit to incarnation create
- `picchu_git_deploy_latency` - Time from git commit to deployed
- `picchu_git_release_latency` - Time from git commit to released
- `picchu_revision_release_weight` - Current traffic weight per revision
- `picchu_incarnation_count` - Count of incarnations by state

## Historical Context

### Major Contributors

| Contributor | Expertise |
|-------------|-----------|
| Bob Corsaro | Project founder, core architecture, release management |
| Sofie Gonzalez | Datadog integration, canary monitoring, Slack alerts |
| David Osemwengie | KEDA scaling, controller-runtime upgrades |
| Micah Noland | API refinements, sidecars, scheduling |

### Key Architectural Decisions

1. **Incarnation/ReleaseManager Refactor (2019)**: Consolidated release logic into ReleaseManager controller
2. **Istio Integration (2019-2020)**: Deep integration for traffic management via VirtualServices
3. **Operator-SDK V1 Migration (2023)**: Modernized to kubebuilder v2 layout
4. **KEDA Integration (2024-2025)**: Event-driven autoscaling support
5. **Datadog Canary Monitoring (2025)**: Automatic rollback when monitors trigger

### Known Sensitive Areas

Based on commit history, these areas have seen reverts and careful iteration:

1. **Scaling/HPA Logic**: `CanRamp` and `CanRampTo` functions - test thoroughly
2. **Datadog Integration**: Still maturing, expect iteration
3. **Controller-Runtime Upgrades**: Plan carefully, have rollback strategy

## Commit Conventions

- Use imperative mood: "Add feature" not "Added feature"
- Reference Jira tickets: `[INF-123] Add new scaling strategy`
- Include PR numbers: `Fix race condition (#456)`
- Prefix experimental changes (they may get reverted)

## Institutional Knowledge (What Engineers Know)

> Auto-enriched by /learn-codebase-v2 on 2026-02-27
> Re-run `/learn-codebase-v2 /Users/ebarth/src/picchu --update` to refresh

### Known Gotchas — Read These Before Making Changes

1. **Scaling/HPA is a minefield**: Updating ReplicaSets during deployment rampup has been attempted and reverted **3 times** (Feb 2026). Making CanRamp smarter about HPA downscaling reverted **twice**. Using live data for CanRampTo was also reverted. Do not modify scaling logic without extensive testing and rollback readiness.

2. **Controller-runtime upgrades are risky**: Upgrading to k8s 1.31+ / controller-runtime beyond 0.18.0 has been attempted and reverted (Jun 2025). Do incrementally with extensive testing.

3. **Remote client cache is bugged**: `controllers/utils/api.go:29` — `checkCache` always looks up `client.ObjectKey{}` (empty key) instead of the actual key. The cache **never hits**. Every reconcile creates a new K8s client, causing unbounded memory growth.

4. **Holiday list is stale**: `controllers/schedule/schedule.go:33-84` — Hardcoded holidays end at January 1, 2024. The humane release schedule silently permits releases on all holidays after that date.

5. **Slack channel is a test channel**: `slack/slack_api.go:114-117` — Canary failure notifications go to `#eng-fredbottest`, not the actual operations channel.

6. **SLO rules must deploy to production clusters**: Multiple reverts confirmed this — deploying SLO/Prometheus rules only to the delivery cluster breaks canary monitoring.

7. **Empty TrafficPolicy objects cause issues**: Istio handling of empty TrafficPolicy in DestinationRules is fragile (reverted Sep 2023).

8. **genScalePlan mutates shared pointers**: `controllers/incarnation.go:494-496` — Mutates cpuTarget/memoryTarget on shared RevisionTarget, potentially corrupting HPA targets for other incarnations.

9. **deleteIfMarked has a logic bug**: `controllers/revision_controller.go:428` — `!ok && val != true` should be `!ok || val != true` (OR not AND).

10. **7 panic sites in production code**: `syncer.go:583`, `releasemanager_controller.go:310`, `revision_controller.go:366`, `utils/api.go:75-89`, `state.go:157-158`, `schedule.go:24`. Any panic crashes the entire operator.

### Implicit Contracts — Don't Break These

- **Revision defaults MUST be set before controllers access them**. The controller panics if `TTL == 0` after `Scheme.Default()`. If the admission webhook is down or bypassed, the operator crashes.
- **RemoteClient requires a Secret** with the same name/namespace as the Cluster CR in the delivery cluster containing valid kubeconfig.
- **CreateOrUpdate type switch** in `plan/common.go` only handles ~20 K8s resource types. Adding a new type requires extending the switch or it silently fails.
- **Every state string** in `ReleaseManagerRevisionStatus.State.Current` MUST have a handler in the state machine `handlers` map, or `tick` panics with nil function call.
- **Observer identifies tags** by reading `tag.picchu.medium.engineering` label from ReplicaSets. ReplicaSets without this label are invisible.
- **ScalableTargetAdapter.CanRampTo** reads `rm.Status.Revisions` creating an implicit dependency on accurate status reporting.

### Engineer Notes (TODO/FIXME)

- `TODO(bob): camelCase` — field naming inconsistency in `api/v1alpha1/common.go:67`
- `TODO(lyra): PodTemplate` — planned abstraction in `api/v1alpha1/common.go:210` never completed
- `TODO(bob): retry on conflict?` — GC update conflicts not retried (`controllers/garbagecollector.go:56`)
- `TODO(bob): return error when this works better` — error swallowed in GC (`controllers/garbagecollector.go:75`)
- `TODO(bob): errors on deployment interface aren't tested` (`controllers/state_test.go:3`) — known critical test gap
- `TODO(mk): possibility of generic timeout` — state machine timeout (`controllers/incarnation.go:321`)
- `TODO(micah): deprecate when AlertRules deprecated` — legacy support (`controllers/incarnation.go:973`)

---

## Risk & Quality

### Risk Map (Severity-Ranked)

| Severity | Area | Location | Issue |
|----------|------|----------|-------|
| **Critical** | Production panics | `syncer.go:583`, `releasemanager_controller.go:310`, `revision_controller.go:366` + 4 more | Crash entire operator, halt all deployments |
| **Critical** | Remote client cache | `controllers/utils/api.go:29` | Cache never hits → unbounded memory growth |
| **High** | Scaling/HPA volatility | `controllers/scaling.go`, `scaling/geometric.go` | 6+ reverts; infinite loop risk when factor ≤ 1 |
| **High** | Division by zero | `incarnation_controller.go:42` | `ClusterCount(true)` can be 0 → +Inf |
| **High** | context.TODO() | `revision_controller.go` (6 sites), `cluster_controller.go` (5 sites) | Bypasses cancellation, blocks workers |
| **Medium** | Stale holidays | `schedule/schedule.go:33-84` | Releases proceed on holidays since 2024 |
| **Medium** | Nil pointer risks | Multiple locations in `incarnation.go`, `scaling.go` | Revision deleted mid-reconcile |
| **Medium** | Unbounded caches | `prometheus/api.go`, `slack/slack_api.go`, `utils/api.go` | Maps grow indefinitely |

### Test Coverage

**Well-tested:**
- State machine transitions (`state_test.go` — exhaustive boolean permutations)
- All plan types (`controllers/plan/*_test.go` — 20+ test files)
- Scaling strategies (`controllers/scaling/*_test.go` — table-driven)
- Observer, syncer, schedule, garbage collector

**Critical gaps:**
- `revision_controller.go` — **NO tests** (contains 1 panic, 1 ignored error, 6 context.TODO)
- `releasemanager_controller.go` — **Minimal tests** (only getFaults; main Reconcile untested)
- `cluster_controller.go` — **NO tests**
- `controllers/utils/` — **NO tests** (contains the cache bug)
- `slack/`, `sentry/` — **NO tests**
- Datadog canary states — **Not covered** in state_test.go
- Istio ambient mesh — **Not covered** in syncApp_test.go

**Run tests:** `make test`

---

## Retrieval Guide

When you need current implementation details, retrieve them fresh:

| What You Need | How To Find It |
|---------------|----------------|
| Deployment flow | `grep -rn 'func.*Reconcile' controllers/*_controller.go` |
| State machine | `grep -rn 'State\|handlers\[' controllers/state.go` |
| Traffic routing | `grep -rn 'VirtualService\|DestinationRule\|Weight' controllers/plan/syncApp.go` |
| Scaling logic | `grep -rn 'CanRamp\|Scale\|HPA\|ScaledObject' controllers/scaling.go controllers/scaling/*.go` |
| Validation | `grep -rn 'Validate\|Default\|panic' api/v1alpha1/revision_webhook.go` |
| Configuration | `grep -rn 'flag\|Config\|env' main.go controllers/utils/config.go` |
| All panics | `grep -rn 'panic(' controllers/ plan/ main.go` |
| CRD types | `grep -rn 'type.*Spec struct' api/v1alpha1/*_types.go` |
| Plan implementations | `grep -rn 'func.*Apply' controllers/plan/*.go` |
| Metrics | `grep -rn 'NewHistogramVec\|NewGaugeVec\|MustRegister' controllers/*.go` |

---

## Related Services

Picchu is infrastructure that deploys and manages progressive rollouts for application services across Kubernetes clusters.

**External integrations:** Prometheus/Thanos (SLO queries), Datadog (canary monitoring), Slack (notifications), Sentry (releases), Istio (traffic routing), KEDA (autoscaling), External Secrets Operator

**Operationally deploys:** Services built in the Medium mono-repo (m2, rito, ml-rank, etc.)

---

## Documentation Maintenance

Update this file when:
- Adding new CRDs or significant API changes
- Changing controller architecture
- Adding new external integrations
- Modifying the state machine
- Updating development workflow
