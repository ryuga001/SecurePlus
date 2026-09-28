package emailanalytics

import "time"

const (
	PeriodDay   = "day"
	PeriodWeek  = "week"
	PeriodMonth = "month"
)

type Window struct {
	From time.Time
	To   time.Time
}

type TrendPoint struct {
	BucketStart time.Time
	Total       int64
	Blocked     int64
	Flagged     int64
}

type Summary struct {
	Total   int64
	Blocked int64
	Flagged int64
}

type UserRanking struct {
	Email   string
	Total   int64
	Blocked int64
	Flagged int64
}

type PolicyRanking struct {
	PolicyID   int
	PolicyName string
	Count      int64
}

type RuleRanking struct {
	RuleID   int
	RuleName string
	RuleType string
	Count    int64
}

type AggregateParams struct {
	TrendFrom time.Time
	TopFrom   time.Time
	Unit      string
	TopLimit  int
}

type Facets struct {
	Trend       []TrendPoint
	Summary     Summary
	TopUsers    []UserRanking
	TopPolicies []PolicyRanking
	TopRules    []RuleRanking
}

type Analytics struct {
	Period      string
	Bucket      string
	TrendWindow Window
	TopWindow   Window
	Summary     Summary
	Trend       []TrendPoint
	TopUsers    []UserRanking
	TopPolicies []PolicyRanking
	TopRules    []RuleRanking
}

type AnalyticsQuery struct {
	Period string `form:"period" binding:"omitempty,oneof=day week month"`
}

type WindowResponse struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type TrendPointResponse struct {
	BucketStart time.Time `json:"bucket_start"`
	Total       int64     `json:"total"`
	Blocked     int64     `json:"blocked"`
	Flagged     int64     `json:"flagged"`
}

type TrendResponse struct {
	Bucket string               `json:"bucket"`
	Points []TrendPointResponse `json:"points"`
}

type SummaryResponse struct {
	Total   int64 `json:"total"`
	Blocked int64 `json:"blocked"`
	Flagged int64 `json:"flagged"`
}

type UserRankingResponse struct {
	Email   string `json:"email"`
	Total   int64  `json:"total"`
	Blocked int64  `json:"blocked"`
	Flagged int64  `json:"flagged"`
}

type PolicyRankingResponse struct {
	PolicyID   int    `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Count      int64  `json:"count"`
}

type RuleRankingResponse struct {
	RuleID   int    `json:"rule_id"`
	RuleName string `json:"rule_name"`
	RuleType string `json:"rule_type"`
	Count    int64  `json:"count"`
}

type AnalyticsResponse struct {
	Period      string                  `json:"period"`
	TrendWindow WindowResponse          `json:"trend_window"`
	TopWindow   WindowResponse          `json:"top_window"`
	Summary     SummaryResponse         `json:"summary"`
	Trend       TrendResponse           `json:"trend"`
	TopUsers    []UserRankingResponse   `json:"top_users"`
	TopPolicies []PolicyRankingResponse `json:"top_policies"`
	TopRules    []RuleRankingResponse   `json:"top_rules"`
}
