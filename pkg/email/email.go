package email

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Provider string

const (
	SES      Provider = "ses"
	SendGrid Provider = "sendgrid"
	Mock     Provider = "mock"
)

var (
	ErrInvalidProvider       = errors.New("invalid email provider")
	ErrInvalidEmail          = errors.New("invalid email address")
	ErrMissingRecipients     = errors.New("no recipients specified")
	ErrMissingSubject        = errors.New("subject is required")
	ErrMissingContent        = errors.New("email content is required")
	ErrSendFailed            = errors.New("failed to send email")
	ErrProviderNotConfigured = errors.New("email provider not properly configured")
)

type Error struct {
	Operation string
	Provider  string
	Err       error
}

func (e *Error) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("email %s operation failed for provider '%s': %v", e.Operation, e.Provider, e.Err)
	}
	return fmt.Sprintf("email %s operation failed: %v", e.Operation, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error with the specified operation, provider, and underlying error
func NewError(operation, provider string, err error) *Error {
	return &Error{
		Operation: operation,
		Provider:  provider,
		Err:       err,
	}
}

// NewOperationError creates a new Error with only operation and underlying error (no provider)
func NewOperationError(operation string, err error) *Error {
	return &Error{
		Operation: operation,
		Err:       err,
	}
}

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Client interface {
	Send(ctx context.Context, message *Message) error
	SendBulk(ctx context.Context, messages []*Message) error
	SendTemplate(ctx context.Context, templateMessage *TemplateMessage) error
	SendBulkTemplate(ctx context.Context, templateMessages []*TemplateMessage) error
	ValidateEmail(email string) error
	GetStats(ctx context.Context) (Stats, error)
	Close() error
}

type Message struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	Subject     string            `json:"subject"`
	Text        string            `json:"text,omitempty"`
	HTML        string            `json:"html,omitempty"`
	Attachments []*Attachment     `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type TemplateMessage struct {
	From         string                 `json:"from"`
	To           []string               `json:"to"`
	CC           []string               `json:"cc,omitempty"`
	BCC          []string               `json:"bcc,omitempty"`
	ReplyTo      string                 `json:"reply_to,omitempty"`
	TemplateID   string                 `json:"template_id"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
	Inline      bool   `json:"inline,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

type Stats struct {
	Sent       int64             `json:"sent"`
	Delivered  int64             `json:"delivered"`
	Bounced    int64             `json:"bounced"`
	Complained int64             `json:"complained"`
	Opened     int64             `json:"opened"`
	Clicked    int64             `json:"clicked"`
	Provider   string            `json:"provider"`
	Metadata   map[string]string `json:"metadata"`
}

type Config struct {
	// Common settings
	Provider    string `json:"provider" yaml:"provider"`
	DefaultFrom string `json:"default_from" yaml:"default_from"`

	// AWS SES settings
	SESRegion           string `json:"ses_region" yaml:"ses_region"`
	SESAccessKey        string `json:"ses_access_key" yaml:"ses_access_key"`
	SESSecretKey        string `json:"ses_secret_key" yaml:"ses_secret_key"`
	SESConfigurationSet string `json:"ses_configuration_set" yaml:"ses_configuration_set"`

	// SendGrid settings
	SendGridAPIKey   string `json:"sendgrid_api_key" yaml:"sendgrid_api_key"`
	SendGridFromName string `json:"sendgrid_from_name" yaml:"sendgrid_from_name"`

	// Rate limiting
	RateLimit       int           `json:"rate_limit" yaml:"rate_limit"`
	RateLimitPeriod time.Duration `json:"rate_limit_period" yaml:"rate_limit_period"`

	// Retry settings
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`

	// Mock settings (for testing)
	MockDelay    time.Duration `json:"mock_delay" yaml:"mock_delay"`
	MockFailRate float64       `json:"mock_fail_rate" yaml:"mock_fail_rate"`
}

type Factory struct {
	logger Logger
}

// NewEmailFactory creates a new email factory
func NewEmailFactory(logger Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateClient creates an email client based on the configuration
func (f *Factory) CreateClient(provider Provider, config *Config) (Client, error) {
	switch provider {
	case SES:
		return f.createSESClient(config)
	case SendGrid:
		return f.createSendGridClient(config)
	case Mock:
		return f.createMockClient(config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}
}

// createSESClient creates an AWS SES email client
func (f *Factory) createSESClient(config *Config) (Client, error) {
	f.setSESDefaults(config)

	client, err := NewSESClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SES client: %w", err)
	}

	f.logger.Info("SES email client created successfully",
		"region", config.SESRegion,
		"default_from", config.DefaultFrom,
		"rate_limit", config.RateLimit,
	)

	return client, nil
}

// createSendGridClient creates a SendGrid email client
func (f *Factory) createSendGridClient(config *Config) (Client, error) {
	f.setSendGridDefaults(config)

	client, err := NewSendGridClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SendGrid client: %w", err)
	}

	f.logger.Info("SendGrid email client created successfully",
		"default_from", config.DefaultFrom,
		"from_name", config.SendGridFromName,
		"rate_limit", config.RateLimit,
	)

	return client, nil
}

// createMockClient creates a mock email client for testing
func (f *Factory) createMockClient(config *Config) (Client, error) {
	f.setMockDefaults(config)

	client := NewMockClient(config, f.logger)

	f.logger.Info("Mock email client created successfully",
		"delay", config.MockDelay,
		"fail_rate", config.MockFailRate,
	)

	return client, nil
}

// setSESDefaults sets default values for SES configuration
func (f *Factory) setSESDefaults(config *Config) {
	if config.SESRegion == "" {
		config.SESRegion = "us-east-1"
	}
	if config.RateLimit == 0 {
		config.RateLimit = 14 // SES default rate limit
	}
	if config.RateLimitPeriod == 0 {
		config.RateLimitPeriod = time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
}

// setSendGridDefaults sets default values for SendGrid configuration
func (f *Factory) setSendGridDefaults(config *Config) {
	if config.RateLimit == 0 {
		config.RateLimit = 100 // SendGrid allows higher rates
	}
	if config.RateLimitPeriod == 0 {
		config.RateLimitPeriod = time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
}

// setMockDefaults sets default values for mock configuration
func (f *Factory) setMockDefaults(config *Config) {
	if config.MockDelay == 0 {
		config.MockDelay = 100 * time.Millisecond
	}
	// MockFailRate defaults to 0 (no failures)
	if config.RateLimit == 0 {
		config.RateLimit = 1000 // High rate for testing
	}
	if config.RateLimitPeriod == 0 {
		config.RateLimitPeriod = time.Second
	}
}

// EmailBuilder provides a fluent interface for building email configurations
type EmailBuilder struct {
	config *Config
	logger Logger
}

// NewEmailBuilder creates a new email builder
func NewEmailBuilder(logger Logger) *EmailBuilder {
	return &EmailBuilder{
		config: &Config{},
		logger: logger,
	}
}

// WithProvider sets the email provider
func (b *EmailBuilder) WithProvider(provider Provider) *EmailBuilder {
	b.config.Provider = string(provider)
	return b
}

// WithDefaultFrom sets the default from address
func (b *EmailBuilder) WithDefaultFrom(from string) *EmailBuilder {
	b.config.DefaultFrom = from
	return b
}

// WithSES configures AWS SES settings
func (b *EmailBuilder) WithSES(region, accessKey, secretKey string) *EmailBuilder {
	b.config.SESRegion = region
	b.config.SESAccessKey = accessKey
	b.config.SESSecretKey = secretKey
	return b
}

// WithSendGrid configures SendGrid settings
func (b *EmailBuilder) WithSendGrid(apiKey, fromName string) *EmailBuilder {
	b.config.SendGridAPIKey = apiKey
	b.config.SendGridFromName = fromName
	return b
}

// WithRateLimit configures rate limiting
func (b *EmailBuilder) WithRateLimit(limit int, period time.Duration) *EmailBuilder {
	b.config.RateLimit = limit
	b.config.RateLimitPeriod = period
	return b
}

// WithRetry configures retry settings
func (b *EmailBuilder) WithRetry(maxRetries int, delay time.Duration) *EmailBuilder {
	b.config.MaxRetries = maxRetries
	b.config.RetryDelay = delay
	return b
}

// WithMock configures mock settings for testing
func (b *EmailBuilder) WithMock(delay time.Duration, failRate float64) *EmailBuilder {
	b.config.MockDelay = delay
	b.config.MockFailRate = failRate
	return b
}

// Build creates the email client
func (b *EmailBuilder) Build(provider Provider) (Client, error) {
	factory := NewEmailFactory(b.logger)
	return factory.CreateClient(provider, b.config)
}

// BuildSES creates an SES email client
func (b *EmailBuilder) BuildSES() (Client, error) {
	return b.Build(SES)
}

// BuildSendGrid creates a SendGrid email client
func (b *EmailBuilder) BuildSendGrid() (Client, error) {
	return b.Build(SendGrid)
}

// BuildMock creates a mock email client
func (b *EmailBuilder) BuildMock() (Client, error) {
	return b.Build(Mock)
}

// GetClientFromConfig creates an email client from configuration
func GetClientFromConfig(config *Config, logger Logger) (Client, error) {
	factory := NewEmailFactory(logger)
	provider := Provider(config.Provider)
	return factory.CreateClient(provider, config)
}
