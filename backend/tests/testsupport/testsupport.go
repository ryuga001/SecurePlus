//go:build integration

package testsupport

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"

	"dpdp-backend/internal/audit/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

var (
	postgresOnce sync.Once
	postgresDB   *gorm.DB
	postgresErr  error
)

func Postgres(t *testing.T) *gorm.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set, run make test-db first")
	}

	postgresOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		postgresDB, postgresErr = config.Postgres{URL: url}.Connect(ctx, "test")
	})

	if postgresErr != nil {
		t.Fatalf("test database connection failed: %v", postgresErr)
	}

	return postgresDB
}

func Redis(t *testing.T) *redis.Client {
	t.Helper()

	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}

	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("TEST_REDIS_URL is not a valid redis url: %v", err)
	}

	if options.DB == 0 {
		t.Fatal("TEST_REDIS_URL must select a non-zero logical database, the suite flushes it")
	}

	if options.DB == redisDatabase(os.Getenv("REDIS_URL")) {
		t.Fatal("TEST_REDIS_URL must select a different database than REDIS_URL, the suite flushes it")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := config.Redis{URL: url}.Connect(ctx)
	if err != nil {
		t.Fatalf("test redis connection failed: %v", err)
	}

	t.Cleanup(func() { client.Close() })

	return client
}

func Mongo(t *testing.T) *mongo.Client {
	t.Helper()

	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		t.Skip("TEST_MONGO_URI is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := config.Mongo{URI: uri, Database: MongoDatabase()}.Connect(ctx)
	if err != nil {
		t.Fatalf("test mongo connection failed: %v", err)
	}

	t.Cleanup(func() { client.Disconnect(context.Background()) })

	return client
}

func MongoDatabase() string {
	database := os.Getenv("TEST_MONGO_DATABASE")
	if database == "" {
		return "dpdp_test"
	}

	return database
}

func ResetMongo(t *testing.T, client *mongo.Client) {
	t.Helper()

	err := client.Database(MongoDatabase()).Collection(utils.DeliveryAuditCollection).Drop(context.Background())
	if err != nil {
		t.Fatalf("mongo reset failed: %v", err)
	}
}

func ResetIncidents(t *testing.T, client *mongo.Client) {
	t.Helper()

	err := client.Database(MongoDatabase()).Collection(utils.EmailIncidentCollection).Drop(context.Background())
	if err != nil {
		t.Fatalf("incident reset failed: %v", err)
	}
}

func Reset(t *testing.T, database *gorm.DB, rdb *redis.Client) {
	t.Helper()

	truncate := `TRUNCATE policy_rule_mapping, policy_group_mapping, email_user_group_mapping,
		policies, rules, groups, email_users, email_provider_configurations
		RESTART IDENTITY CASCADE`

	if err := database.Exec(truncate).Error; err != nil {
		t.Fatalf("truncate failed: %v", err)
	}
	if err := database.Exec("DELETE FROM customers WHERE id > 1").Error; err != nil {
		t.Fatalf("customer cleanup failed: %v", err)
	}
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("redis flush failed: %v", err)
	}
}

func Customer(t *testing.T, database *gorm.DB) db.Customer {
	t.Helper()

	customer := db.Customer{OrgName: "test-" + strconv.FormatInt(time.Now().UnixNano(), 36)}

	if err := database.Omit("JWTSecret", "CreatedAt").Create(&customer).Error; err != nil {
		t.Fatalf("customer insert failed: %v", err)
	}

	return customer
}

func Role(t *testing.T, database *gorm.DB, customerID int, privileges ...string) db.Role {
	t.Helper()

	role := db.Role{Name: "Admin", Type: "admin", CustomerID: customerID}

	if err := database.Omit("Privileges").Create(&role).Error; err != nil {
		t.Fatalf("role insert failed: %v", err)
	}

	for _, name := range privileges {
		grant := `INSERT INTO role_privileges (role_id, privilege_id) SELECT ?, id FROM privileges WHERE name = ? ON CONFLICT DO NOTHING`

		if err := database.Exec(grant, role.ID, name).Error; err != nil {
			t.Fatalf("privilege grant failed: %v", err)
		}
	}

	return role
}

func Group(t *testing.T, database *gorm.DB, customerID int, name string) db.Group {
	t.Helper()

	row := db.Group{CustomerID: customerID, Name: name, Type: db.GroupTypeUser}

	if err := database.Create(&row).Error; err != nil {
		t.Fatalf("group insert failed: %v", err)
	}

	return row
}

func EmailUser(t *testing.T, database *gorm.DB, customerID int, email string) db.EmailUser {
	t.Helper()

	row := db.EmailUser{CustomerID: customerID, Email: email, FirstName: "Test", LastName: "User"}

	if err := database.Create(&row).Error; err != nil {
		t.Fatalf("email user insert failed: %v", err)
	}

	return row
}

func Rule(t *testing.T, database *gorm.DB, customerID int, name, kind, value string) db.Rule {
	t.Helper()

	row := db.Rule{CustomerID: customerID, RuleName: name, Type: kind, Value: value}

	if err := database.Create(&row).Error; err != nil {
		t.Fatalf("rule insert failed: %v", err)
	}

	return row
}

func Router(t *testing.T, customerID int, roleID *int) (*gin.Engine, *gin.RouterGroup) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	group := router.Group("/api/v1")

	group.Use(func(c *gin.Context) {
		auth.SetClaims(c, &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "1"},
			CustomerID:       customerID,
			RoleID:           roleID,
		})
		c.Next()
	})

	return router, group
}

func redisDatabase(url string) int {
	options, err := redis.ParseURL(url)
	if err != nil {
		return 0
	}

	return options.DB
}

func Policy(t *testing.T, database *gorm.DB, customerID int, name, action string, domain db.Restriction) db.Policy {
	t.Helper()

	row := db.Policy{
		CustomerID:            customerID,
		PolicyName:            name,
		Type:                  db.PolicyTypeEmail,
		Action:                action,
		Active:                true,
		DomainRestriction:     domain,
		AttachmentRestriction: db.Restriction{Mode: db.RestrictionNone, Values: []string{}},
	}

	if err := database.Omit("CreatedAt", "UpdatedAt").Create(&row).Error; err != nil {
		t.Fatalf("policy insert failed: %v", err)
	}

	return row
}

func BindPolicy(t *testing.T, database *gorm.DB, customerID, policyID, groupID int, ruleIDs ...int) {
	t.Helper()

	mapping := db.PolicyGroupMapping{PolicyID: policyID, GroupID: groupID, CustomerID: customerID}
	if err := database.Create(&mapping).Error; err != nil {
		t.Fatalf("policy group mapping failed: %v", err)
	}

	for _, ruleID := range ruleIDs {
		row := db.PolicyRuleMapping{PolicyID: policyID, RuleID: ruleID, CustomerID: customerID}
		if err := database.Create(&row).Error; err != nil {
			t.Fatalf("policy rule mapping failed: %v", err)
		}
	}
}

func AddMember(t *testing.T, database *gorm.DB, customerID, groupID, emailUserID int) {
	t.Helper()

	row := db.EmailUserGroupMapping{EmailUserID: emailUserID, GroupID: groupID, CustomerID: customerID}

	if err := database.Create(&row).Error; err != nil {
		t.Fatalf("group membership failed: %v", err)
	}
}
