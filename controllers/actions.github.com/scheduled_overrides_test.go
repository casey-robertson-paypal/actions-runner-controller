package actionsgithubcom

import (
	"testing"
	"time"

	"github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResolveMinRunners(t *testing.T) {
	minRunners := 1
	now := time.Now()

	t.Run("no overrides", func(t *testing.T) {
		spec := &v1alpha1.AutoscalingRunnerSetSpec{MinRunners: &minRunners}

		got, _, hasNext, err := resolveMinRunners(spec, now)
		require.NoError(t, err)
		assert.Equal(t, 1, got)
		assert.False(t, hasNext)
	})

	t.Run("active override overrides minRunners and next transition is its end", func(t *testing.T) {
		overrideMinRunners := 5
		spec := &v1alpha1.AutoscalingRunnerSetSpec{
			MinRunners: &minRunners,
			ScheduledOverrides: []v1alpha1.ScheduledOverride{
				{
					StartTime:  metav1.NewTime(now.Add(-time.Hour)),
					EndTime:    metav1.NewTime(now.Add(time.Hour)),
					MinRunners: &overrideMinRunners,
				},
			},
		}

		got, next, hasNext, err := resolveMinRunners(spec, now)
		require.NoError(t, err)
		assert.Equal(t, 5, got)
		require.True(t, hasNext)
		assert.WithinDuration(t, now.Add(time.Hour), next, time.Second)
	})

	t.Run("upcoming override does not override minRunners but sets next transition to its start", func(t *testing.T) {
		overrideMinRunners := 5
		spec := &v1alpha1.AutoscalingRunnerSetSpec{
			MinRunners: &minRunners,
			ScheduledOverrides: []v1alpha1.ScheduledOverride{
				{
					StartTime:  metav1.NewTime(now.Add(time.Hour)),
					EndTime:    metav1.NewTime(now.Add(2 * time.Hour)),
					MinRunners: &overrideMinRunners,
				},
			},
		}

		got, next, hasNext, err := resolveMinRunners(spec, now)
		require.NoError(t, err)
		assert.Equal(t, 1, got)
		require.True(t, hasNext)
		assert.WithinDuration(t, now.Add(time.Hour), next, time.Second)
	})

	t.Run("invalid frequency returns an error", func(t *testing.T) {
		spec := &v1alpha1.AutoscalingRunnerSetSpec{
			MinRunners: &minRunners,
			ScheduledOverrides: []v1alpha1.ScheduledOverride{
				{
					StartTime: metav1.NewTime(now.Add(-time.Hour)),
					EndTime:   metav1.NewTime(now.Add(time.Hour)),
					RecurrenceRule: v1alpha1.RecurrenceRule{
						Frequency: "Hourly", // not a valid frequency
					},
				},
			},
		}

		_, _, _, err := resolveMinRunners(spec, now)
		assert.Error(t, err)
	})
}

func TestScheduledOverrideRequeueAfter(t *testing.T) {
	minRunners := 1
	now := time.Now()

	t.Run("no overrides means no requeue", func(t *testing.T) {
		spec := &v1alpha1.AutoscalingRunnerSetSpec{MinRunners: &minRunners}

		d, err := scheduledOverrideRequeueAfter(spec, now)
		require.NoError(t, err)
		assert.Zero(t, d)
	})

	t.Run("requeues at the next transition", func(t *testing.T) {
		overrideMinRunners := 5
		spec := &v1alpha1.AutoscalingRunnerSetSpec{
			MinRunners: &minRunners,
			ScheduledOverrides: []v1alpha1.ScheduledOverride{
				{
					StartTime:  metav1.NewTime(now.Add(time.Hour)),
					EndTime:    metav1.NewTime(now.Add(2 * time.Hour)),
					MinRunners: &overrideMinRunners,
				},
			},
		}

		d, err := scheduledOverrideRequeueAfter(spec, now)
		require.NoError(t, err)
		assert.InDelta(t, time.Hour.Seconds(), d.Seconds(), 2)
	})

	t.Run("caps the requeue for far-future transitions", func(t *testing.T) {
		overrideMinRunners := 5
		spec := &v1alpha1.AutoscalingRunnerSetSpec{
			MinRunners: &minRunners,
			ScheduledOverrides: []v1alpha1.ScheduledOverride{
				{
					StartTime:  metav1.NewTime(now.Add(365 * 24 * time.Hour)),
					EndTime:    metav1.NewTime(now.Add(366 * 24 * time.Hour)),
					MinRunners: &overrideMinRunners,
				},
			},
		}

		d, err := scheduledOverrideRequeueAfter(spec, now)
		require.NoError(t, err)
		assert.Equal(t, maxScheduledOverrideRequeueAfter, d)
	})
}
