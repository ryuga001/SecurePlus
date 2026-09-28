package emailanalytics

import (
	"context"
	"time"

	dto "dpdp-backend/internal/audit/dto/emailanalytics"
	repo "dpdp-backend/internal/audit/repositories/emailanalytics"
)

const (
	dayBuckets   = 14
	weekBuckets  = 8
	monthBuckets = 6
	topLimit     = 5
)

type Schedule struct {
	Period      string
	Bucket      string
	Buckets     int
	TrendWindow dto.Window
	TopWindow   dto.Window
}

type EmailAnalyticsService struct {
	repo *repo.EmailAnalyticsRepository
}

func NewEmailAnalyticsService(repository *repo.EmailAnalyticsRepository) *EmailAnalyticsService {
	return &EmailAnalyticsService{repo: repository}
}

func (s *EmailAnalyticsService) Analytics(ctx context.Context, customerID int, period string) (dto.Analytics, error) {
	schedule := Plan(period, time.Now())

	facets, err := s.repo.Aggregate(ctx, customerID, dto.AggregateParams{
		TrendFrom: schedule.TrendWindow.From,
		TopFrom:   schedule.TopWindow.From,
		Unit:      schedule.Bucket,
		TopLimit:  topLimit,
	})
	if err != nil {
		return dto.Analytics{}, err
	}

	return dto.Analytics{
		Period:      schedule.Period,
		Bucket:      schedule.Bucket,
		TrendWindow: schedule.TrendWindow,
		TopWindow:   schedule.TopWindow,
		Summary:     facets.Summary,
		Trend:       ZeroFill(facets.Trend, schedule),
		TopUsers:    facets.TopUsers,
		TopPolicies: facets.TopPolicies,
		TopRules:    facets.TopRules,
	}, nil
}

func Plan(period string, now time.Time) Schedule {
	now = now.UTC()

	switch period {
	case dto.PeriodDay:
		return Schedule{
			Period:      dto.PeriodDay,
			Bucket:      dto.PeriodDay,
			Buckets:     dayBuckets,
			TrendWindow: dto.Window{From: startOfDay(now).AddDate(0, 0, -(dayBuckets - 1)), To: now},
			TopWindow:   dto.Window{From: now.Add(-24 * time.Hour), To: now},
		}
	case dto.PeriodMonth:
		return Schedule{
			Period:      dto.PeriodMonth,
			Bucket:      dto.PeriodMonth,
			Buckets:     monthBuckets,
			TrendWindow: dto.Window{From: startOfMonth(now).AddDate(0, -(monthBuckets - 1), 0), To: now},
			TopWindow:   dto.Window{From: now.AddDate(0, 0, -30), To: now},
		}
	default:
		return Schedule{
			Period:      dto.PeriodWeek,
			Bucket:      dto.PeriodWeek,
			Buckets:     weekBuckets,
			TrendWindow: dto.Window{From: startOfWeek(now).AddDate(0, 0, -7*(weekBuckets-1)), To: now},
			TopWindow:   dto.Window{From: now.AddDate(0, 0, -7), To: now},
		}
	}
}

func ZeroFill(points []dto.TrendPoint, schedule Schedule) []dto.TrendPoint {
	found := make(map[time.Time]dto.TrendPoint, len(points))
	for _, point := range points {
		found[point.BucketStart.UTC()] = point
	}

	filled := make([]dto.TrendPoint, 0, schedule.Buckets)

	for index := range schedule.Buckets {
		start := step(schedule.TrendWindow.From, schedule.Bucket, index)

		point, ok := found[start]
		if !ok {
			point = dto.TrendPoint{}
		}

		point.BucketStart = start
		filled = append(filled, point)
	}

	return filled
}

func step(start time.Time, bucket string, index int) time.Time {
	switch bucket {
	case dto.PeriodDay:
		return start.AddDate(0, 0, index)
	case dto.PeriodMonth:
		return start.AddDate(0, index, 0)
	default:
		return start.AddDate(0, 0, 7*index)
	}
}

func startOfDay(moment time.Time) time.Time {
	return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)
}

func startOfWeek(moment time.Time) time.Time {
	day := startOfDay(moment)
	offset := (int(day.Weekday()) + 6) % 7

	return day.AddDate(0, 0, -offset)
}

func startOfMonth(moment time.Time) time.Time {
	return time.Date(moment.Year(), moment.Month(), 1, 0, 0, 0, 0, time.UTC)
}
