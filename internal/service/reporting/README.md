# `internal/service/reporting`

Analytics helper that reads Google Sheets ranges to produce human-friendly KPIs.

## Public API
- `NewService(repository, logger)`: constructor.
- `GenerateDailyReport(ctx, date) (string, error)`: builds a WhatsApp-ready summary covering eggs, feed, mortality, sales, expenses, and profit with day-over-day deltas. Also embeds the weekly rollup.
- `GenerateWeeklyReport(ctx, date) (string, error)`: aggregates totals for the ISO week containing `date` (Monday → provided day).
- `CalculateEggsSummary`, `CalculateMortalityRate`, `CalculateFeedEfficiency`: lightweight blurbs used immediately after command ingestion.

## Implementation Notes
- **Ranges**: uses the same constants as the command dispatcher (`Eggs!A:C`, `Feed!A:C`, etc.) to avoid drift between ingest + analytics.
- **Helpers**: `aggregate*` functions compute daily vs previous day snapshots; `sum*Between` aids weekly reporting.
- **Formatting**: `formatInt`, `formatFloat`, `formatDelta` helpers keep WhatsApp messages clean with thousand separators and emoji labels.
- **Population estimation**: `estimatePopulation` walks feed records backwards to extract the latest non-zero population.

## Future Hooks
- Scheduler inputs: `GenerateDailyReport` is intentionally pure (only dependencies are repository + logger) so it can be triggered from cron, Cloud Tasks, or manual CLI.
- PDF/dashboard: placeholders exist at the end of the report builder for attaching richer analytics once ready.
