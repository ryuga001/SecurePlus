package emailincident

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/emailincident"
	"dpdp-backend/internal/audit/utils"
)

type recipientDocument struct {
	Email  string `bson:"email"`
	Domain string `bson:"domain"`
}

type withheldDocument struct {
	Email      string `bson:"email"`
	Domain     string `bson:"domain"`
	PolicyID   int    `bson:"policy_id"`
	PolicyName string `bson:"policy_name"`
}

type violationDocument struct {
	Kind        string `bson:"kind"`
	Mode        string `bson:"mode"`
	Value       string `bson:"value"`
	Filename    string `bson:"filename"`
	ContentType string `bson:"content_type"`
	PolicyID    int    `bson:"policy_id"`
	PolicyName  string `bson:"policy_name"`
}

type matchDocument struct {
	PolicyID        int      `bson:"policy_id"`
	PolicyName      string   `bson:"policy_name"`
	RuleID          int      `bson:"rule_id"`
	RuleName        string   `bson:"rule_name"`
	RuleType        string   `bson:"rule_type"`
	ConfiguredValue string   `bson:"configured_value"`
	Occurrences     int      `bson:"occurrences"`
	Locations       []string `bson:"locations"`
}

type document struct {
	CorrelationID         string              `bson:"correlation_id"`
	MessageID             string              `bson:"message_id"`
	CustomerID            int                 `bson:"customer_id"`
	ConfigID              int                 `bson:"config_id"`
	From                  string              `bson:"from"`
	SenderDomain          string              `bson:"sender_domain"`
	Recipients            []recipientDocument `bson:"recipients"`
	EmailUserID           int                 `bson:"email_user_id"`
	EvaluatedPolicyCount  int                 `bson:"evaluated_policy_count"`
	TriggeredPolicyIDs    []int               `bson:"triggered_policy_ids"`
	Decision              string              `bson:"decision"`
	Trigger               string              `bson:"trigger"`
	EffectiveAction       string              `bson:"effective_action"`
	ActionInvoked         string              `bson:"action_invoked"`
	ActionStatus          string              `bson:"action_status"`
	ActionError           string              `bson:"action_error"`
	WithheldRecipients    []withheldDocument  `bson:"withheld_recipients"`
	RestrictionViolations []violationDocument `bson:"restriction_violations"`
	Matches               []matchDocument     `bson:"matches"`
	CreatedAt             time.Time           `bson:"created_at"`
	UpdatedAt             time.Time           `bson:"updated_at"`
}

type EmailIncidentRepository struct {
	collection *mongo.Collection
}

func NewEmailIncidentRepository(client *mongo.Client, database string) *EmailIncidentRepository {
	return &EmailIncidentRepository{
		collection: client.Database(database).Collection(utils.EmailIncidentCollection),
	}
}

func (r *EmailIncidentRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "correlation_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("correlation_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("customer_created_at"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, models)

	return err
}

func (r *EmailIncidentRepository) Upsert(ctx context.Context, record dto.Record) error {
	now := time.Now().UTC()

	doc := document{
		CorrelationID:         record.CorrelationID,
		MessageID:             record.MessageID,
		CustomerID:            record.CustomerID,
		ConfigID:              record.ConfigID,
		From:                  record.From,
		SenderDomain:          record.SenderDomain,
		Recipients:            toRecipientDocuments(record.Recipients),
		EmailUserID:           record.EmailUserID,
		EvaluatedPolicyCount:  record.EvaluatedPolicyCount,
		TriggeredPolicyIDs:    identifiers(record.TriggeredPolicyIDs),
		Decision:              record.Decision,
		Trigger:               record.Trigger,
		EffectiveAction:       record.EffectiveAction,
		ActionInvoked:         record.ActionInvoked,
		ActionStatus:          record.ActionStatus,
		WithheldRecipients:    toWithheldDocuments(record.WithheldRecipients),
		RestrictionViolations: toViolationDocuments(record.RestrictionViolations),
		Matches:               toMatchDocuments(record.Matches),
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	update := bson.D{{Key: "$setOnInsert", Value: doc}}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filterByCorrelation(record.CorrelationID), update, opts)

	return err
}

func (r *EmailIncidentRepository) ApplyAction(ctx context.Context, correlationID string, outcome dto.ActionOutcome) error {
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "action_invoked", Value: outcome.Action},
		{Key: "action_status", Value: outcome.Status},
		{Key: "action_error", Value: outcome.Error},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	_, err := r.collection.UpdateOne(ctx, filterByCorrelation(correlationID), update)

	return err
}

func filterByCorrelation(correlationID string) bson.D {
	return bson.D{{Key: "correlation_id", Value: correlationID}}
}

func identifiers(ids []int) []int {
	if ids == nil {
		return []int{}
	}

	return ids
}

func toRecipientDocuments(recipients []dto.Recipient) []recipientDocument {
	documents := make([]recipientDocument, 0, len(recipients))

	for _, recipient := range recipients {
		documents = append(documents, recipientDocument{Email: recipient.Email, Domain: recipient.Domain})
	}

	return documents
}

func toWithheldDocuments(recipients []dto.WithheldRecipient) []withheldDocument {
	documents := make([]withheldDocument, 0, len(recipients))

	for _, recipient := range recipients {
		documents = append(documents, withheldDocument{
			Email:      recipient.Email,
			Domain:     recipient.Domain,
			PolicyID:   recipient.PolicyID,
			PolicyName: recipient.PolicyName,
		})
	}

	return documents
}

func toViolationDocuments(violations []dto.RestrictionViolation) []violationDocument {
	documents := make([]violationDocument, 0, len(violations))

	for _, violation := range violations {
		documents = append(documents, violationDocument{
			Kind:        violation.Kind,
			Mode:        violation.Mode,
			Value:       violation.Value,
			Filename:    violation.Filename,
			ContentType: violation.ContentType,
			PolicyID:    violation.PolicyID,
			PolicyName:  violation.PolicyName,
		})
	}

	return documents
}

func toMatchDocuments(matches []dto.Match) []matchDocument {
	documents := make([]matchDocument, 0, len(matches))

	for _, match := range matches {
		locations := match.Locations
		if locations == nil {
			locations = []string{}
		}

		documents = append(documents, matchDocument{
			PolicyID:        match.PolicyID,
			PolicyName:      match.PolicyName,
			RuleID:          match.RuleID,
			RuleName:        match.RuleName,
			RuleType:        match.RuleType,
			ConfiguredValue: match.ConfiguredValue,
			Occurrences:     match.Occurrences,
			Locations:       locations,
		})
	}

	return documents
}

func (r *EmailIncidentRepository) Search(ctx context.Context, customerID int, params dto.ListParams) ([]dto.Incident, error) {
	direction := 1
	if params.SortDesc {
		direction = -1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: params.SortBy, Value: direction}}).
		SetSkip(int64(adminutils.Offset(params.Page, params.PageSize))).
		SetLimit(int64(params.PageSize))

	cursor, err := r.collection.Find(ctx, searchFilter(customerID, params), opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	var documents []document
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	incidents := make([]dto.Incident, 0, len(documents))
	for _, doc := range documents {
		incidents = append(incidents, fromDocument(doc))
	}

	return incidents, nil
}

func (r *EmailIncidentRepository) Count(ctx context.Context, customerID int, params dto.ListParams) (int64, error) {
	return r.collection.CountDocuments(ctx, searchFilter(customerID, params))
}

func (r *EmailIncidentRepository) FindByCorrelationID(ctx context.Context, customerID int, correlationID string) (dto.Incident, error) {
	filter := bson.D{
		{Key: "correlation_id", Value: correlationID},
		{Key: "customer_id", Value: customerID},
	}

	var doc document
	if err := r.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		return dto.Incident{}, err
	}

	return fromDocument(doc), nil
}

func searchFilter(customerID int, params dto.ListParams) bson.D {
	filter := bson.D{{Key: "customer_id", Value: customerID}}

	if params.Trigger != "" {
		filter = append(filter, bson.E{Key: "trigger", Value: params.Trigger})
	}

	if params.Action != "" {
		filter = append(filter, bson.E{Key: "effective_action", Value: params.Action})
	}

	if params.Decision != "" {
		filter = append(filter, bson.E{Key: "decision", Value: params.Decision})
	}

	window := bson.D{}

	if !params.From.IsZero() {
		window = append(window, bson.E{Key: "$gte", Value: params.From})
	}

	if !params.To.IsZero() {
		window = append(window, bson.E{Key: "$lte", Value: params.To})
	}

	if len(window) > 0 {
		filter = append(filter, bson.E{Key: "created_at", Value: window})
	}

	if params.Search != "" {
		pattern := bson.Regex{Pattern: regexp.QuoteMeta(params.Search), Options: "i"}

		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "correlation_id", Value: pattern}},
			bson.D{{Key: "message_id", Value: pattern}},
			bson.D{{Key: "from", Value: pattern}},
			bson.D{{Key: "sender_domain", Value: pattern}},
			bson.D{{Key: "recipients.email", Value: pattern}},
			bson.D{{Key: "matches.rule_name", Value: pattern}},
			bson.D{{Key: "restriction_violations.policy_name", Value: pattern}},
		}})
	}

	return filter
}

func fromDocument(doc document) dto.Incident {
	incident := dto.Incident{
		Record: dto.Record{
			CorrelationID:         doc.CorrelationID,
			MessageID:             doc.MessageID,
			CustomerID:            doc.CustomerID,
			ConfigID:              doc.ConfigID,
			From:                  doc.From,
			SenderDomain:          doc.SenderDomain,
			Recipients:            make([]dto.Recipient, 0, len(doc.Recipients)),
			EmailUserID:           doc.EmailUserID,
			EvaluatedPolicyCount:  doc.EvaluatedPolicyCount,
			TriggeredPolicyIDs:    identifiers(doc.TriggeredPolicyIDs),
			Decision:              doc.Decision,
			Trigger:               doc.Trigger,
			EffectiveAction:       doc.EffectiveAction,
			ActionInvoked:         doc.ActionInvoked,
			ActionStatus:          doc.ActionStatus,
			WithheldRecipients:    make([]dto.WithheldRecipient, 0, len(doc.WithheldRecipients)),
			RestrictionViolations: make([]dto.RestrictionViolation, 0, len(doc.RestrictionViolations)),
			Matches:               make([]dto.Match, 0, len(doc.Matches)),
		},
		ActionError: doc.ActionError,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}

	for _, recipient := range doc.Recipients {
		incident.Recipients = append(incident.Recipients, dto.Recipient{
			Email:  recipient.Email,
			Domain: recipient.Domain,
		})
	}

	for _, recipient := range doc.WithheldRecipients {
		incident.WithheldRecipients = append(incident.WithheldRecipients, dto.WithheldRecipient{
			Email:      recipient.Email,
			Domain:     recipient.Domain,
			PolicyID:   recipient.PolicyID,
			PolicyName: recipient.PolicyName,
		})
	}

	for _, violation := range doc.RestrictionViolations {
		incident.RestrictionViolations = append(incident.RestrictionViolations, dto.RestrictionViolation{
			Kind:        violation.Kind,
			Mode:        violation.Mode,
			Value:       violation.Value,
			Filename:    violation.Filename,
			ContentType: violation.ContentType,
			PolicyID:    violation.PolicyID,
			PolicyName:  violation.PolicyName,
		})
	}

	for _, match := range doc.Matches {
		locations := match.Locations
		if locations == nil {
			locations = []string{}
		}

		incident.Matches = append(incident.Matches, dto.Match{
			PolicyID:        match.PolicyID,
			PolicyName:      match.PolicyName,
			RuleID:          match.RuleID,
			RuleName:        match.RuleName,
			RuleType:        match.RuleType,
			ConfiguredValue: match.ConfiguredValue,
			Occurrences:     match.Occurrences,
			Locations:       locations,
		})
	}

	return incident
}
