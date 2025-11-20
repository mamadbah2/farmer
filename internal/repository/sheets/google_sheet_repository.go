package sheets

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/mamadbah2/farmer/internal/config"
)

// Repository defines the persistence operations supported by the Google Sheets adapter.
type Repository interface {
	WriteRow(ctx context.Context, sheetRange string, values []interface{}) error
	ReadRange(ctx context.Context, sheetRange string) ([][]interface{}, error)
}

// GoogleSheetRepository implements the Repository interface using the official Google Sheets API.
type GoogleSheetRepository struct {
	service       *sheetsapi.Service
	spreadsheetID string
	logger        *zap.Logger
}

// NewGoogleSheetRepository builds a Google Sheets backed repository instance.
func NewGoogleSheetRepository(ctx context.Context, cfg config.SheetsConfig, logger *zap.Logger) (Repository, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	service, err := sheetsapi.NewService(ctx, option.WithCredentialsFile(cfg.CredentialsPath), option.WithScopes(sheetsapi.SpreadsheetsScope))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sheets client: %w", err)
	}

	return &GoogleSheetRepository{
		service:       service,
		spreadsheetID: cfg.SpreadsheetID,
		logger:        logger,
	}, nil
}

// WriteRow appends the provided values to the supplied sheet range.
func (r *GoogleSheetRepository) WriteRow(ctx context.Context, sheetRange string, values []interface{}) error {
	if sheetRange == "" {
		return fmt.Errorf("sheetRange must not be empty")
	}

	payload := &sheetsapi.ValueRange{Values: [][]interface{}{values}}

	call := r.service.Spreadsheets.Values.Append(r.spreadsheetID, sheetRange, payload).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx)

	if _, err := call.Do(); err != nil {
		return fmt.Errorf("append row into range %s: %w", sheetRange, err)
	}

	r.logger.Debug("row appended to sheet", zap.String("range", sheetRange))
	return nil
}

// ReadRange fetches a rectangular data range from the spreadsheet.
func (r *GoogleSheetRepository) ReadRange(ctx context.Context, sheetRange string) ([][]interface{}, error) {
	if sheetRange == "" {
		return nil, fmt.Errorf("sheetRange must not be empty")
	}

	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, sheetRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("read range %s: %w", sheetRange, err)
	}

	return resp.Values, nil
}
