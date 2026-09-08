package deliveryaudit

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	adminutils "dpdp-backend/internal/admin/utils"
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
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "correlation_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("correlation_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("customer_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("customer_status_created_at"),
		},
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetName("message_id"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, models)

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

func (r *DeliveryAuditRepository) Search(ctx context.Context, customerID int, params dto.ListParams) ([]dto.Audit, error) {
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

	audits := make([]dto.Audit, 0, len(documents))
	for _, doc := range documents {
		audits = append(audits, fromDocument(doc))
	}

	return audits, nil
}

func (r *DeliveryAuditRepository) Count(ctx context.Context, customerID int, params dto.ListParams) (int64, error) {
	return r.collection.CountDocuments(ctx, searchFilter(customerID, params))
}

func (r *DeliveryAuditRepository) FindByCorrelationID(ctx context.Context, customerID int, correlationID string) (dto.Audit, error) {
	filter := bson.D{
		{Key: "correlation_id", Value: correlationID},
		{Key: "customer_id", Value: customerID},
	}

	var doc document
	if err := r.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		return dto.Audit{}, err
	}

	return fromDocument(doc), nil
}

func searchFilter(customerID int, params dto.ListParams) bson.D {
	filter := bson.D{{Key: "customer_id", Value: customerID}}

	if params.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: params.Status})
	}

	if params.Failure != "" {
		filter = append(filter, bson.E{Key: "failure.type", Value: params.Failure})
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
		}})
	}

	return filter
}

func filterByCorrelation(correlationID string) bson.D {
	return bson.D{{Key: "correlation_id", Value: correlationID}}
}

func fromDocument(doc document) dto.Audit {
	audit := dto.Audit{
		CorrelationID: doc.CorrelationID,
		MessageID:     doc.MessageID,
		CustomerID:    doc.CustomerID,
		ConfigID:      doc.ConfigID,
		From:          doc.From,
		SenderDomain:  doc.SenderDomain,
		Recipients:    make([]dto.Recipient, 0, len(doc.Recipients)),
		Status:        doc.Status,
		DKIM: dto.DKIM{
			Domain:   doc.DKIM.Domain,
			Selector: doc.DKIM.Selector,
			Signed:   doc.DKIM.Signed,
		},
		Attempts:  make([]dto.Attempt, 0, len(doc.Attempts)),
		Size:      doc.Size,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}

	for _, recipient := range doc.Recipients {
		audit.Recipients = append(audit.Recipients, dto.Recipient{
			Email:    recipient.Email,
			Domain:   recipient.Domain,
			Status:   recipient.Status,
			SMTPCode: recipient.SMTPCode,
			Error:    recipient.Error,
		})
	}

	for _, attempt := range doc.Attempts {
		audit.Attempts = append(audit.Attempts, dto.Attempt{
			Number:     attempt.Number,
			StartedAt:  attempt.StartedAt,
			FinishedAt: attempt.FinishedAt,
			MXHost:     attempt.MXHost,
			TLS:        attempt.TLS,
			SMTPCode:   attempt.SMTPCode,
			Error:      attempt.Error,
		})
	}

	if doc.Failure != nil {
		audit.Failure = &dto.Failure{
			Type:     doc.Failure.Type,
			Reason:   doc.Failure.Reason,
			SMTPCode: doc.Failure.SMTPCode,
		}
	}

	return audit
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
