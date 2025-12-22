package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mamadbah2/farmer/internal/domain/models"
)

// Repository defines the interface for report storage.
type Repository interface {
	SaveDailyReport(ctx context.Context, report models.DailyReport) error
	GetDailyReports(ctx context.Context, start, end time.Time) ([]models.DailyReport, error)
	SaveStockItem(ctx context.Context, item models.StateStockRecord) error
}

// MongoDBRepository implements the Repository interface for MongoDB.
type MongoDBRepository struct {
	client        *mongo.Client
	dbName        string
	collName      string
	stockCollName string
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
		client:        client,
		dbName:        dbName,
		collName:      "daily_reports",
		stockCollName: "stock_items",
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

// GetDailyReports retrieves daily reports within a date range.
func (r *MongoDBRepository) GetDailyReports(ctx context.Context, start, end time.Time) ([]models.DailyReport, error) {
	collection := r.client.Database(r.dbName).Collection(r.collName)
	filter := bson.M{
		"date": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find daily reports: %w", err)
	}
	defer cursor.Close(ctx)

	var reports []models.DailyReport
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, fmt.Errorf("failed to decode daily reports: %w", err)
	}

	return reports, nil
}

// SaveStockItem saves a physical stock item to the database.
func (r *MongoDBRepository) SaveStockItem(ctx context.Context, item models.StateStockRecord) error {
	collection := r.client.Database(r.dbName).Collection(r.stockCollName)
	_, err := collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("failed to insert stock item: %w", err)
	}
	return nil
}

// Close closes the MongoDB connection.
func (r *MongoDBRepository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}
