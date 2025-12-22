package models

import "time"

// StateStockRecord captures physical assets added to inventory.
type StateStockRecord struct {
	Date      time.Time
	ItemName  string
	Quantity  float64
	UnitPrice float64
	Condition string // "etat"
}
