# `internal/repository/sheets`

Google Sheets persistence adapter that keeps storage simple and observable.

## Interfaces
- `Repository`
  - `WriteRow(ctx, range, values)`: appends a row using `USER_ENTERED` mode.
  - `ReadRange(ctx, range)`: fetches rectangular data for analytics/reporting.

## Implementation
`GoogleSheetRepository` wraps the official `google.golang.org/api/sheets/v4` client.

```go
repo, err := sheets.NewGoogleSheetRepository(ctx, cfg.Sheets, logger)
repo.WriteRow(ctx, "Eggs!A:C", []interface{}{date, qty, notes})
rows, _ := repo.ReadRange(ctx, "Feed!A:C")
```

### Design Highlights
- Uses service-account credentials file + `SpreadsheetsScope` for minimal permissions.
- Adds structured logging (`logger.Debug`) whenever rows are appended.
- Validates `sheetRange` inputs to avoid silent no-ops.

### Adding New Sheets
1. Create the tab in Google Sheets with the desired headers.
2. Define the A1 range constant inside the consuming service.
3. Call `WriteRow`/`ReadRange` with the new range; no repository code changes needed.
