package models

import "time"

// DailyReport represents the aggregated daily data to be stored in MongoDB.
type DailyReport struct {
	Date          time.Time `bson:"date" json:"date"`
	EggsCollected int       `bson:"eggs_collected" json:"eggs_collected"`
	Mortality     int       `bson:"mortality" json:"mortality"`
	FeedConsumed  float64   `bson:"feed_consumed" json:"feed_consumed"`
	SalesAmount   float64   `bson:"sales_amount" json:"sales_amount"`
	UnpaidBalance float64   `bson:"unpaid_balance" json:"unpaid_balance"`
	Expenses      float64   `bson:"expenses" json:"expenses"`
	Profit        float64   `bson:"profit" json:"profit"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
}
