package email

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridClient implements Client interface using SendGrid
type SendGridClient struct {
	client *sendgrid.Client
	config *Config
	logger Logger
	stats  *sendGridStats
}

type sendGridStats struct {
	sent int64
}

// NewSendGridClient creates a new SendGrid email client
func NewSendGridClient(config *Config, logger Logger) (*SendGridClient, error) {
	if config.SendGridAPIKey == "" {
		return nil, NewError("create_sendgrid_client", "sendgrid", ErrProviderNotConfigured)
	}

	client := &SendGridClient{
		client: sendgrid.NewSendClient(config.SendGridAPIKey),
		config: config,
		logger: logger,
		stats:  &sendGridStats{},
	}

	return client, nil
}

func (sg *SendGridClient) Send(ctx context.Context, message *Message) error {
	if err := sg.validateMessage(message); err != nil {
		return err
	}

	sgMessage := sg.buildSendGridMessage(message)

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= sg.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sg.config.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}

		response, err := sg.client.Send(sgMessage)
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			sg.stats.sent++
			sg.logger.Debug("Email sent successfully via SendGrid",
				"to", message.To,
				"subject", message.Subject,
				"status_code", response.StatusCode,
				"attempt", attempt+1,
			)
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("SendGrid API returned status %d: %s", response.StatusCode, response.Body)
		}

		sg.logger.Debug("Email send attempt failed",
			"attempt", attempt+1,
			"error", lastErr.Error(),
		)
	}

	return NewError("send", "sendgrid", lastErr)
}

func (sg *SendGridClient) SendBulk(ctx context.Context, messages []*Message) error {
	// SendGrid supports bulk sending, but for simplicity, we'll send individually
	// In production, you'd want to use SendGrid's batch functionality
	for _, message := range messages {
		if err := sg.Send(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (sg *SendGridClient) SendTemplate(ctx context.Context, templateMessage *TemplateMessage) error {
	if err := sg.validateTemplateMessage(templateMessage); err != nil {
		return err
	}

	sgMessage := sg.buildSendGridTemplateMessage(templateMessage)

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= sg.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sg.config.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}

		response, err := sg.client.Send(sgMessage)
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			sg.stats.sent++
			sg.logger.Debug("Template email sent successfully via SendGrid",
				"to", templateMessage.To,
				"template_id", templateMessage.TemplateID,
				"status_code", response.StatusCode,
				"attempt", attempt+1,
			)
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("SendGrid API returned status %d: %s", response.StatusCode, response.Body)
		}

		sg.logger.Debug("Template email send attempt failed",
			"attempt", attempt+1,
			"error", lastErr.Error(),
		)
	}

	return NewError("send_template", "sendgrid", lastErr)
}

func (sg *SendGridClient) SendBulkTemplate(ctx context.Context, templateMessages []*TemplateMessage) error {
	for _, message := range templateMessages {
		if err := sg.SendTemplate(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (sg *SendGridClient) ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func (sg *SendGridClient) GetStats(ctx context.Context) (Stats, error) {
	// SendGrid stats would require additional API calls
	// For now, return basic stats
	return Stats{
		Sent:     sg.stats.sent,
		Provider: "sendgrid",
		Metadata: map[string]string{
			"api_key_prefix": sg.config.SendGridAPIKey[:8] + "...",
		},
	}, nil
}

func (sg *SendGridClient) Close() error {
	sg.logger.Info("SendGrid client closed")
	return nil
}

// Helper methods

func (sg *SendGridClient) validateMessage(message *Message) error {
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
	from := sg.getFromAddress(message.From)
	if err := sg.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := sg.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (sg *SendGridClient) validateTemplateMessage(message *TemplateMessage) error {
	if len(message.To) == 0 {
		return ErrMissingRecipients
	}
	if message.TemplateID == "" {
		return fmt.Errorf("template ID is required")
	}

	// Validate email addresses
	from := sg.getFromAddress(message.From)
	if err := sg.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := sg.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (sg *SendGridClient) getFromAddress(from string) string {
	if from == "" {
		return sg.config.DefaultFrom
	}
	return from
}

func (sg *SendGridClient) buildSendGridMessage(message *Message) *mail.SGMailV3 {
	fromEmail := mail.NewEmail(sg.config.SendGridFromName, sg.getFromAddress(message.From))

	sgMessage := mail.NewV3Mail()
	sgMessage.SetFrom(fromEmail)
	sgMessage.Subject = message.Subject

	// Add recipients
	personalization := mail.NewPersonalization()
	for _, to := range message.To {
		personalization.AddTos(mail.NewEmail("", to))
	}
	for _, cc := range message.CC {
		personalization.AddCCs(mail.NewEmail("", cc))
	}
	for _, bcc := range message.BCC {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}
	sgMessage.AddPersonalizations(personalization)

	// Set content
	if message.Text != "" {
		sgMessage.AddContent(mail.NewContent("text/plain", message.Text))
	}
	if message.HTML != "" {
		sgMessage.AddContent(mail.NewContent("text/html", message.HTML))
	}

	// Add attachments
	for _, attachment := range message.Attachments {
		sgAttachment := mail.NewAttachment()
		sgAttachment.SetFilename(attachment.Filename)
		sgAttachment.SetContent(base64.StdEncoding.EncodeToString(attachment.Content))
		sgAttachment.SetType(attachment.ContentType)
		if attachment.Inline {
			sgAttachment.SetDisposition("inline")
			if attachment.ContentID != "" {
				sgAttachment.SetContentID(attachment.ContentID)
			}
		}
		sgMessage.AddAttachment(sgAttachment)
	}

	// Add reply-to
	if message.ReplyTo != "" {
		sgMessage.SetReplyTo(mail.NewEmail("", message.ReplyTo))
	}

	return sgMessage
}

func (sg *SendGridClient) buildSendGridTemplateMessage(templateMessage *TemplateMessage) *mail.SGMailV3 {
	fromEmail := mail.NewEmail(sg.config.SendGridFromName, sg.getFromAddress(templateMessage.From))

	sgMessage := mail.NewV3Mail()
	sgMessage.SetFrom(fromEmail)
	sgMessage.SetTemplateID(templateMessage.TemplateID)

	// Add recipients and template data
	personalization := mail.NewPersonalization()
	for _, to := range templateMessage.To {
		personalization.AddTos(mail.NewEmail("", to))
	}
	for _, cc := range templateMessage.CC {
		personalization.AddCCs(mail.NewEmail("", cc))
	}
	for _, bcc := range templateMessage.BCC {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	// Add template data
	for key, value := range templateMessage.TemplateData {
		personalization.SetDynamicTemplateData(key, value)
	}

	sgMessage.AddPersonalizations(personalization)

	// Add reply-to
	if templateMessage.ReplyTo != "" {
		sgMessage.SetReplyTo(mail.NewEmail("", templateMessage.ReplyTo))
	}

	return sgMessage
}
