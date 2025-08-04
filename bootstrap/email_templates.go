package bootstrap

import (
	"context"
	"embed"
	"fmt"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/log"
	"path/filepath"
	"strings"
)

//go:embed templates/email/*.html
var emailTemplates embed.FS

// EmailTemplateRepository interface for template operations
type EmailTemplateRepository interface {
	FindByCodeAndLocale(ctx context.Context, code domain.EmailCode, locale string, option *domain.FindOneOption) (*domain.EmailTemplate, error)
	Create(ctx context.Context, template *domain.EmailTemplate) error
}

// EmailTemplateConfig holds configuration for email templates
type EmailTemplateConfig struct {
	AppName      string
	AppURL       string
	SupportEmail string
	Provider     string
}

// DefaultEmailTemplate represents a default email template
type DefaultEmailTemplate struct {
	Code        domain.EmailCode
	Name        string
	Subject     string
	ContentFile string
	Description string
	Locale      string
}

// GetDefaultEmailTemplates returns the list of default email templates
func GetDefaultEmailTemplates() []DefaultEmailTemplate {
	return []DefaultEmailTemplate{
		{
			Code:        domain.EmailCodeWelcome,
			Name:        "Welcome Email",
			Subject:     "Welcome to {{.app_name}}! 🎉",
			ContentFile: "welcome.html",
			Description: "Welcome email sent to new users after successful registration",
			Locale:      "en",
		},
		{
			Code:        domain.EmailCodeVerification,
			Name:        "Email Verification",
			Subject:     "Verify your email address - {{.app_name}}",
			ContentFile: "verification.html",
			Description: "Email verification sent to users to verify their email address",
			Locale:      "en",
		},
		{
			Code:        domain.EmailCodePasswordReset,
			Name:        "Password Reset",
			Subject:     "Reset your password - {{.app_name}}",
			ContentFile: "password_reset.html",
			Description: "Password reset email sent to users who request password reset",
			Locale:      "en",
		},
	}
}

// InitializeEmailTemplates initializes default email templates if they don't exist in the database
func InitializeEmailTemplates(
	ctx context.Context,
	templateRepo EmailTemplateRepository,
	config EmailTemplateConfig,
	logger log.Logger,
) error {
	logger.Info("Initializing email templates...")

	defaultTemplates := GetDefaultEmailTemplates()

	for _, defaultTemplate := range defaultTemplates {
		// Check if template already exists
		existing, err := templateRepo.FindByCodeAndLocale(ctx, defaultTemplate.Code, defaultTemplate.Locale, nil)
		if err != nil && !isNotFoundError(err) {
			logger.Error("Failed to check existing template",
				log.String("code", string(defaultTemplate.Code)),
				log.String("locale", defaultTemplate.Locale),
				log.Error(err),
			)
			continue
		}

		if existing != nil {
			logger.Warn("Email template already exists, skipping",
				log.String("code", string(defaultTemplate.Code)),
				log.String("locale", defaultTemplate.Locale),
				log.String("name", defaultTemplate.Name),
			)
			continue
		}

		// Load template content
		content, err := loadTemplateContent(defaultTemplate.ContentFile)
		if err != nil {
			logger.Error("Failed to load template content",
				log.String("file", defaultTemplate.ContentFile),
				log.Error(err),
			)
			continue
		}

		// Create new template
		template := &domain.EmailTemplate{
			Code:        defaultTemplate.Code,
			Name:        defaultTemplate.Name,
			Subject:     defaultTemplate.Subject,
			Content:     content,
			Description: defaultTemplate.Description,
			Locale:      defaultTemplate.Locale,
			IsActive:    true,
		}

		if err := templateRepo.Create(ctx, template); err != nil {
			logger.Error("Failed to create email template",
				log.String("code", string(defaultTemplate.Code)),
				log.String("name", defaultTemplate.Name),
				log.Error(err),
			)
			continue
		}

		logger.Info("Created email template",
			log.String("code", string(defaultTemplate.Code)),
			log.String("name", defaultTemplate.Name),
			log.String("locale", defaultTemplate.Locale),
		)
	}

	logger.Info("Email templates initialization completed")
	return nil
}

// loadTemplateContent loads email template content from embedded files
func loadTemplateContent(filename string) (string, error) {
	filePath := filepath.Join("templates", "email", filename)

	content, err := emailTemplates.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	return string(content), nil
}

// isNotFoundError checks if the error is a "not found" error
// This should be adapted based on your error handling implementation
func isNotFoundError(err error) bool {
	// Adapt this based on your domain error types
	if err == domain.ErrRecordNotFound {
		return true
	}

	// Check for domain DetailedError
	if de, ok := err.(*domain.DetailedError); ok {
		return de.IDField == "NOT_FOUND" || strings.Contains(strings.ToLower(de.ErrorField), "not found")
	}

	return false
}

// EmailTemplateSeeder provides methods for seeding email templates
type EmailTemplateSeeder struct {
	templateRepo EmailTemplateRepository
	config       EmailTemplateConfig
	logger       log.Logger
}

// NewEmailTemplateSeeder creates a new email template seeder
func NewEmailTemplateSeeder(
	templateRepo EmailTemplateRepository,
	config EmailTemplateConfig,
	logger log.Logger,
) *EmailTemplateSeeder {
	return &EmailTemplateSeeder{
		templateRepo: templateRepo,
		config:       config,
		logger:       logger,
	}
}

// Seed initializes all default email templates
func (s *EmailTemplateSeeder) Seed(ctx context.Context) error {
	return InitializeEmailTemplates(ctx, s.templateRepo, s.config, s.logger)
}

// SeedTemplate initializes a specific email template
func (s *EmailTemplateSeeder) SeedTemplate(ctx context.Context, code domain.EmailCode, locale string) error {
	defaultTemplates := GetDefaultEmailTemplates()

	for _, defaultTemplate := range defaultTemplates {
		if defaultTemplate.Code != code || defaultTemplate.Locale != locale {
			continue
		}

		// Check if template already exists
		existing, err := s.templateRepo.FindByCodeAndLocale(ctx, code, locale, nil)
		if err != nil && !isNotFoundError(err) {
			return fmt.Errorf("failed to check existing template: %w", err)
		}

		if existing != nil {
			s.logger.Warn("Email template already exists",
				log.String("code", string(code)),
				log.String("locale", locale),
			)
			return nil
		}

		// Load template content
		content, err := loadTemplateContent(defaultTemplate.ContentFile)
		if err != nil {
			return fmt.Errorf("failed to load template content: %w", err)
		}

		// Create new template
		template := &domain.EmailTemplate{
			Code:        defaultTemplate.Code,
			Name:        defaultTemplate.Name,
			Subject:     defaultTemplate.Subject,
			Content:     content,
			Description: defaultTemplate.Description,
			Locale:      defaultTemplate.Locale,
			IsActive:    true,
		}

		if err := s.templateRepo.Create(ctx, template); err != nil {
			return fmt.Errorf("failed to create email template: %w", err)
		}

		s.logger.Info("Created email template",
			log.String("code", string(code)),
			log.String("locale", locale),
		)
		return nil
	}

	return fmt.Errorf("default template not found for code %s and locale %s", code, locale)
}

// GetTemplatePreview returns a preview of a template with sample data
func GetTemplatePreview(templateType domain.EmailCode) map[string]interface{} {
	baseData := map[string]interface{}{
		"app_name":     "YourApp",
		"app_url":      "https://yourapp.com",
		"user_name":    "John Doe",
		"user_email":   "john.doe@example.com",
		"current_year": "2024",
	}

	switch templateType {
	case domain.EmailCodeWelcome:
		return baseData

	case domain.EmailCodeVerification:
		baseData["verification_code"] = "123456"
		baseData["verification_url"] = "https://yourapp.com/verify?token=abc123"
		baseData["expires_in"] = "15 minutes"
		return baseData

	case domain.EmailCodePasswordReset:
		baseData["reset_url"] = "https://yourapp.com/reset-password?token=def456"
		baseData["request_time"] = "2024-01-01 10:30:00 UTC"
		baseData["ip_address"] = "192.168.1.1"
		baseData["expires_in"] = "24 hours"
		return baseData

	default:
		return baseData
	}
}
