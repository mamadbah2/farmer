package reporting

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/domain/models"
	"github.com/mamadbah2/farmer/internal/repository/mongodb"
	repo "github.com/mamadbah2/farmer/internal/repository/sheets"
)

const (
	dateLayout         = "2006-01-02"
	eggsDataRange      = "Eggs!A:C"
	feedDataRange      = "Feed!A:C"
	mortalityDataRange = "Mortality!A:D"
	salesDataRange     = "Sales!A:E"
	expensesDataRange  = "Expenses!A:C"
)

// Service exposes lightweight analytics for WhatsApp summaries.
type Service struct {
	repo       repo.Repository
	reportRepo mongodb.Repository
	logger     *zap.Logger
}

// NewService wires a new reporting service instance.
func NewService(repository repo.Repository, reportRepo mongodb.Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repository, reportRepo: reportRepo, logger: logger}
}

// GenerateDailyReport aggregates key metrics for the provided date and formats a WhatsApp-ready message.
func (s *Service) GenerateDailyReport(ctx context.Context, reportDate time.Time) (string, error) {
	referenceDate := truncateToDay(reportDate)
	previousDate := referenceDate.AddDate(0, 0, -1)

	eggRows, err := s.repo.ReadRange(ctx, eggsDataRange)
	if err != nil {
		return "", fmt.Errorf("load eggs data: %w", err)
	}
	feedRows, err := s.repo.ReadRange(ctx, feedDataRange)
	if err != nil {
		return "", fmt.Errorf("load feed data: %w", err)
	}
	mortalityRows, err := s.repo.ReadRange(ctx, mortalityDataRange)
	if err != nil {
		return "", fmt.Errorf("load mortality data: %w", err)
	}
	salesRows, err := s.repo.ReadRange(ctx, salesDataRange)
	if err != nil {
		return "", fmt.Errorf("load sales data: %w", err)
	}
	expenseRows, err := s.repo.ReadRange(ctx, expensesDataRange)
	if err != nil {
		return "", fmt.Errorf("load expenses data: %w", err)
	}

	eggsToday, eggsPrev := aggregateEggs(eggRows, referenceDate, previousDate)
	feedToday, feedPrev := aggregateFeed(feedRows, referenceDate, previousDate)
	mortalityToday, mortalityPrev := aggregateMortality(mortalityRows, referenceDate, previousDate)
	salesToday, salesPrev := aggregateSales(salesRows, referenceDate, previousDate)
	expensesToday, expensesPrev := aggregateExpenses(expenseRows, referenceDate, previousDate)
	profitToday := salesToday.Paid - expensesToday.Total
	profitPrev := salesPrev.Paid - expensesPrev.Total

	// Save to MongoDB
	if s.reportRepo != nil {
		report := models.DailyReport{
			Date:          referenceDate,
			EggsCollected: eggsToday,
			Mortality:     mortalityToday,
			FeedConsumed:  feedToday.TotalKg,
			SalesAmount:   salesToday.Paid,
			UnpaidBalance: salesToday.Unpaid,
			Expenses:      expensesToday.Total,
			Profit:        profitToday,
			CreatedAt:     time.Now(),
		}
		if err := s.reportRepo.SaveDailyReport(ctx, report); err != nil {
			s.logger.Error("failed to save daily report to mongodb", zap.Error(err))
		}
	}

	weeklySummary, err := s.GenerateWeeklyReport(ctx, referenceDate)
	if err != nil {
		s.logger.Debug("weekly summary failed", zap.Error(err))
		weeklySummary = "Weekly summary will be available once data sync completes."
	}

	var builder strings.Builder
	writeDivider(&builder)
	fmt.Fprintf(&builder, "🐔 DAILY REPORT – %s\n", referenceDate.Format("02/01/2006"))
	fmt.Fprintf(&builder, "🥚 Eggs collected: %s (%s vs yesterday)\n", formatInt(eggsToday), formatDelta(eggsToday-eggsPrev))
	fmt.Fprintf(&builder, "🪦 Mortality: %s birds (%s vs yesterday)\n", formatInt(mortalityToday), formatDelta(mortalityToday-mortalityPrev))
	feedLine := formatFeedLine(feedToday, feedPrev)
	fmt.Fprintf(&builder, "%s\n", feedLine)
	fmt.Fprintf(&builder, "💸 Sales: %s GNF (%s vs yesterday)\n", formatFloat(salesToday.Paid, 0), formatCurrencyDelta(salesToday.Paid-salesPrev.Paid))
	fmt.Fprintf(&builder, "📉 Unpaid balance: %s GNF\n", formatFloat(salesToday.Unpaid, 0))
	fmt.Fprintf(&builder, "🧾 Expenses: %s GNF (%s vs yesterday)\n", formatFloat(expensesToday.Total, 0), formatCurrencyDelta(expensesToday.Total-expensesPrev.Total))
	fmt.Fprintf(&builder, "📈 Profit: %s GNF (%s vs yesterday)\n", formatFloat(profitToday, 0), formatCurrencyDelta(profitToday-profitPrev))
	writeDivider(&builder)
	fmt.Fprintf(&builder, "%s\n", weeklySummary)
	writeDivider(&builder)
	fmt.Fprintf(&builder, "Next goals: Increase survival rates and reduce feed cost.\n")
	writeDivider(&builder)
	builder.WriteString("TODO: Attach PDF dashboard and schedule broadcast once BI module ships.\n")

	return builder.String(), nil
}

// GenerateWeeklyReport produces a lightweight overview for the week of the provided date.
func (s *Service) GenerateWeeklyReport(ctx context.Context, referenceDate time.Time) (string, error) {
	weekEnd := truncateToDay(referenceDate)
	weekStart := mondayStart(weekEnd)

	eggRows, err := s.repo.ReadRange(ctx, eggsDataRange)
	if err != nil {
		return "", fmt.Errorf("load eggs data: %w", err)
	}
	feedRows, err := s.repo.ReadRange(ctx, feedDataRange)
	if err != nil {
		return "", fmt.Errorf("load feed data: %w", err)
	}
	mortalityRows, err := s.repo.ReadRange(ctx, mortalityDataRange)
	if err != nil {
		return "", fmt.Errorf("load mortality data: %w", err)
	}
	salesRows, err := s.repo.ReadRange(ctx, salesDataRange)
	if err != nil {
		return "", fmt.Errorf("load sales data: %w", err)
	}
	expenseRows, err := s.repo.ReadRange(ctx, expensesDataRange)
	if err != nil {
		return "", fmt.Errorf("load expenses data: %w", err)
	}

	weeklyEggs := sumEggsBetween(eggRows, weekStart, weekEnd)
	weeklyFeed := sumFeedBetween(feedRows, weekStart, weekEnd)
	weeklyMortality := sumMortalityBetween(mortalityRows, weekStart, weekEnd)
	weeklySales := sumSalesBetween(salesRows, weekStart, weekEnd)
	weeklyExpenses := sumExpensesBetween(expenseRows, weekStart, weekEnd)
	weeklyProfit := weeklySales.Paid - weeklyExpenses.Total

	return fmt.Sprintf("Weekly summary (%s-%s) – 🥚 %s eggs, 🌾 %.2f kg feed, 🪦 %s mortality, 💸 %s GNF sales, 🧾 %s GNF expenses, 📈 %s GNF profit.",
		weekStart.Format("02/01"), weekEnd.Format("02/01"), formatInt(weeklyEggs), weeklyFeed.TotalKg, formatInt(weeklyMortality),
		formatFloat(weeklySales.Paid, 0), formatFloat(weeklyExpenses.Total, 0), formatFloat(weeklyProfit, 0)), nil
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

type feedSnapshot struct {
	TotalKg    float64
	Population int
}

type salesSnapshot struct {
	Paid     float64
	Expected float64
	Unpaid   float64
}

type expenseSnapshot struct {
	Total float64
}

func aggregateEggs(rows [][]interface{}, target, previous time.Time) (int, int) {
	var today, prev int
	targetKey := target.Format(dateLayout)
	prevKey := previous.Format(dateLayout)

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}
		qty, err := parseInt(row[1])
		if err != nil {
			continue
		}
		switch dateValue.Format(dateLayout) {
		case targetKey:
			today += qty
		case prevKey:
			prev += qty
		}
	}

	return today, prev
}

func aggregateMortality(rows [][]interface{}, target, previous time.Time) (int, int) {
	var today, prev int
	targetKey := target.Format(dateLayout)
	prevKey := previous.Format(dateLayout)

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}

		b1, _ := parseInt(row[1])
		b2, _ := parseInt(row[2])
		b3, _ := parseInt(row[3])
		qty := b1 + b2 + b3

		switch dateValue.Format(dateLayout) {
		case targetKey:
			today += qty
		case prevKey:
			prev += qty
		}
	}

	return today, prev
}

func aggregateFeed(rows [][]interface{}, target, previous time.Time) (feedSnapshot, feedSnapshot) {
	var today feedSnapshot
	var prev feedSnapshot
	targetKey := target.Format(dateLayout)
	prevKey := previous.Format(dateLayout)

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}
		feedKg, err := parseFloat(row[1])
		if err != nil {
			continue
		}
		population := 0
		if len(row) > 2 {
			if pop, err := parseInt(row[2]); err == nil && pop > 0 {
				population = pop
			}
		}

		var snapshot *feedSnapshot
		switch dateValue.Format(dateLayout) {
		case targetKey:
			snapshot = &today
		case prevKey:
			snapshot = &prev
		default:
			continue
		}

		snapshot.TotalKg += feedKg
		if population > 0 {
			snapshot.Population = population
		}
	}

	return today, prev
}

func aggregateSales(rows [][]interface{}, target, previous time.Time) (salesSnapshot, salesSnapshot) {
	var today salesSnapshot
	var prev salesSnapshot
	targetKey := target.Format(dateLayout)
	prevKey := previous.Format(dateLayout)

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}
		qty, err := parseInt(row[2])
		if err != nil {
			continue
		}
		price, err := parseFloat(row[3])
		if err != nil {
			continue
		}
		paid := price * float64(qty)
		if len(row) > 4 {
			if v, err := parseFloat(row[4]); err == nil {
				paid = v
			}
		}
		expected := float64(qty) * price
		unpaid := expected - paid
		if unpaid < 0 {
			unpaid = 0
		}

		var snapshot *salesSnapshot
		switch dateValue.Format(dateLayout) {
		case targetKey:
			snapshot = &today
		case prevKey:
			snapshot = &prev
		default:
			continue
		}

		snapshot.Paid += paid
		snapshot.Expected += expected
		snapshot.Unpaid += unpaid
	}

	return today, prev
}

func aggregateExpenses(rows [][]interface{}, target, previous time.Time) (expenseSnapshot, expenseSnapshot) {
	var today expenseSnapshot
	var prev expenseSnapshot
	targetKey := target.Format(dateLayout)
	prevKey := previous.Format(dateLayout)

	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil {
			continue
		}
		amount, err := parseFloat(row[2])
		if err != nil {
			continue
		}

		switch dateValue.Format(dateLayout) {
		case targetKey:
			today.Total += amount
		case prevKey:
			prev.Total += amount
		}
	}

	return today, prev
}

func sumEggsBetween(rows [][]interface{}, start, end time.Time) int {
	var total int
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
			continue
		}
		total += qty
	}
	return total
}

func sumMortalityBetween(rows [][]interface{}, start, end time.Time) int {
	var total int
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}

		b1, _ := parseInt(row[1])
		b2, _ := parseInt(row[2])
		b3, _ := parseInt(row[3])
		total += b1 + b2 + b3
	}
	return total
}

func sumFeedBetween(rows [][]interface{}, start, end time.Time) feedSnapshot {
	var snapshot feedSnapshot
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}
		feedKg, err := parseFloat(row[1])
		if err != nil {
			continue
		}
		snapshot.TotalKg += feedKg
		if len(row) > 2 {
			if pop, err := parseInt(row[2]); err == nil && pop > 0 {
				snapshot.Population = pop
			}
		}
	}
	return snapshot
}

func sumSalesBetween(rows [][]interface{}, start, end time.Time) salesSnapshot {
	var snapshot salesSnapshot
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}
		qty, err := parseInt(row[2])
		if err != nil {
			continue
		}
		price, err := parseFloat(row[3])
		if err != nil {
			continue
		}
		expected := float64(qty) * price
		paid := expected
		if len(row) > 4 {
			if v, err := parseFloat(row[4]); err == nil {
				paid = v
			}
		}
		unpaid := expected - paid
		if unpaid < 0 {
			unpaid = 0
		}
		snapshot.Expected += expected
		snapshot.Paid += paid
		snapshot.Unpaid += unpaid
	}
	return snapshot
}

func sumExpensesBetween(rows [][]interface{}, start, end time.Time) expenseSnapshot {
	var snapshot expenseSnapshot
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		dateValue, err := parseDate(row[0])
		if err != nil || dateValue.Before(start) || dateValue.After(end) {
			continue
		}
		amount, err := parseFloat(row[2])
		if err != nil {
			continue
		}
		snapshot.Total += amount
	}
	return snapshot
}

func formatFeedLine(today feedSnapshot, previous feedSnapshot) string {
	ratioText := "population pending"
	if today.Population > 0 && today.TotalKg > 0 {
		ratio := (today.TotalKg * 1000) / float64(today.Population)
		ratioText = fmt.Sprintf("%.0f g/bird", ratio)
	}
	return fmt.Sprintf("🌾 Feed consumption: %.2f kg (%s, %s vs yesterday)", today.TotalKg, ratioText, formatDeltaFloat(today.TotalKg-previous.TotalKg))
}

func formatDelta(delta int) string {
	if delta > 0 {
		return "+" + formatInt(delta)
	}
	if delta < 0 {
		return "-" + formatInt(-delta)
	}
	return "no change"
}

func formatCurrencyDelta(delta float64) string {
	if delta > 0 {
		return "+" + formatFloat(delta, 0)
	}
	if delta < 0 {
		return "-" + formatFloat(-delta, 0)
	}
	return "no change"
}

func formatDeltaFloat(delta float64) string {
	if delta > 0 {
		return fmt.Sprintf("+%.2f kg", delta)
	}
	if delta < 0 {
		return fmt.Sprintf("%.2f kg", delta)
	}
	return "no change"
}

func formatInt(value int) string {
	return addThousandsSeparator(strconv.Itoa(value))
}

func formatFloat(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	formatted := fmt.Sprintf(format, value)
	if strings.Contains(formatted, ".") {
		parts := strings.Split(formatted, ".")
		return addThousandsSeparator(parts[0]) + "." + strings.TrimRight(parts[1], "0")
	}
	return addThousandsSeparator(formatted)
}

func addThousandsSeparator(input string) string {
	sign := ""
	if strings.HasPrefix(input, "-") {
		sign = "-"
		input = input[1:]
	}
	n := len(input)
	if n <= 3 {
		return sign + input
	}
	var builder strings.Builder
	rem := n % 3
	if rem > 0 {
		builder.WriteString(input[:rem])
		if n > rem {
			builder.WriteString(",")
		}
	}
	for i := rem; i < n; i += 3 {
		builder.WriteString(input[i : i+3])
		if i+3 < n {
			builder.WriteString(",")
		}
	}
	return sign + builder.String()
}

func writeDivider(builder *strings.Builder) {
	builder.WriteString("----------------------------------------------------\n")
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func mondayStart(t time.Time) time.Time {
	s := truncateToDay(t)
	weekday := int(s.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	delta := weekday - 1
	return s.AddDate(0, 0, -delta)
}
