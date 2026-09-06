//go:build integration

package testsupport

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

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

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}

	index, err := strconv.Atoi(os.Getenv("TEST_REDIS_DB"))
	if err != nil || index == 0 {
		t.Fatal("TEST_REDIS_DB must be set to a non-zero logical database, the suite flushes it")
	}

	if index == redisIndex(os.Getenv("REDIS_DB")) {
		t.Fatal("TEST_REDIS_DB must differ from REDIS_DB, the suite flushes it")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := config.Redis{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), DB: index}.Connect(ctx)
	if err != nil {
		t.Fatalf("test redis connection failed: %v", err)
	}

	t.Cleanup(func() { client.Close() })

	return client
}

func Reset(t *testing.T, database *gorm.DB, rdb *redis.Client) {
	t.Helper()

	if err := database.Exec("TRUNCATE email_provider_configurations RESTART IDENTITY CASCADE").Error; err != nil {
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

func redisIndex(raw string) int {
	index, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return index
}
