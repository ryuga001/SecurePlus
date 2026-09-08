package deliveryaudit

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	dto "dpdp-backend/internal/audit/dto/deliveryaudit"
	"dpdp-backend/internal/audit/utils"
)

type recipientDocument struct {
	Email    string `bson:"email"`
	Domain   string `bson:"domain"`
	Status   string `bson:"status"`
	SMTPCode int    `bson:"smtp_code"`
	Error    string `bson:"error"`
}

type attemptDocument struct {
	Number     int       `bson:"number"`
	StartedAt  time.Time `bson:"started_at"`
	FinishedAt time.Time `bson:"finished_at"`
	MXHost     string    `bson:"mx_host"`
	TLS        string    `bson:"tls"`
	SMTPCode   int       `bson:"smtp_code"`
	Error      string    `bson:"error"`
}

type failureDocument struct {
	Type     string `bson:"type"`
	Reason   string `bson:"reason"`
	SMTPCode int    `bson:"smtp_code"`
}

type dkimDocument struct {
	Domain   string `bson:"domain"`
	Selector string `bson:"selector"`
	Signed   bool   `bson:"signed"`
}

type document struct {
	CorrelationID string              `bson:"correlation_id"`
	MessageID     string              `bson:"message_id"`
	CustomerID    int                 `bson:"customer_id"`
	ConfigID      int                 `bson:"config_id"`
	From          string              `bson:"from"`
	SenderDomain  string              `bson:"sender_domain"`
	Recipients    []recipientDocument `bson:"recipients"`
	Status        string              `bson:"status"`
	Failure       *failureDocument    `bson:"failure"`
	DKIM          dkimDocument        `bson:"dkim"`
	Attempts      []attemptDocument   `bson:"attempts"`
	Size          int64               `bson:"size"`
	CreatedAt     time.Time           `bson:"created_at"`
	UpdatedAt     time.Time           `bson:"updated_at"`
}

type DeliveryAuditRepository struct {
	collection *mongo.Collection
}

func NewDeliveryAuditRepository(client *mongo.Client, database string) *DeliveryAuditRepository {
	return &DeliveryAuditRepository{
		collection: client.Database(database).Collection(utils.DeliveryAuditCollection),
	}
}

func (r *DeliveryAuditRepository) EnsureIndexes(ctx context.Context) error {
	model := mongo.IndexModel{
		Keys:    bson.D{{Key: "correlation_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("correlation_id_unique"),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, model)

	return err
}

func (r *DeliveryAuditRepository) Insert(ctx context.Context, record dto.Record) error {
	now := time.Now().UTC()

	doc := document{
		CorrelationID: record.CorrelationID,
		MessageID:     record.MessageID,
		CustomerID:    record.CustomerID,
		ConfigID:      record.ConfigID,
		From:          record.From,
		SenderDomain:  record.SenderDomain,
		Recipients:    toRecipientDocuments(record.Recipients),
		Status:        utils.StatusProcessing,
		Attempts:      []attemptDocument{},
		Size:          record.Size,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := r.collection.InsertOne(ctx, doc)

	return err
}

func (r *DeliveryAuditRepository) PushAttempt(ctx context.Context, correlationID string, attempt dto.Attempt) error {
	update := bson.D{
		{Key: "$push", Value: bson.D{{Key: "attempts", Value: toAttemptDocument(attempt)}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
	}

	_, err := r.collection.UpdateOne(ctx, filterByCorrelation(correlationID), update)

	return err
}

func (r *DeliveryAuditRepository) ApplyResult(ctx context.Context, correlationID string, result dto.Result) error {
	fields := bson.D{
		{Key: "status", Value: result.Status},
		{Key: "recipients", Value: toRecipientDocuments(result.Recipients)},
		{Key: "dkim", Value: dkimDocument{
			Domain:   result.DKIM.Domain,
			Selector: result.DKIM.Selector,
			Signed:   result.DKIM.Signed,
		}},
		{Key: "updated_at", Value: time.Now().UTC()},
	}

	if result.Failure != nil {
		fields = append(fields, bson.E{Key: "failure", Value: failureDocument{
			Type:     result.Failure.Type,
			Reason:   result.Failure.Reason,
			SMTPCode: result.Failure.SMTPCode,
		}})
	} else {
		fields = append(fields, bson.E{Key: "failure", Value: nil})
	}

	update := bson.D{{Key: "$set", Value: fields}}

	_, err := r.collection.UpdateOne(ctx, filterByCorrelation(correlationID), update)

	return err
}

func filterByCorrelation(correlationID string) bson.D {
	return bson.D{{Key: "correlation_id", Value: correlationID}}
}

func toRecipientDocuments(recipients []dto.Recipient) []recipientDocument {
	documents := make([]recipientDocument, 0, len(recipients))

	for _, recipient := range recipients {
		documents = append(documents, recipientDocument{
			Email:    recipient.Email,
			Domain:   recipient.Domain,
			Status:   recipient.Status,
			SMTPCode: recipient.SMTPCode,
			Error:    recipient.Error,
		})
	}

	return documents
}

func toAttemptDocument(attempt dto.Attempt) attemptDocument {
	return attemptDocument{
		Number:     attempt.Number,
		StartedAt:  attempt.StartedAt.UTC(),
		FinishedAt: attempt.FinishedAt.UTC(),
		MXHost:     attempt.MXHost,
		TLS:        attempt.TLS,
		SMTPCode:   attempt.SMTPCode,
		Error:      attempt.Error,
	}
}
