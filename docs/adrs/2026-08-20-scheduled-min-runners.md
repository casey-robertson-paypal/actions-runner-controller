# Schedule-based `minRunners` overrides

**Status**: Proposed

## Context

`minRunners` on `AutoscalingRunnerSet` is static. Users who want to scale
runners to 0 outside business hours, or bump capacity ahead of a known
batch, have to edit the CRD themselves or run something external that
does it for them. Issues #3313 and #2986 both ask for this, mainly for
cost: idle self-hosted runner pods (and the nodes backing them) cost
money overnight and on weekends.

The predecessor project, summerwind/actions-runner-controller, had this
for the `HorizontalRunnerAutoscaler` CRD as `scheduledOverrides`. PR
#3564 ported that feature to `AutoscalingRunnerSet` in June 2024 but went
stale and was never merged.

## Decision

Add `spec.scheduledOverrides` to `AutoscalingRunnerSet`, reusing the
legacy `ScheduledOverride`/`RecurrenceRule` schema from PR #3564 with
`minReplicas` renamed to `minRunners`:

```go
type ScheduledOverride struct {
    StartTime      metav1.Time
    EndTime        metav1.Time
    MinRunners     *int
    RecurrenceRule RecurrenceRule
}

type RecurrenceRule struct {
    Frequency string // "", "Daily", "Weekly", "Monthly", "Yearly"
    UntilTime metav1.Time
}
```

Overrides are prioritized by list order: the earliest matching entry in
`scheduledOverrides` wins. If none match, `spec.minRunners` applies as
today.

### Considered alternatives

1. **In-CRD `scheduledOverrides`** (chosen). Declarative, GitOps-friendly,
   matches the field a user already sets. Feature parity with the legacy
   HRA behavior people are asking for in #3313/#2986. Cost: the
   controller has to evaluate schedules and requeue at the next
   boundary, which it doesn't do today.

2. **Scale subresource + external autoscaler (e.g. KEDA's cron
   scaler)**. Keeps ARC itself free of time-based logic and reuses an
   autoscaler most clusters already run. But `AutoscalingRunnerSet`
   doesn't have a `/scale` subresource or the selector labels an HPA/KEDA
   needs on the runner pods (see PR #3291, also unmerged), so this needs
   its own CRD change first. It also adds an operator dependency, and
   HPA-style "desired replica count" semantics don't map cleanly onto
   `minRunners`, which is a floor, not a target.

3. **External CronJob patching `spec.minRunners` directly**. Works today,
   no ARC changes needed. But the CronJob overwrites the field with no
   record of the original value, so there's nothing to roll back to if
   the job fails to run the "restore" side, and every team ends up
   writing a slightly different version of the same script.

Option 1 is the only one that's declarative, reversible, and doesn't
require a second controller.

### Implementation

- Effective `minRunners` is computed where `resourcebuilder.go` already
  builds the `AutoscalingListenerSpec` (`newAutoscalingListener`,
  ~resourcebuilder.go:124): instead of reading `spec.MinRunners`
  directly, resolve it against `scheduledOverrides` for the current
  time first, falling back to `spec.MinRunners` if none apply.
- No new status fields. `AutoscalingRunnerSetStatus` was trimmed to just
  `Phase` in #4557, and this doesn't reopen that decision — no
  `desiredMinRunners` or schedule-summary field.
- The existing listener spec-drift check in
  `autoscalingrunnerset_controller.go` (~line 358, `cmp.Equal(listener.Spec,
  desired.Spec)`) already deletes and recreates the `AutoscalingListener`
  whenever its spec changes. Because effective `minRunners` flows through
  that same spec, an override taking effect looks like any other spec
  change to this code path — no new recreation logic needed.
- The reconciler returns `RequeueAfter` set to the next override
  start/end boundary when `scheduledOverrides` is non-empty. If no
  overrides are configured, there's no time-based requeue — same as
  today. Existing users see no behavior change.
- All override validation runs at evaluation time, not admission. The
  effective `minRunners` is clamped to `spec.maxRunners`. An entry whose
  `endTime <= startTime`, or a recurring entry whose duration exceeds its
  fixed per-frequency invariant (Daily=24h, Weekly=7d, Monthly=28d,
  Yearly=365d — the minimum possible interval for each), is skipped with
  a logged error; other, valid entries in `scheduledOverrides` still
  apply, and if none apply, `spec.minRunners` holds. Reconciliation is
  never blocked. Upstream issue #2509 reports the `minRunners >
  maxRunners` case against the legacy HRA feature; this ADR fixes it
  rather than inheriting it.
- The legacy `schedule.go` computes the per-frequency interval from the
  reconciliation time, so the same 30-day Monthly override passes when
  evaluated in a 31-day month and fails in February — validity depends on
  when it's checked, which is wrong. The fixed invariants above replace
  that, so a given override is always valid or always invalid regardless
  of when it's evaluated. The bound is deliberately conservative: a
  Monthly override anchored on the 31st with a 30-day duration is
  rejected even though its real occurrences are always at least 31 days
  apart. Exact per-rule occurrence-gap validation is possible but isn't
  worth the complexity for that edge; the entry is skipped and logged
  like any other invalid one.
- CRD CEL validation rules could reject malformed overrides at admission
  instead. `apis/` has no `+kubebuilder:validation:XValidation` usage
  today, and this ADR doesn't introduce the first one; adding CEL rules
  here is left as a follow-up for maintainers to opt into.
- `scheduledOverrides` is exposed as a passthrough value in the
  `gha-runner-scale-set` Helm chart: `values.yaml`, template rendering,
  and a chart test.

## Consequences

The controller becomes time-aware for `AutoscalingRunnerSet` resources
that use this field. Recurring rules are evaluated using the timezone
offset embedded in `startTime`/`endTime`; note that Kubernetes serializes
`metav1.Time` to UTC, so the on-cluster object always shows UTC times
even if the offset was preserved during evaluation (this is the same
gotcha reported against the HRA version in #1916, not a new one).

Skipped invalid overrides are visible only in the controller logs. The
controllers in this API group have no EventRecorder plumbing today, so
surfacing them as Kubernetes warning events would be new wiring; that's
left as a follow-up if maintainers want resource-visible signals.

A fixed RFC3339 offset is not an IANA timezone: it carries no DST rules,
so a `Daily`/`Weekly` recurrence will drift by an hour against local wall
time across a DST transition. This is identical to the legacy HRA
feature's behavior, not a regression, and we're not adding an IANA
`timeZone` field now — it's scope creep against parity with the legacy
schema, and can be added later as an additive, optional field if needed.

Each time an override boundary crosses, the listener pod restarts. This
is one pod restart, not a runner restart, and it's cheap.

The port also fixes a legacy edge case: the occurrence search was
bounded to one calendar period ahead, so a Monthly rule anchored on day
31 or a Yearly rule anchored on Feb 29 could miss its next occurrence in
a shorter month or a non-leap year.

The field is optional and additive: existing `AutoscalingRunnerSet`
resources without `scheduledOverrides` behave exactly as they do today,
including the absence of periodic requeue.
