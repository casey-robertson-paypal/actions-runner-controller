package actionsgithubcom

import (
	"time"

	"github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var scheduledOverrideLog = logf.Log.WithName("scheduled-override")

// maxScheduledOverrideRequeueAfter bounds how far out a scheduled-override requeue
// can be scheduled. Without a cap, a one-time override set far in the future would
// leave the controller waiting for calendar years; capping it just means we check
// back periodically instead, at no cost to correctness.
const maxScheduledOverrideRequeueAfter = 24 * time.Hour

// resolveMinRunners returns the MinRunners value that should be used for the
// AutoscalingListener at time now, taking any currently active
// v1alpha1.ScheduledOverride into account. The earlier a ScheduledOverride is
// listed in spec.ScheduledOverrides, the higher priority it has over later ones.
//
// A ScheduledOverride entry that fails to evaluate (e.g. its duration doesn't fit
// its recurrence frequency) is skipped and logged rather than failing the whole
// resolution: one bad entry must not take down the rest of the schedule, let alone
// the AutoscalingRunnerSet. If every entry is invalid, or none apply, the result is
// exactly as if spec.ScheduledOverrides were empty.
//
// The resolved value is clamped to spec.MaxRunners (when set) so that an override
// can never make the listener's minRunners exceed its maxRunners; see
// https://github.com/actions/actions-runner-controller/issues/2509.
//
// It also returns the time of the next schedule transition (an override starting
// or ending), if any, so callers can requeue reconciliation for exactly then
// instead of relying on a periodic poll.
func resolveMinRunners(spec *v1alpha1.AutoscalingRunnerSetSpec, now time.Time) (minRunners int, nextTransition time.Time, hasNextTransition bool) {
	if spec.MinRunners != nil {
		minRunners = *spec.MinRunners
	}

	var active, upcoming *Period
	var activeMinRunners *int

	for _, o := range spec.ScheduledOverrides {
		a, u, err := MatchSchedule(now, o.StartTime.Time, o.EndTime.Time, RecurrenceRule{
			Frequency: o.RecurrenceRule.Frequency,
			UntilTime: o.RecurrenceRule.UntilTime.Time,
		})
		if err != nil {
			scheduledOverrideLog.Error(err, "Skipping invalid scheduled override",
				"startTime", o.StartTime.Time, "endTime", o.EndTime.Time, "frequency", o.RecurrenceRule.Frequency)
			continue
		}

		// The first (i.e. earliest-listed) active override wins, per the
		// documented priority order.
		if a != nil && active == nil {
			active = a
			activeMinRunners = o.MinRunners
		}

		if u != nil && (upcoming == nil || u.StartTime.Before(upcoming.StartTime)) {
			upcoming = u
		}
	}

	if activeMinRunners != nil {
		minRunners = *activeMinRunners
	}

	if spec.MaxRunners != nil && minRunners > *spec.MaxRunners {
		minRunners = *spec.MaxRunners
	}

	switch {
	case active != nil && upcoming != nil:
		if active.EndTime.Before(upcoming.StartTime) {
			return minRunners, active.EndTime, true
		}
		return minRunners, upcoming.StartTime, true
	case active != nil:
		return minRunners, active.EndTime, true
	case upcoming != nil:
		return minRunners, upcoming.StartTime, true
	default:
		return minRunners, time.Time{}, false
	}
}

// scheduledOverrideRequeueAfter returns the ctrl.Result RequeueAfter duration for
// the next scheduled-override transition, capped at maxScheduledOverrideRequeueAfter.
// It returns 0 when there is no upcoming transition (i.e. no overrides configured).
func scheduledOverrideRequeueAfter(spec *v1alpha1.AutoscalingRunnerSetSpec, now time.Time) time.Duration {
	_, next, hasNext := resolveMinRunners(spec, now)
	if !hasNext {
		return 0
	}

	d := next.Sub(now)
	switch {
	case d <= 0:
		return time.Second
	case d > maxScheduledOverrideRequeueAfter:
		return maxScheduledOverrideRequeueAfter
	default:
		return d
	}
}
