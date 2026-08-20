package actionsgithubcom

import (
	"testing"
	"time"
)

func TestCalculateActiveAndUpcomingRecurringPeriods(t *testing.T) {
	type recurrence struct {
		Start string
		End   string
		Freq  string
		Until string
	}

	type testcase struct {
		now string

		recurrence recurrence

		wantActive   string
		wantUpcoming string
	}

	check := func(t *testing.T, tc testcase) {
		t.Helper()

		_, err := time.Parse(time.RFC3339, "2021-05-08T00:00:00Z")
		if err != nil {
			t.Fatal(err)
		}

		now, err := time.Parse(time.RFC3339, tc.now)
		if err != nil {
			t.Fatal(err)
		}

		active, upcoming, err := parseAndMatchRecurringPeriod(now, tc.recurrence.Start, tc.recurrence.End, tc.recurrence.Freq, tc.recurrence.Until)
		if err != nil {
			t.Fatal(err)
		}

		if active.String() != tc.wantActive {
			t.Errorf("unexpected active: want %q, got %q", tc.wantActive, active)
		}

		if upcoming.String() != tc.wantUpcoming {
			t.Errorf("unexpected upcoming: want %q, got %q", tc.wantUpcoming, upcoming)
		}
	}

	t.Run("onetime override about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
			},

			now: "2021-04-30T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
		})
	})

	t.Run("onetime override started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
			},

			now: "2021-05-01T00:00:00+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("onetime override about to end", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
			},

			now: "2021-05-02T23:59:59+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("onetime override ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
			},

			now: "2021-05-03T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "",
		})
	})

	t.Run("weekly override about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-04-30T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
		})
	})

	t.Run("weekly override started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-01T00:00:00+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
		})
	})

	t.Run("weekly override about to end", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-02T23:59:59+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
		})
	})

	t.Run("weekly override ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-03T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
		})
	})

	t.Run("weekly override reccurrence about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-07T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
		})
	})

	t.Run("weekly override reccurrence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-08T00:00:00+09:00",

			wantActive:   "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
			wantUpcoming: "2021-05-15T00:00:00+09:00-2021-05-17T00:00:00+09:00",
		})
	})

	t.Run("weekly override reccurrence about to end", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-09T23:59:59+09:00",

			wantActive:   "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
			wantUpcoming: "2021-05-15T00:00:00+09:00-2021-05-17T00:00:00+09:00",
		})
	})

	t.Run("weekly override reccurrence ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-10T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "2021-05-15T00:00:00+09:00-2021-05-17T00:00:00+09:00",
		})
	})

	t.Run("weekly override's last reccurrence about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-04-29T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2022-04-30T00:00:00+09:00-2022-05-02T00:00:00+09:00",
		})
	})

	t.Run("weekly override reccurrence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-04-30T00:00:00+09:00",

			wantActive:   "2022-04-30T00:00:00+09:00-2022-05-02T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("weekly override reccurrence about to end", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-01T23:59:59+09:00",

			wantActive:   "2022-04-30T00:00:00+09:00-2022-05-02T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("weekly override reccurrence ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-02T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "",
		})
	})

	t.Run("weekly override repeated forever started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Weekly",
			},

			now: "2021-05-08T00:00:00+09:00",

			wantActive:   "2021-05-08T00:00:00+09:00-2021-05-10T00:00:00+09:00",
			wantUpcoming: "2021-05-15T00:00:00+09:00-2021-05-17T00:00:00+09:00",
		})
	})

	t.Run("monthly override started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-01T00:00:00+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "2021-06-01T00:00:00+09:00-2021-06-03T00:00:00+09:00",
		})
	})

	t.Run("monthly override recurrence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-06-01T00:00:00+09:00",

			wantActive:   "2021-06-01T00:00:00+09:00-2021-06-03T00:00:00+09:00",
			wantUpcoming: "2021-07-01T00:00:00+09:00-2021-07-03T00:00:00+09:00",
		})
	})

	t.Run("monthly override's last reccurence about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-04-30T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
		})
	})

	t.Run("monthly override's last reccurence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-01T00:00:00+09:00",

			wantActive:   "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("monthly override's last reccurence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-01T00:00:01+09:00",

			wantActive:   "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("monthly override's last reccurence ending", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-02T23:59:59+09:00",

			wantActive:   "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("monthly override's last reccurence ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Monthly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2022-05-03T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "",
		})
	})

	t.Run("yearly override started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2022-05-01T00:00:00+09:00",
			},

			now: "2021-05-01T00:00:00+09:00",

			wantActive:   "2021-05-01T00:00:00+09:00-2021-05-03T00:00:00+09:00",
			wantUpcoming: "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
		})
	})

	t.Run("yearly override reccurrence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2023-05-01T00:00:00+09:00",
			},

			now: "2022-05-01T00:00:00+09:00",

			wantActive:   "2022-05-01T00:00:00+09:00-2022-05-03T00:00:00+09:00",
			wantUpcoming: "2023-05-01T00:00:00+09:00-2023-05-03T00:00:00+09:00",
		})
	})

	t.Run("yearly override's last recurrence about to start", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2023-05-01T00:00:00+09:00",
			},

			now: "2023-04-30T23:59:59+09:00",

			wantActive:   "",
			wantUpcoming: "2023-05-01T00:00:00+09:00-2023-05-03T00:00:00+09:00",
		})
	})

	t.Run("yearly override's last recurrence started", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2023-05-01T00:00:00+09:00",
			},

			now: "2023-05-01T00:00:00+09:00",

			wantActive:   "2023-05-01T00:00:00+09:00-2023-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("yearly override's last recurrence ending", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2023-05-01T00:00:00+09:00",
			},

			now: "2023-05-02T23:23:59+09:00",

			wantActive:   "2023-05-01T00:00:00+09:00-2023-05-03T00:00:00+09:00",
			wantUpcoming: "",
		})
	})

	t.Run("yearly override's last recurrence ended", func(t *testing.T) {
		t.Helper()

		check(t, testcase{
			recurrence: recurrence{
				Start: "2021-05-01T00:00:00+09:00",
				End:   "2021-05-03T00:00:00+09:00",
				Freq:  "Yearly",
				Until: "2023-05-01T00:00:00+09:00",
			},

			now: "2023-05-03T00:00:00+09:00",

			wantActive:   "",
			wantUpcoming: "",
		})
	})
}

func parseAndMatchRecurringPeriod(now time.Time, start, end, frequency, until string) (*Period, *Period, error) {
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, nil, err
	}

	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return nil, nil, err
	}

	var untilTime time.Time

	if until != "" {
		ut, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return nil, nil, err
		}

		untilTime = ut
	}

	return MatchSchedule(now, startTime, endTime, RecurrenceRule{Frequency: frequency, UntilTime: untilTime})
}

// TestDurationInvariantIsFixed checks that a Monthly override's maximum allowed
// duration no longer depends on the calendar month it happens to be evaluated in.
// A 30-day override exceeds the fixed 28-day Monthly invariant and must be
// rejected consistently, regardless of whether `now` falls in a 31-day month or
// in February.
func TestDurationInvariantIsFixed(t *testing.T) {
	start, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	end := start.Add(30 * 24 * time.Hour)

	for _, now := range []string{
		"2021-01-15T00:00:00Z", // evaluated during a 31-day month
		"2021-02-15T00:00:00Z", // evaluated during a 28-day month
	} {
		t.Run(now, func(t *testing.T) {
			nowTime, err := time.Parse(time.RFC3339, now)
			if err != nil {
				t.Fatal(err)
			}

			_, _, err = MatchSchedule(nowTime, start, end, RecurrenceRule{Frequency: "Monthly"})
			if err == nil {
				t.Fatal("expected an error: a 30-day override exceeds the fixed 28-day Monthly invariant")
			}
		})
	}
}

// TestUpcomingOccurrenceSurvivesMonthWithoutTheStartDay checks that a Monthly
// override starting on the 31st still finds its next occurrence when evaluated
// from within February, which has no 31st. The naive "one period after now"
// window used to miss it entirely.
func TestUpcomingOccurrenceSurvivesMonthWithoutTheStartDay(t *testing.T) {
	start, err := time.Parse(time.RFC3339, "2021-01-31T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	end := start.Add(time.Hour)
	now, err := time.Parse(time.RFC3339, "2021-02-15T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	_, upcoming, err := MatchSchedule(now, start, end, RecurrenceRule{Frequency: "Monthly"})
	if err != nil {
		t.Fatal(err)
	}

	want := "2021-03-31T00:00:00Z-2021-03-31T01:00:00Z"
	if got := upcoming.String(); got != want {
		t.Errorf("unexpected upcoming: want %q, got %q", want, got)
	}
}

// TestUpcomingOccurrenceSurvivesLeapDay checks that a Yearly override starting on
// Feb 29 still finds its next occurrence (the following leap year) when evaluated
// shortly after March 1 of a non-leap year. The naive "one period after now"
// window used to miss it entirely, since the three intervening non-leap years
// have no Feb 29 at all.
func TestUpcomingOccurrenceSurvivesLeapDay(t *testing.T) {
	start, err := time.Parse(time.RFC3339, "2020-02-29T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	end := start.Add(time.Hour)
	now, err := time.Parse(time.RFC3339, "2021-03-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	_, upcoming, err := MatchSchedule(now, start, end, RecurrenceRule{Frequency: "Yearly"})
	if err != nil {
		t.Fatal(err)
	}

	want := "2024-02-29T00:00:00Z-2024-02-29T01:00:00Z"
	if got := upcoming.String(); got != want {
		t.Errorf("unexpected upcoming: want %q, got %q", want, got)
	}
}

func FuzzMatchSchedule(f *testing.F) {
	start := time.Now()
	end := time.Now()
	now := time.Now()
	f.Fuzz(func(t *testing.T, freq string) {
		// Verify that it never panics
		_, _, _ = MatchSchedule(now, start, end, RecurrenceRule{Frequency: freq})
	})
}
