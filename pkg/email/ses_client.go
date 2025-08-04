package email

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type SESClient struct {
	client *ses.Client
	config *Config
	logger Logger
	stats  *sesStats
}

type sesStats struct {
	sent int64
}

// NewSESClient creates a new AWS SES email client
func NewSESClient(emailConfig *Config, logger Logger) (*SESClient, error) {
	if emailConfig.SESAccessKey == "" || emailConfig.SESSecretKey == "" {
		return nil, NewError("create_ses_client", "ses", ErrProviderNotConfigured)
	}

	// Create AWS config with custom credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(emailConfig.SESRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			emailConfig.SESAccessKey,
			emailConfig.SESSecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, NewError("create_ses_config", "ses", err)
	}

	client := &SESClient{
		client: ses.NewFromConfig(cfg),
		config: emailConfig,
		logger: logger,
		stats:  &sesStats{},
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping SES service: %w", err)
	}

	return client, nil
}

func (s *SESClient) ping(ctx context.Context) error {
	input := &ses.GetSendStatisticsInput{}
	_, err := s.client.GetSendStatistics(ctx, input)
	return err
}

func (s *SESClient) Send(ctx context.Context, message *Message) error {
	if err := s.validateMessage(message); err != nil {
		return err
	}

	input := s.buildSESInput(message)

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}

		_, err := s.client.SendEmail(ctx, input)
		if err == nil {
			s.stats.sent++
			s.logger.Debug("Email sent successfully via SES",
				"to", message.To,
				"subject", message.Subject,
				"attempt", attempt+1,
			)
			return nil
		}

		lastErr = err
		s.logger.Debug("Email send attempt failed",
			"attempt", attempt+1,
			"error", err.Error(),
		)
	}

	return NewError("send", "ses", lastErr)
}

func (s *SESClient) SendBulk(ctx context.Context, messages []*Message) error {
	// SES doesn't have native bulk send, so we send individually
	// In production, you might want to implement batching logic
	for _, message := range messages {
		if err := s.Send(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (s *SESClient) SendTemplate(ctx context.Context, templateMessage *TemplateMessage) error {
	if err := s.validateTemplateMessage(templateMessage); err != nil {
		return err
	}

	input := &ses.SendTemplatedEmailInput{
		Source:       aws.String(s.getFromAddress(templateMessage.From)),
		Template:     aws.String(templateMessage.TemplateID),
		TemplateData: aws.String(s.encodeTemplateData(templateMessage.TemplateData)),
		Destination: &types.Destination{
			ToAddresses: templateMessage.To,
		},
	}

	if len(templateMessage.CC) > 0 {
		input.Destination.CcAddresses = templateMessage.CC
	}
	if len(templateMessage.BCC) > 0 {
		input.Destination.BccAddresses = templateMessage.BCC
	}
	if templateMessage.ReplyTo != "" {
		input.ReplyToAddresses = []string{templateMessage.ReplyTo}
	}

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}

		_, err := s.client.SendTemplatedEmail(ctx, input)
		if err == nil {
			s.stats.sent++
			s.logger.Debug("Template email sent successfully via SES",
				"to", templateMessage.To,
				"template_id", templateMessage.TemplateID,
				"attempt", attempt+1,
			)
			return nil
		}

		lastErr = err
		s.logger.Debug("Template email send attempt failed",
			"attempt", attempt+1,
			"error", err.Error(),
		)
	}

	return NewError("send_template", "ses", lastErr)
}

func (s *SESClient) SendBulkTemplate(ctx context.Context, templateMessages []*TemplateMessage) error {
	// SES has bulk template send capability, but for simplicity, we'll send individually
	for _, message := range templateMessages {
		if err := s.SendTemplate(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (s *SESClient) ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func (s *SESClient) GetStats(ctx context.Context) (Stats, error) {
	// Get SES statistics
	input := &ses.GetSendStatisticsInput{}
	result, err := s.client.GetSendStatistics(ctx, input)
	if err != nil {
		return Stats{}, NewError("get_stats", "ses", err)
	}

	var totalSent, totalBounced, totalComplaints int64
	for _, point := range result.SendDataPoints {
		totalSent += point.DeliveryAttempts
		totalBounced += point.Bounces
		totalComplaints += point.Complaints
	}

	return Stats{
		Sent:       totalSent,
		Delivered:  totalSent - totalBounced,
		Bounced:    totalBounced,
		Complained: totalComplaints,
		Provider:   "ses",
		Metadata: map[string]string{
			"region": s.config.SESRegion,
		},
	}, nil
}

func (s *SESClient) Close() error {
	// SES client doesn't need explicit closing
	s.logger.Info("SES client closed")
	return nil
}

// Helper methods

func (s *SESClient) validateMessage(message *Message) error {
	if len(message.To) == 0 {
		return ErrMissingRecipients
	}
	if message.Subject == "" {
		return ErrMissingSubject
	}
	if message.Text == "" && message.HTML == "" {
		return ErrMissingContent
	}

	// Validate email addresses
	from := s.getFromAddress(message.From)
	if err := s.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := s.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (s *SESClient) validateTemplateMessage(message *TemplateMessage) error {
	if len(message.To) == 0 {
		return ErrMissingRecipients
	}
	if message.TemplateID == "" {
		return fmt.Errorf("template ID is required")
	}

	// Validate email addresses
	from := s.getFromAddress(message.From)
	if err := s.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := s.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (s *SESClient) getFromAddress(from string) string {
	if from == "" {
		return s.config.DefaultFrom
	}
	return from
}

func (s *SESClient) buildSESInput(message *Message) *ses.SendEmailInput {
	input := &ses.SendEmailInput{
		Source: aws.String(s.getFromAddress(message.From)),
		Destination: &types.Destination{
			ToAddresses: message.To,
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(message.Subject),
			},
		},
	}

	// Set body
	body := &types.Body{}
	if message.Text != "" {
		body.Text = &types.Content{
			Data: aws.String(message.Text),
		}
	}
	if message.HTML != "" {
		body.Html = &types.Content{
			Data: aws.String(message.HTML),
		}
	}
	input.Message.Body = body

	// Set optional fields
	if len(message.CC) > 0 {
		input.Destination.CcAddresses = message.CC
	}
	if len(message.BCC) > 0 {
		input.Destination.BccAddresses = message.BCC
	}
	if message.ReplyTo != "" {
		input.ReplyToAddresses = []string{message.ReplyTo}
	}

	return input
}

func (s *SESClient) encodeTemplateData(data map[string]interface{}) string {
	// Simple JSON encoding for template data
	// In production, you'd use proper JSON marshaling
	result := "{"
	first := true
	for key, value := range data {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%v"`, key, value)
		first = false
	}
	result += "}"
	return result
}
