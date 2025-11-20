package models

import "strings"

// CommandType enumerates supported worker command categories.
type CommandType string

const (
	CommandEggs      CommandType = "eggs"
	CommandFeed      CommandType = "feed"
	CommandMortality CommandType = "mortality"
	CommandSales     CommandType = "sales"
	CommandExpenses  CommandType = "expenses"
	CommandUnknown   CommandType = "unknown"
)

// Command represents a parsed worker instruction extracted from WhatsApp text.
type Command struct {
	Type CommandType
	Raw  string
	Args []string
}

// ParseCommand derives a Command instance from free-form text messages.
func ParseCommand(message string) Command {
	normalized := strings.TrimSpace(strings.ToLower(message))

	if normalized == "" {
		return Command{Type: CommandUnknown, Raw: message}
	}

	tokens := strings.Fields(normalized)
	cmd := Command{Raw: message}

	if len(tokens) == 0 {
		cmd.Type = CommandUnknown
		return cmd
	}

	head := strings.TrimPrefix(tokens[0], "/")
	switch head {
	case string(CommandEggs):
		cmd.Type = CommandEggs
	case string(CommandFeed):
		cmd.Type = CommandFeed
	case string(CommandMortality):
		cmd.Type = CommandMortality
	case string(CommandSales):
		cmd.Type = CommandSales
	case string(CommandExpenses):
		cmd.Type = CommandExpenses
	default:
		cmd.Type = CommandUnknown
	}

	if len(tokens) > 1 {
		cmd.Args = tokens[1:]
	}

	return cmd
}
