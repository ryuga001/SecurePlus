package emailanalytics

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	dto "dpdp-backend/internal/audit/dto/emailanalytics"
	"dpdp-backend/internal/audit/utils"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

const startOfWeekMonday = "monday"

type EmailAnalyticsRepository struct {
	collection *mongo.Collection
}

func NewEmailAnalyticsRepository(client *mongo.Client, database string) *EmailAnalyticsRepository {
	return &EmailAnalyticsRepository{
		collection: client.Database(database).Collection(utils.EmailIncidentCollection),
	}
}

type trendRow struct {
	BucketStart time.Time `bson:"_id"`
	Total       int64     `bson:"total"`
	Blocked     int64     `bson:"blocked"`
}

type summaryRow struct {
	Total   int64 `bson:"total"`
	Blocked int64 `bson:"blocked"`
}

type userRow struct {
	Email   string `bson:"_id"`
	Total   int64  `bson:"total"`
	Blocked int64  `bson:"blocked"`
}

type policyKey struct {
	PolicyID   int    `bson:"policy_id"`
	PolicyName string `bson:"policy_name"`
}

type policyRow struct {
	Key   policyKey `bson:"_id"`
	Count int64     `bson:"count"`
}

type ruleKey struct {
	RuleID   int    `bson:"rule_id"`
	RuleName string `bson:"rule_name"`
	RuleType string `bson:"rule_type"`
}

type ruleRow struct {
	Key   ruleKey `bson:"_id"`
	Count int64   `bson:"count"`
}

type facetResult struct {
	Trend       []trendRow   `bson:"trend"`
	Summary     []summaryRow `bson:"summary"`
	TopUsers    []userRow    `bson:"top_users"`
	TopPolicies []policyRow  `bson:"top_policies"`
	TopRules    []ruleRow    `bson:"top_rules"`
}

func (r *EmailAnalyticsRepository) Aggregate(ctx context.Context, customerID int, params dto.AggregateParams) (dto.Facets, error) {
	cursor, err := r.collection.Aggregate(ctx, pipeline(customerID, params))
	if err != nil {
		return dto.Facets{}, err
	}

	defer cursor.Close(ctx)

	var results []facetResult
	if err := cursor.All(ctx, &results); err != nil {
		return dto.Facets{}, err
	}

	if len(results) == 0 {
		return dto.Facets{}, nil
	}

	return toFacets(results[0]), nil
}

func pipeline(customerID int, params dto.AggregateParams) mongo.Pipeline {
	return mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "customer_id", Value: customerID},
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: params.TrendFrom}}},
		}}},
		bson.D{{Key: "$facet", Value: bson.D{
			{Key: "trend", Value: trendStages(params)},
			{Key: "summary", Value: summaryStages(params)},
			{Key: "top_users", Value: userStages(params)},
			{Key: "top_policies", Value: matchStages(params, "policy")},
			{Key: "top_rules", Value: matchStages(params, "rule")},
		}}},
	}
}

func trendStages(params dto.AggregateParams) bson.A {
	truncate := bson.D{
		{Key: "date", Value: "$created_at"},
		{Key: "unit", Value: params.Unit},
	}

	if params.Unit == dto.PeriodWeek {
		truncate = append(truncate, bson.E{Key: "startOfWeek", Value: startOfWeekMonday})
	}

	return bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateTrunc", Value: truncate}}},
			{Key: "total", Value: counter()},
			{Key: "blocked", Value: blockedCounter()},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func summaryStages(params dto.AggregateParams) bson.A {
	return bson.A{
		since(params.TopFrom),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: counter()},
			{Key: "blocked", Value: blockedCounter()},
		}}},
	}
}

func userStages(params dto.AggregateParams) bson.A {
	return bson.A{
		since(params.TopFrom),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$from"},
			{Key: "total", Value: counter()},
			{Key: "blocked", Value: blockedCounter()},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}}},
		bson.D{{Key: "$limit", Value: params.TopLimit}},
	}
}

func matchStages(params dto.AggregateParams, kind string) bson.A {
	identity := bson.D{
		{Key: kind + "_id", Value: "$matches." + kind + "_id"},
		{Key: kind + "_name", Value: "$matches." + kind + "_name"},
	}

	regrouped := bson.D{
		{Key: kind + "_id", Value: "$_id." + kind + "_id"},
		{Key: kind + "_name", Value: "$_id." + kind + "_name"},
	}

	if kind == "rule" {
		identity = append(identity, bson.E{Key: "rule_type", Value: "$matches.rule_type"})
		regrouped = append(regrouped, bson.E{Key: "rule_type", Value: "$_id.rule_type"})
	}

	deduped := append(bson.D{{Key: "correlation_id", Value: "$correlation_id"}}, identity...)

	return bson.A{
		since(params.TopFrom),
		bson.D{{Key: "$unwind", Value: "$matches"}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: deduped}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: regrouped},
			{Key: "count", Value: counter()},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "count", Value: -1},
			{Key: "_id." + kind + "_name", Value: 1},
		}}},
		bson.D{{Key: "$limit", Value: params.TopLimit}},
	}
}

func since(moment time.Time) bson.D {
	return bson.D{{Key: "$match", Value: bson.D{
		{Key: "created_at", Value: bson.D{{Key: "$gte", Value: moment}}},
	}}}
}

func counter() bson.D {
	return bson.D{{Key: "$sum", Value: 1}}
}

func blockedCounter() bson.D {
	return bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$eq", Value: bson.A{"$effective_action", deliveryutils.ActionBlock}}},
		1,
		0,
	}}}}}
}

func toFacets(result facetResult) dto.Facets {
	facets := dto.Facets{
		Trend:       make([]dto.TrendPoint, 0, len(result.Trend)),
		TopUsers:    make([]dto.UserRanking, 0, len(result.TopUsers)),
		TopPolicies: make([]dto.PolicyRanking, 0, len(result.TopPolicies)),
		TopRules:    make([]dto.RuleRanking, 0, len(result.TopRules)),
	}

	for _, row := range result.Trend {
		facets.Trend = append(facets.Trend, dto.TrendPoint{
			BucketStart: row.BucketStart.UTC(),
			Total:       row.Total,
			Blocked:     row.Blocked,
			Flagged:     row.Total - row.Blocked,
		})
	}

	if len(result.Summary) > 0 {
		row := result.Summary[0]
		facets.Summary = dto.Summary{Total: row.Total, Blocked: row.Blocked, Flagged: row.Total - row.Blocked}
	}

	for _, row := range result.TopUsers {
		facets.TopUsers = append(facets.TopUsers, dto.UserRanking{
			Email:   row.Email,
			Total:   row.Total,
			Blocked: row.Blocked,
			Flagged: row.Total - row.Blocked,
		})
	}

	for _, row := range result.TopPolicies {
		facets.TopPolicies = append(facets.TopPolicies, dto.PolicyRanking{
			PolicyID:   row.Key.PolicyID,
			PolicyName: row.Key.PolicyName,
			Count:      row.Count,
		})
	}

	for _, row := range result.TopRules {
		facets.TopRules = append(facets.TopRules, dto.RuleRanking{
			RuleID:   row.Key.RuleID,
			RuleName: row.Key.RuleName,
			RuleType: row.Key.RuleType,
			Count:    row.Count,
		})
	}

	return facets
}
