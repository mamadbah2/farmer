package reporting

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap"

	repo "github.com/mamadbah2/farmer/internal/repository/sheets"
)

const (
	dateLayout         = "2006-01-02"
	eggsDataRange      = "Eggs!A:C"
	feedDataRange      = "Feed!A:C"
	mortalityDataRange = "Mortality!A:C"
)

// Service exposes lightweight analytics for WhatsApp summaries.
type Service struct {
	repo   repo.Repository
	logger *zap.Logger
}

// NewService wires a new reporting service instance.
func NewService(repository repo.Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repository, logger: logger}
}

// CalculateEggsSummary aggregates egg production for a period and returns a formatted string.
func (s *Service) CalculateEggsSummary(ctx context.Context, start, end time.Time) (string, error) {
	rows, err := s.repo.ReadRange(ctx, eggsDataRange)
	if err != nil {
		return "", fmt.Errorf("load eggs range: %w", err)
	}

	var total int
	var entries int

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		dateValue, err := parseDate(row[0])
		if err != nil {
			s.logger.Debug("skip eggs row with invalid date", zap.Any("value", row[0]), zap.Error(err))
			continue
		}
		if dateValue.Before(start) || dateValue.After(end) {
			continue
		}

		qty, err := parseInt(row[1])
		if err != nil {
			s.logger.Debug("skip eggs row with invalid qty", zap.Any("value", row[1]), zap.Error(err))
			continue
		}

		total += qty
		entries++
	}

	if entries == 0 {
		return fmt.Sprintf("Egg summary (%s-%s): no records yet.", start.Format(dateLayout), end.Format(dateLayout)), nil
	}

	return fmt.Sprintf("Egg summary (%s-%s): %d eggs across %d updates.", start.Format(dateLayout), end.Format(dateLayout), total, entries), nil
}

// CalculateMortalityRate produces a simple mortality ratio using the latest population information.
func (s *Service) CalculateMortalityRate(ctx context.Context, start, end time.Time) (string, error) {
	rows, err := s.repo.ReadRange(ctx, mortalityDataRange)
	if err != nil {
		return "", fmt.Errorf("load mortality range: %w", err)
	}

	var totalDeaths int
	var events int

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}

		qty, err := parseInt(row[1])
		if err != nil {
			s.logger.Debug("skip mortality row with invalid qty", zap.Any("value", row[1]), zap.Error(err))
			continue
		}

		totalDeaths += qty
		events++
	}

	if events == 0 {
		return fmt.Sprintf("Mortality (%s-%s): no incidents logged.", start.Format(dateLayout), end.Format(dateLayout)), nil
	}

	population := s.estimatePopulation(ctx, start, end)

	var ratioStatement string
	if population > 0 {
		rate := (float64(totalDeaths) / float64(population)) * 100
		rate = math.Round(rate*100) / 100
		ratioStatement = fmt.Sprintf("Mortality rate %.2f%% based on population %d.", rate, population)
	} else {
		ratioStatement = "Population unknown. Log /feed with population to compute rate."
	}

	return fmt.Sprintf("Mortality (%s-%s): %d deaths across %d reports. %s", start.Format(dateLayout), end.Format(dateLayout), totalDeaths, events, ratioStatement), nil
}

// CalculateFeedEfficiency estimates feed usage per bird for a period.
func (s *Service) CalculateFeedEfficiency(ctx context.Context, start, end time.Time) (string, error) {
	rows, err := s.repo.ReadRange(ctx, feedDataRange)
	if err != nil {
		return "", fmt.Errorf("load feed range: %w", err)
	}

	var totalFeed float64
	var population int
	var entries int

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}

		feedValue, err := parseFloat(row[1])
		if err != nil {
			s.logger.Debug("skip feed row with invalid feedkg", zap.Any("value", row[1]), zap.Error(err))
			continue
		}

		totalFeed += feedValue
		thisPopulation := 0
		if len(row) > 2 {
			if pop, err := parseInt(row[2]); err == nil {
				thisPopulation = pop
			}
		}

		if thisPopulation > 0 {
			population = thisPopulation
		}
		entries++
	}

	if entries == 0 {
		return fmt.Sprintf("Feed (%s-%s): awaiting data.", start.Format(dateLayout), end.Format(dateLayout)), nil
	}

	var efficiencyStatement string
	if population > 0 {
		efficiency := totalFeed / float64(population)
		efficiencyStatement = fmt.Sprintf("Feed per bird %.3f kg.", efficiency)
	} else {
		efficiencyStatement = "Population not provided; feed per bird pending." // TODO: incorporate historical averages.
	}

	return fmt.Sprintf("Feed (%s-%s): %.2f kg consumed across %d entries. %s", start.Format(dateLayout), end.Format(dateLayout), totalFeed, entries, efficiencyStatement), nil
}

// TODO: integrate with scheduled reports & dashboards when cron engine is introduced.

func (s *Service) estimatePopulation(ctx context.Context, start, end time.Time) int {
	rows, err := s.repo.ReadRange(ctx, feedDataRange)
	if err != nil {
		s.logger.Debug("fallback population lookup failed", zap.Error(err))
		return 0
	}

	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if len(row) < 3 {
			continue
		}

		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}

		if dateValue.Before(start) || dateValue.After(end) {
			continue
		}

		pop, err := parseInt(row[2])
		if err != nil || pop <= 0 {
			continue
		}

		return pop
	}

	return 0
}

func parseDate(value interface{}) (time.Time, error) {
	str := fmt.Sprint(value)
	if str == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	if len(str) > 10 {
		str = str[:10]
	}
	return time.Parse(dateLayout, str)
}

func parseInt(value interface{}) (int, error) {
	str := fmt.Sprint(value)
	if str == "" {
		return 0, fmt.Errorf("empty numeric value")
	}
	return strconv.Atoi(str)
}

func parseFloat(value interface{}) (float64, error) {
	str := fmt.Sprint(value)
	if str == "" {
		return 0, fmt.Errorf("empty numeric value")
	}
	return strconv.ParseFloat(str, 64)
}
