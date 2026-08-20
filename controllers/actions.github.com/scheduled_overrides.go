package actionsgithubcom

import (
	"fmt"
	"time"

	"github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
)

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
// It also returns the time of the next schedule transition (an override starting
// or ending), if any, so callers can requeue reconciliation for exactly then
// instead of relying on a periodic poll.
func resolveMinRunners(spec *v1alpha1.AutoscalingRunnerSetSpec, now time.Time) (minRunners int, nextTransition time.Time, hasNextTransition bool, err error) {
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
			return 0, time.Time{}, false, fmt.Errorf("matching scheduled override starting %s: %w", o.StartTime.Time, err)
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

	switch {
	case active != nil && upcoming != nil:
		if active.EndTime.Before(upcoming.StartTime) {
			return minRunners, active.EndTime, true, nil
		}
		return minRunners, upcoming.StartTime, true, nil
	case active != nil:
		return minRunners, active.EndTime, true, nil
	case upcoming != nil:
		return minRunners, upcoming.StartTime, true, nil
	default:
		return minRunners, time.Time{}, false, nil
	}
}

// scheduledOverrideRequeueAfter returns the ctrl.Result RequeueAfter duration for
// the next scheduled-override transition, capped at maxScheduledOverrideRequeueAfter.
// It returns 0 when there is no upcoming transition (i.e. no overrides configured).
func scheduledOverrideRequeueAfter(spec *v1alpha1.AutoscalingRunnerSetSpec, now time.Time) (time.Duration, error) {
	_, next, hasNext, err := resolveMinRunners(spec, now)
	if err != nil {
		return 0, err
	}
	if !hasNext {
		return 0, nil
	}

	d := next.Sub(now)
	switch {
	case d <= 0:
		return time.Second, nil
	case d > maxScheduledOverrideRequeueAfter:
		return maxScheduledOverrideRequeueAfter, nil
	default:
		return d, nil
	}
}
