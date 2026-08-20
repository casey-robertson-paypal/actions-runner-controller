package actionsgithubcom

import (
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

type RecurrenceRule struct {
	Frequency string
	UntilTime time.Time
}

type Period struct {
	StartTime time.Time
	EndTime   time.Time
}

func (r *Period) String() string {
	if r == nil {
		return ""
	}

	return r.StartTime.Format(time.RFC3339) + "-" + r.EndTime.Format(time.RFC3339)
}

func MatchSchedule(now time.Time, startTime, endTime time.Time, recurrenceRule RecurrenceRule) (*Period, *Period, error) {
	return calculateActiveAndUpcomingRecurringPeriods(
		now,
		startTime,
		endTime,
		recurrenceRule.Frequency,
		recurrenceRule.UntilTime,
	)
}

// freqMinDuration is, for each recurring frequency, the shortest possible interval
// between two consecutive occurrences (e.g. the shortest calendar month is 28
// days). An override's duration must fit within that interval or it would overlap
// its own next occurrence. This is a fixed invariant rather than one derived from
// the current reconciliation time: computing it from `now` (as the legacy
// summerwind/actions-runner-controller ScheduledOverride implementation did) made
// validity depend on when the check happened to run — e.g. a 30-day Monthly
// override would pass when checked in a 31-day month and fail when checked in
// February.
var freqMinDuration = map[string]time.Duration{
	"Daily":   24 * time.Hour,
	"Weekly":  7 * 24 * time.Hour,
	"Monthly": 28 * 24 * time.Hour,
	"Yearly":  365 * 24 * time.Hour,
}

func calculateActiveAndUpcomingRecurringPeriods(now, startTime, endTime time.Time, frequency string, untilTime time.Time) (*Period, *Period, error) {
	var freqValue rrule.Frequency

	switch frequency {
	case "Daily":
		freqValue = rrule.DAILY
	case "Weekly":
		freqValue = rrule.WEEKLY
	case "Monthly":
		freqValue = rrule.MONTHLY
	case "Yearly":
		freqValue = rrule.YEARLY
	case "":
		if now.Before(startTime) {
			return nil, &Period{StartTime: startTime, EndTime: endTime}, nil
		}

		if now.Before(endTime) {
			return &Period{StartTime: startTime, EndTime: endTime}, nil, nil
		}

		return nil, nil, nil
	default:
		return nil, nil, fmt.Errorf(`invalid freq %q: It must be one of "Daily", "Weekly", "Monthly", and "Yearly"`, frequency)
	}

	overrideDuration := endTime.Sub(startTime)
	if minDuration := freqMinDuration[frequency]; overrideDuration > minDuration {
		return nil, nil, fmt.Errorf("override's duration %s must be equal to or shorter than the minimum interval implied by freq %q (%s)", overrideDuration, frequency, minDuration)
	}

	rrule, err := rrule.NewRRule(rrule.ROption{
		Freq:    freqValue,
		Dtstart: startTime,
		Until:   untilTime,
	})
	if err != nil {
		return nil, nil, err
	}

	overrideDurationBefore := now.Add(-overrideDuration + 1)
	activeOverrideStarts := rrule.Between(overrideDurationBefore, now, true)

	var active *Period

	if len(activeOverrideStarts) > 1 {
		return nil, nil, fmt.Errorf("[bug] unexpted number of active overrides found: %v", activeOverrideStarts)
	} else if len(activeOverrideStarts) == 1 {
		active = &Period{
			StartTime: activeOverrideStarts[0],
			EndTime:   activeOverrideStarts[0].Add(overrideDuration),
		}
	}

	// Find the very next occurrence after now, however far out it is, rather than
	// bounding the search to one calendar period after now. A fixed one-period
	// window can miss the next occurrence entirely for recurrences whose start day
	// doesn't exist in every period (e.g. Monthly on the 31st, or Yearly on Feb
	// 29): rrule skips periods without a matching day, so the next real occurrence
	// can fall outside a naively-computed single-period window. After() is bounded
	// by the rule's Until (or ~290 years when unset), which is a sufficient hard
	// stop on its own.
	var next *Period

	if nextStart := rrule.After(now, false); !nextStart.IsZero() {
		next = &Period{
			StartTime: nextStart,
			EndTime:   nextStart.Add(overrideDuration),
		}
	}

	return active, next, nil
}
