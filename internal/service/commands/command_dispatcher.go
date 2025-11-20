package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/domain/models"
	repo "github.com/mamadbah2/farmer/internal/repository/sheets"
)

// ErrInvalidArguments indicates the command payload could not be parsed.
var ErrInvalidArguments = errors.New("invalid command arguments")

// ErrUnsupportedCommand indicates we do not yet support the requested command.
var ErrUnsupportedCommand = errors.New("unsupported command")

const (
	eggsWriteRange      = "Eggs!A:C"
	feedWriteRange      = "Feed!A:C"
	mortalityWriteRange = "Mortality!A:C"
	salesWriteRange     = "Sales!A:E"
	expenseWriteRange   = "Expenses!A:C"
	dateFormat          = "2006-01-02"
)

// ReportingAdapter defines the reporting functions required by the dispatcher.
type ReportingAdapter interface {
	CalculateEggsSummary(ctx context.Context, start, end time.Time) (string, error)
	CalculateMortalityRate(ctx context.Context, start, end time.Time) (string, error)
	CalculateFeedEfficiency(ctx context.Context, start, end time.Time) (string, error)
}

// Dispatcher executes parsed commands and persists the structured payloads.
type Dispatcher interface {
	HandleCommand(ctx context.Context, cmd models.Command, sender string) (string, error)
	SaveEggsRecord(ctx context.Context, record models.EggRecord) error
	SaveFeedRecord(ctx context.Context, record models.FeedRecord) error
	SaveMortalityRecord(ctx context.Context, record models.MortalityRecord) error
	SaveSaleRecord(ctx context.Context, record models.SaleRecord) error
	SaveExpenseRecord(ctx context.Context, record models.ExpenseRecord) error
}

// Service implements the Dispatcher interface.
type Service struct {
	repo      repo.Repository
	reporting ReportingAdapter
	logger    *zap.Logger
	now       func() time.Time
}

// NewService constructs a command dispatcher.
func NewService(repository repo.Repository, reporting ReportingAdapter, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:      repository,
		reporting: reporting,
		logger:    logger,
		now:       time.Now,
	}
}

// HandleCommand converts the command to its record representation and persists it.
func (s *Service) HandleCommand(ctx context.Context, cmd models.Command, sender string) (string, error) {
	normalizedNow := s.now().UTC()
	startOfWeek := mondayStart(normalizedNow)

	s.logger.Debug("dispatching command", zap.String("command", string(cmd.Type)), zap.String("sender", sender), zap.Any("args", cmd.Args))

	switch cmd.Type {
	case models.CommandEggs:
		record, err := s.buildEggRecord(cmd, normalizedNow)
		if err != nil {
			return "", err
		}
		if err := s.SaveEggsRecord(ctx, record); err != nil {
			return "", err
		}
		summary := s.safeSummary(ctx, func(ctx context.Context) (string, error) {
			if s.reporting == nil {
				return "", nil
			}
			return s.reporting.CalculateEggsSummary(ctx, startOfWeek, normalizedNow)
		})
		message := fmt.Sprintf("Egg record saved for %s with %d eggs.", record.Date.Format(dateFormat), record.Quantity)
		if summary != "" {
			message += "\n" + summary
		}
		return message, nil
	case models.CommandFeed:
		record, err := s.buildFeedRecord(cmd, normalizedNow)
		if err != nil {
			return "", err
		}
		if err := s.SaveFeedRecord(ctx, record); err != nil {
			return "", err
		}
		summary := s.safeSummary(ctx, func(ctx context.Context) (string, error) {
			if s.reporting == nil {
				return "", nil
			}
			return s.reporting.CalculateFeedEfficiency(ctx, startOfWeek, normalizedNow)
		})
		message := fmt.Sprintf("Feed usage saved for %s: %.2f kg.", record.Date.Format(dateFormat), record.FeedKg)
		if record.Population > 0 {
			message += fmt.Sprintf(" Population %d birds.", record.Population)
		}
		if summary != "" {
			message += "\n" + summary
		}
		return message, nil
	case models.CommandMortality:
		record, err := s.buildMortalityRecord(cmd, normalizedNow)
		if err != nil {
			return "", err
		}
		if err := s.SaveMortalityRecord(ctx, record); err != nil {
			return "", err
		}
		summary := s.safeSummary(ctx, func(ctx context.Context) (string, error) {
			if s.reporting == nil {
				return "", nil
			}
			return s.reporting.CalculateMortalityRate(ctx, startOfWeek, normalizedNow)
		})
		message := fmt.Sprintf("Mortality logged for %s: %d birds.", record.Date.Format(dateFormat), record.Quantity)
		if record.Reason != "" {
			message += fmt.Sprintf(" Reason: %s.", record.Reason)
		}
		if summary != "" {
			message += "\n" + summary
		}
		return message, nil
	case models.CommandSales:
		record, err := s.buildSaleRecord(cmd, normalizedNow)
		if err != nil {
			return "", err
		}
		if err := s.SaveSaleRecord(ctx, record); err != nil {
			return "", err
		}
		total := float64(record.Quantity) * record.PricePerUnit
		message := fmt.Sprintf("Sale recorded for %s: %d units @ %.2f (expected %.2f, paid %.2f).", record.Client, record.Quantity, record.PricePerUnit, total, record.Paid)
		return message, nil
	case models.CommandExpenses:
		record, err := s.buildExpenseRecord(cmd, normalizedNow)
		if err != nil {
			return "", err
		}
		if err := s.SaveExpenseRecord(ctx, record); err != nil {
			return "", err
		}
		message := fmt.Sprintf("Expense logged: %s %.2f on %s.", record.Label, record.Amount, record.Date.Format(dateFormat))
		return message, nil
	default:
		return "", ErrUnsupportedCommand
	}
}

// SaveEggsRecord persists an egg record to Google Sheets.
func (s *Service) SaveEggsRecord(ctx context.Context, record models.EggRecord) error {
	values := []interface{}{record.Date.Format(dateFormat), record.Quantity, record.Notes}
	return s.repo.WriteRow(ctx, eggsWriteRange, values)
}

// SaveFeedRecord persists feed consumption data.
func (s *Service) SaveFeedRecord(ctx context.Context, record models.FeedRecord) error {
	values := []interface{}{record.Date.Format(dateFormat), record.FeedKg, record.Population}
	return s.repo.WriteRow(ctx, feedWriteRange, values)
}

// SaveMortalityRecord persists mortality data.
func (s *Service) SaveMortalityRecord(ctx context.Context, record models.MortalityRecord) error {
	values := []interface{}{record.Date.Format(dateFormat), record.Quantity, record.Reason}
	return s.repo.WriteRow(ctx, mortalityWriteRange, values)
}

// SaveSaleRecord persists sales transactions.
func (s *Service) SaveSaleRecord(ctx context.Context, record models.SaleRecord) error {
	values := []interface{}{record.Date.Format(dateFormat), record.Client, record.Quantity, record.PricePerUnit, record.Paid}
	return s.repo.WriteRow(ctx, salesWriteRange, values)
}

// SaveExpenseRecord persists expenses transactions.
func (s *Service) SaveExpenseRecord(ctx context.Context, record models.ExpenseRecord) error {
	values := []interface{}{record.Date.Format(dateFormat), record.Label, record.Amount}
	return s.repo.WriteRow(ctx, expenseWriteRange, values)
}

func (s *Service) buildEggRecord(cmd models.Command, now time.Time) (models.EggRecord, error) {
	if len(cmd.Args) == 0 {
		return models.EggRecord{}, ErrInvalidArguments
	}

	quantity, err := strconv.Atoi(cmd.Args[0])
	if err != nil {
		return models.EggRecord{}, ErrInvalidArguments
	}

	notes := ""
	if len(cmd.Args) > 1 {
		notes = strings.Join(cmd.Args[1:], " ")
	}

	return models.EggRecord{Date: now, Quantity: quantity, Notes: notes}, nil
}

func (s *Service) buildFeedRecord(cmd models.Command, now time.Time) (models.FeedRecord, error) {
	if len(cmd.Args) == 0 {
		return models.FeedRecord{}, ErrInvalidArguments
	}

	feedKg, err := strconv.ParseFloat(cmd.Args[0], 64)
	if err != nil {
		return models.FeedRecord{}, ErrInvalidArguments
	}

	population := 0
	if len(cmd.Args) > 1 {
		pop, err := strconv.Atoi(cmd.Args[1])
		if err == nil {
			population = pop
		}
	}

	return models.FeedRecord{Date: now, FeedKg: feedKg, Population: population}, nil
}

func (s *Service) buildMortalityRecord(cmd models.Command, now time.Time) (models.MortalityRecord, error) {
	if len(cmd.Args) == 0 {
		return models.MortalityRecord{}, ErrInvalidArguments
	}

	quantity, err := strconv.Atoi(cmd.Args[0])
	if err != nil {
		return models.MortalityRecord{}, ErrInvalidArguments
	}

	reason := ""
	if len(cmd.Args) > 1 {
		reason = strings.Join(cmd.Args[1:], " ")
	}

	return models.MortalityRecord{Date: now, Quantity: quantity, Reason: reason}, nil
}

func (s *Service) buildSaleRecord(cmd models.Command, now time.Time) (models.SaleRecord, error) {
	if len(cmd.Args) < 2 {
		return models.SaleRecord{}, ErrInvalidArguments
	}

	quantity, err := strconv.Atoi(cmd.Args[0])
	if err != nil {
		return models.SaleRecord{}, ErrInvalidArguments
	}

	pricePerUnit, err := strconv.ParseFloat(cmd.Args[1], 64)
	if err != nil {
		return models.SaleRecord{}, ErrInvalidArguments
	}

	paid := float64(quantity) * pricePerUnit
	idx := 2
	if len(cmd.Args) > 2 {
		if v, err := strconv.ParseFloat(cmd.Args[2], 64); err == nil {
			paid = v
			idx = 3
		}
	}

	client := "Walk-in"
	if len(cmd.Args) > idx {
		client = strings.Join(cmd.Args[idx:], " ")
	}

	return models.SaleRecord{
		Date:         now,
		Client:       client,
		Quantity:     quantity,
		PricePerUnit: pricePerUnit,
		Paid:         paid,
	}, nil
}

func (s *Service) buildExpenseRecord(cmd models.Command, now time.Time) (models.ExpenseRecord, error) {
	if len(cmd.Args) < 2 {
		return models.ExpenseRecord{}, ErrInvalidArguments
	}

	amount, err := strconv.ParseFloat(cmd.Args[0], 64)
	if err != nil {
		return models.ExpenseRecord{}, ErrInvalidArguments
	}

	label := strings.Join(cmd.Args[1:], " ")
	return models.ExpenseRecord{Date: now, Label: label, Amount: amount}, nil
}

func (s *Service) safeSummary(ctx context.Context, fn func(context.Context) (string, error)) string {
	if fn == nil {
		return ""
	}

	summary, err := fn(ctx)
	if err != nil {
		s.logger.Debug("analytics summary failed", zap.Error(err))
		return ""
	}

	return summary
}

func mondayStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	daysSinceMonday := (weekday + 6) % 7
	start := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}
