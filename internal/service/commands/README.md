# `internal/service/commands`

Command dispatcher that transforms WhatsApp text commands into structured Google Sheets writes and optional analytics summaries.

## Key Types
- `Service`: implements the `Dispatcher` interface.
- `Dispatcher` interface:
  - `HandleCommand(ctx, cmd, sender) (string, error)` — main entry point used by the WhatsApp service.
  - `SaveEggsRecord`, `SaveFeedRecord`, `SaveMortalityRecord`, `SaveSaleRecord`, `SaveExpenseRecord` — individual persistence hooks (exposed for future reuse/testing).
- `ReportingAdapter`: thin interface satisfied by the reporting service for weekly trend blurbs.

## Supported Commands
| Command | Example | Sheet Range |
|---------|---------|-------------|
| `/eggs 120 cracked 3` | `Eggs!A:C` (`date, quantity, notes`). |
| `/feed 6.5 1200` | `Feed!A:C` (`date, feedKg, population`). |
| `/mortality 3 heat stress` | `Mortality!A:C` (`date, qty, reason`). |
| `/sales 10 250000 250000 CoopMarket` | `Sales!A:E`. |
| `/expenses 75000 vaccines` | `Expenses!A:C`. |

## Flow
1. `HandleCommand` normalizes timestamps (`time.Now().UTC()`), logs the attempt, and branches on `CommandType`.
2. Builders such as `buildEggRecord` parse args into strongly typed structs, validating numeric inputs along the way.
3. Records are written to Google Sheets via `repo.Repository.WriteRow`.
4. Optional analytics (egg summary, feed efficiency, mortality rate) are fetched through `ReportingAdapter` and appended to the response message.
5. Responses return human-readable confirmations for the WhatsApp service to relay.

## Error Handling
- `ErrInvalidArguments`: returned when the command payload cannot be parsed.
- `ErrUnsupportedCommand`: returned when the command does not match a known type.

## Extending Commands
1. Add a new `CommandType` in `internal/domain/models/commands.go`.
2. Teach `ParseCommand` about the command keyword.
3. Add a range constant + `buildXRecord` + `SaveXRecord` here.
4. Update the WhatsApp `commandReplies` map to instruct workers on the syntax.
