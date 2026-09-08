package config

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Mongo struct {
	URI      string
	Database string
}

func loadMongo() Mongo {
	database := os.Getenv("MONGO_DATABASE")
	if database == "" {
		database = "dpdp"
	}

	return Mongo{
		URI:      os.Getenv("MONGO_URI"),
		Database: database,
	}
}

func (m Mongo) Connect(ctx context.Context) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(m.URI))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}
