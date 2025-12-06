package models

import "time"

// EggRecord captures daily egg production metrics.
type EggRecord struct {
	Date     time.Time
	Band1    int
	Band2    int
	Band3    int
	Quantity int // Total
	Notes    string
}

// FeedRecord captures daily feed usage.
type FeedRecord struct {
	Date       time.Time
	FeedKg     float64
	Population int
}

// MortalityRecord captures mortality incidents.
type MortalityRecord struct {
	Date     time.Time
	Quantity int
	Reason   string
}

// SaleRecord captures sales transactions.
type SaleRecord struct {
	Date         time.Time
	Client       string
	Quantity     int
	PricePerUnit float64
	Paid         float64
}

// ExpenseRecord captures operating expenses.
type ExpenseRecord struct {
	Date      time.Time
	Category  string
	Quantity  float64
	UnitPrice float64
	Amount    float64 // Total amount (Quantity * UnitPrice)
	Notes     string
}

// EggReceptionRecord captures eggs received by the seller.
type EggReceptionRecord struct {
	Date      time.Time
	Quantity  int
	UnitPrice float64
}
