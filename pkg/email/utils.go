package email

import (
	"context"
	"fmt"
	"time"
)

// SendWelcomeEmail sends a welcome email using template
func SendWelcomeEmail(ctx context.Context, client Client, to, name string) error {
	message := &TemplateMessage{
		To:         []string{to},
		TemplateID: "welcome_template",
		TemplateData: map[string]interface{}{
			"name": name,
			"date": time.Now().Format("2006-01-02"),
		},
	}
	return client.SendTemplate(ctx, message)
}

// SendPasswordResetEmail sends a password reset email
func SendPasswordResetEmail(ctx context.Context, client Client, to, resetToken string) error {
	message := &TemplateMessage{
		To:         []string{to},
		TemplateID: "password_reset_template",
		TemplateData: map[string]interface{}{
			"reset_token": resetToken,
			"expires_in":  "24 hours",
		},
	}
	return client.SendTemplate(ctx, message)
}

// SendVerificationEmail sends an email verification email
func SendVerificationEmail(ctx context.Context, client Client, to, verificationCode string) error {
	message := &TemplateMessage{
		To:         []string{to},
		TemplateID: "email_verification_template",
		TemplateData: map[string]interface{}{
			"verification_code": verificationCode,
			"expires_in":        "15 minutes",
		},
	}
	return client.SendTemplate(ctx, message)
}

// SendNotificationEmail sends a simple notification email
func SendNotificationEmail(ctx context.Context, client Client, to, subject, text, html string) error {
	message := &Message{
		To:      []string{to},
		Subject: subject,
		Text:    text,
		HTML:    html,
	}
	return client.Send(ctx, message)
}

// BatchSendEmails sends emails in batches to avoid rate limiting
func BatchSendEmails(ctx context.Context, client Client, messages []*Message, batchSize int, delay time.Duration) error {
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		if err := client.SendBulk(ctx, batch); err != nil {
			return fmt.Errorf("failed to send batch %d-%d: %w", i, end-1, err)
		}

		// Delay between batches to respect rate limits
		if delay > 0 && end < len(messages) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue
			}
		}
	}
	return nil
}

// CreateHTMLEmail creates an HTML email message
func CreateHTMLEmail(from, to, subject, htmlContent string) *Message {
	return &Message{
		From:    from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlContent,
	}
}

// CreateTextEmail creates a plain text email message
func CreateTextEmail(from, to, subject, textContent string) *Message {
	return &Message{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    textContent,
	}
}

// CreateMultipartEmail creates an email with both text and HTML content
func CreateMultipartEmail(from, to, subject, textContent, htmlContent string) *Message {
	return &Message{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    textContent,
		HTML:    htmlContent,
	}
}

// AddAttachment adds an attachment to an email message
func AddAttachment(message *Message, filename string, content []byte, contentType string) {
	if message.Attachments == nil {
		message.Attachments = make([]*Attachment, 0)
	}

	attachment := &Attachment{
		Filename:    filename,
		Content:     content,
		ContentType: contentType,
	}

	message.Attachments = append(message.Attachments, attachment)
}

// AddInlineAttachment adds an inline attachment (for embedding images in HTML)
func AddInlineAttachment(message *Message, filename, contentID string, content []byte, contentType string) {
	if message.Attachments == nil {
		message.Attachments = make([]*Attachment, 0)
	}

	attachment := &Attachment{
		Filename:    filename,
		Content:     content,
		ContentType: contentType,
		Inline:      true,
		ContentID:   contentID,
	}

	message.Attachments = append(message.Attachments, attachment)
}
