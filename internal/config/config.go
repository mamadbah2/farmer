package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config represents the full application configuration surface.
type Config struct {
	Server   ServerConfig
	WhatsApp WhatsAppConfig
	Sheets   SheetsConfig
}

// ServerConfig holds HTTP server related options.
type ServerConfig struct {
	Port string
}

// WhatsAppConfig contains credentials and options for the Meta WhatsApp Cloud API.
type WhatsAppConfig struct {
	AccessToken   string
	PhoneNumberID string
	VerifyToken   string
	BaseURL       string
	APIVersion    string
}

// SheetsConfig contains configuration required to interact with Google Sheets.
type SheetsConfig struct {
	CredentialsPath string
	SpreadsheetID   string
}

// Load reads environment variables (optionally from the provided file) and
// materializes a Config instance.
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed loading env file %s: %w", envFile, err)
			}
		}
	} else {
		// Ignore the returned error here; missing .env files are acceptable when
		// configuration comes from the environment directly.
		_ = godotenv.Load()
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getenvWithDefault("APP_PORT", "8080"),
		},
		WhatsApp: WhatsAppConfig{
			AccessToken:   os.Getenv("WHATSAPP_TOKEN"),
			PhoneNumberID: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
			VerifyToken:   os.Getenv("META_VERIFY_TOKEN"),
			BaseURL:       getenvWithDefault("WHATSAPP_BASE_URL", "https://graph.facebook.com"),
			APIVersion:    getenvWithDefault("WHATSAPP_API_VERSION", "v20.0"),
		},
		Sheets: SheetsConfig{
			CredentialsPath: os.Getenv("GOOGLE_SHEETS_CREDENTIALS_PATH"),
			SpreadsheetID:   os.Getenv("GOOGLE_SHEET_DATABASE_ID"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures that required configuration fields are populated.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if c.Server.Port == "" {
		return errors.New("APP_PORT must be provided")
	}

	switch {
	case c.WhatsApp.AccessToken == "":
		return errors.New("WHATSAPP_TOKEN must be provided")
	case c.WhatsApp.PhoneNumberID == "":
		return errors.New("WHATSAPP_PHONE_NUMBER_ID must be provided")
	case c.WhatsApp.VerifyToken == "":
		return errors.New("META_VERIFY_TOKEN must be provided")
	}

	if c.WhatsApp.BaseURL == "" {
		return errors.New("WHATSAPP_BASE_URL must not be empty")
	}

	if c.WhatsApp.APIVersion == "" {
		return errors.New("WHATSAPP_API_VERSION must not be empty")
	}

	if c.Sheets.CredentialsPath == "" {
		return errors.New("GOOGLE_SHEETS_CREDENTIALS_PATH must be provided")
	}

	if c.Sheets.SpreadsheetID == "" {
		return errors.New("GOOGLE_SHEET_DATABASE_ID must be provided")
	}

	return nil
}

func getenvWithDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
