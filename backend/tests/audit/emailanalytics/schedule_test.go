package emailanalytics_test

import (
	"testing"
	"time"

	dto "dpdp-backend/internal/audit/dto/emailanalytics"
	service "dpdp-backend/internal/audit/services/emailanalytics"
)

var reference = time.Date(2026, 9, 29, 14, 37, 12, 0, time.UTC)

func TestPlanDayUsesFourteenDailyBuckets(t *testing.T) {
	schedule := service.Plan(dto.PeriodDay, reference)

	if schedule.Period != dto.PeriodDay || schedule.Bucket != dto.PeriodDay || schedule.Buckets != 14 {
		t.Fatalf("schedule = %+v", schedule)
	}

	midnight := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	if !schedule.TrendWindow.From.Equal(midnight.AddDate(0, 0, -13)) {
		t.Fatalf("trend from = %s", schedule.TrendWindow.From)
	}
	if !schedule.TopWindow.From.Equal(reference.Add(-24 * time.Hour)) {
		t.Fatalf("top from = %s", schedule.TopWindow.From)
	}
}

func TestPlanWeekStartsBucketsOnMonday(t *testing.T) {
	schedule := service.Plan(dto.PeriodWeek, reference)

	if schedule.Buckets != 8 || schedule.Bucket != dto.PeriodWeek {
		t.Fatalf("schedule = %+v", schedule)
	}
	if schedule.TrendWindow.From.Weekday() != time.Monday {
		t.Fatalf("trend from %s is a %s, want Monday", schedule.TrendWindow.From, schedule.TrendWindow.From.Weekday())
	}
	if hour, minute, second := schedule.TrendWindow.From.Clock(); hour != 0 || minute != 0 || second != 0 {
		t.Fatalf("trend from is not midnight: %s", schedule.TrendWindow.From)
	}

	if span := schedule.TrendWindow.To.Sub(schedule.TrendWindow.From); span < 7*7*24*time.Hour {
		t.Fatalf("trend window spans %s, want at least 7 whole weeks", span)
	}
	if !schedule.TopWindow.From.Equal(reference.AddDate(0, 0, -7)) {
		t.Fatalf("top from = %s", schedule.TopWindow.From)
	}
}

func TestPlanMonthStartsBucketsOnTheFirst(t *testing.T) {
	schedule := service.Plan(dto.PeriodMonth, reference)

	if schedule.Buckets != 6 || schedule.Bucket != dto.PeriodMonth {
		t.Fatalf("schedule = %+v", schedule)
	}

	want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !schedule.TrendWindow.From.Equal(want) {
		t.Fatalf("trend from = %s, want %s", schedule.TrendWindow.From, want)
	}
	if !schedule.TopWindow.From.Equal(reference.AddDate(0, 0, -30)) {
		t.Fatalf("top from = %s", schedule.TopWindow.From)
	}
}

func TestPlanFallsBackToWeek(t *testing.T) {
	for _, period := range []string{"", "quarter", "DAY"} {
		if got := service.Plan(period, reference); got.Period != dto.PeriodWeek {
			t.Errorf("period %q resolved to %q, want week", period, got.Period)
		}
	}
}

func TestPlanNormalisesToUTC(t *testing.T) {
	zone := time.FixedZone("IST", int(5.5*3600))
	schedule := service.Plan(dto.PeriodDay, reference.In(zone))

	if schedule.TrendWindow.From.Location() != time.UTC {
		t.Fatalf("trend from is in %s", schedule.TrendWindow.From.Location())
	}
}

func TestZeroFillProducesAFixedLengthSeries(t *testing.T) {
	schedule := service.Plan(dto.PeriodDay, reference)
	third := schedule.TrendWindow.From.AddDate(0, 0, 2)

	filled := service.ZeroFill([]dto.TrendPoint{
		{BucketStart: third, Total: 9, Blocked: 4, Flagged: 5},
	}, schedule)

	if len(filled) != 14 {
		t.Fatalf("length = %d, want 14", len(filled))
	}

	if filled[2].Total != 9 || filled[2].Blocked != 4 || filled[2].Flagged != 5 {
		t.Fatalf("seeded bucket = %+v", filled[2])
	}

	for index, point := range filled {
		if index == 2 {
			continue
		}
		if point.Total != 0 || point.Blocked != 0 || point.Flagged != 0 {
			t.Fatalf("bucket %d should be empty: %+v", index, point)
		}
	}
}

func TestZeroFillKeepsBucketsChronological(t *testing.T) {
	for _, period := range []string{dto.PeriodDay, dto.PeriodWeek, dto.PeriodMonth} {
		schedule := service.Plan(period, reference)
		filled := service.ZeroFill(nil, schedule)

		if len(filled) != schedule.Buckets {
			t.Fatalf("%s: length = %d, want %d", period, len(filled), schedule.Buckets)
		}

		for index := 1; index < len(filled); index++ {
			if !filled[index].BucketStart.After(filled[index-1].BucketStart) {
				t.Fatalf("%s: bucket %d (%s) does not follow %s", period, index, filled[index].BucketStart, filled[index-1].BucketStart)
			}
		}

		if !filled[0].BucketStart.Equal(schedule.TrendWindow.From) {
			t.Fatalf("%s: first bucket = %s, want %s", period, filled[0].BucketStart, schedule.TrendWindow.From)
		}
	}
}

func TestZeroFillDropsBucketsOutsideTheWindow(t *testing.T) {
	schedule := service.Plan(dto.PeriodWeek, reference)

	filled := service.ZeroFill([]dto.TrendPoint{
		{BucketStart: schedule.TrendWindow.From.AddDate(0, 0, -7), Total: 40},
		{BucketStart: schedule.TrendWindow.From, Total: 3},
	}, schedule)

	if filled[0].Total != 3 {
		t.Fatalf("first bucket = %+v", filled[0])
	}

	var total int64
	for _, point := range filled {
		total += point.Total
	}

	if total != 3 {
		t.Fatalf("total across buckets = %d, want 3 (the out-of-window point must be dropped)", total)
	}
}
