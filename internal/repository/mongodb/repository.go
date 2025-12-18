package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mamadbah2/farmer/internal/domain/models"
)

// Repository defines the interface for report storage.
type Repository interface {
	SaveDailyReport(ctx context.Context, report models.DailyReport) error
}

// MongoDBRepository implements the Repository interface for MongoDB.
type MongoDBRepository struct {
	client   *mongo.Client
	dbName   string
	collName string
}

// NewMongoDBRepository creates a new MongoDB repository.
func NewMongoDBRepository(ctx context.Context, uri string, dbName string) (*MongoDBRepository, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &MongoDBRepository{
		client:   client,
		dbName:   dbName,
		collName: "daily_reports",
	}, nil
}

// SaveDailyReport saves a daily report to the database.
func (r *MongoDBRepository) SaveDailyReport(ctx context.Context, report models.DailyReport) error {
	collection := r.client.Database(r.dbName).Collection(r.collName)
	_, err := collection.InsertOne(ctx, report)
	if err != nil {
		return fmt.Errorf("failed to insert daily report: %w", err)
	}
	return nil
}

// Close closes the MongoDB connection.
func (r *MongoDBRepository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}
