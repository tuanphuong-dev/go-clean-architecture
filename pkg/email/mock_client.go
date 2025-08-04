package email

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"time"
)

// MockClient implements Client interface for testing purposes
type MockClient struct {
	config     *Config
	logger     Logger
	stats      *mockStats
	sentEmails []MockSentEmail
	mu         sync.RWMutex
}

type mockStats struct {
	sent      int64
	delivered int64
	bounced   int64
	failed    int64
}

type MockSentEmail struct {
	Message     *Message         `json:"message,omitempty"`
	Template    *TemplateMessage `json:"template,omitempty"`
	SentAt      time.Time        `json:"sent_at"`
	DeliveredAt *time.Time       `json:"delivered_at,omitempty"`
	Status      string           `json:"status"` // sent, delivered, bounced, failed
	Error       string           `json:"error,omitempty"`
}

// NewMockClient creates a new mock email client for testing
func NewMockClient(config *Config, logger Logger) *MockClient {
	return &MockClient{
		config:     config,
		logger:     logger,
		stats:      &mockStats{},
		sentEmails: make([]MockSentEmail, 0),
	}
}

func (m *MockClient) Send(ctx context.Context, message *Message) error {
	if err := m.validateMessage(message); err != nil {
		return err
	}

	// Simulate processing delay
	if m.config.MockDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.config.MockDelay):
			// Continue
		}
	}

	// Simulate random failures
	if m.shouldFail() {
		err := fmt.Errorf("mock email send failure (simulated)")
		m.recordSentEmail(&MockSentEmail{
			Message: message,
			SentAt:  time.Now(),
			Status:  "failed",
			Error:   err.Error(),
		})
		m.stats.failed++
		return NewError("send", "mock", err)
	}

	// Record successful send
	sentEmail := MockSentEmail{
		Message: message,
		SentAt:  time.Now(),
		Status:  "sent",
	}

	// Simulate delivery delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		deliveredAt := time.Now()
		sentEmail.DeliveredAt = &deliveredAt
		sentEmail.Status = "delivered"
		m.stats.delivered++
	}()

	m.recordSentEmail(&sentEmail)
	m.stats.sent++

	m.logger.Debug("Mock email sent successfully",
		"to", message.To,
		"subject", message.Subject,
		"delay", m.config.MockDelay,
	)

	return nil
}

func (m *MockClient) SendBulk(ctx context.Context, messages []*Message) error {
	for _, message := range messages {
		if err := m.Send(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockClient) SendTemplate(ctx context.Context, templateMessage *TemplateMessage) error {
	if err := m.validateTemplateMessage(templateMessage); err != nil {
		return err
	}

	// Simulate processing delay
	if m.config.MockDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.config.MockDelay):
			// Continue
		}
	}

	// Simulate random failures
	if m.shouldFail() {
		err := fmt.Errorf("mock template email send failure (simulated)")
		m.recordSentEmail(&MockSentEmail{
			Template: templateMessage,
			SentAt:   time.Now(),
			Status:   "failed",
			Error:    err.Error(),
		})
		m.stats.failed++
		return NewError("send_template", "mock", err)
	}

	// Record successful send
	sentEmail := MockSentEmail{
		Template: templateMessage,
		SentAt:   time.Now(),
		Status:   "sent",
	}

	// Simulate delivery delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		deliveredAt := time.Now()
		sentEmail.DeliveredAt = &deliveredAt
		sentEmail.Status = "delivered"
		m.stats.delivered++
	}()

	m.recordSentEmail(&sentEmail)
	m.stats.sent++

	m.logger.Debug("Mock template email sent successfully",
		"to", templateMessage.To,
		"template_id", templateMessage.TemplateID,
		"delay", m.config.MockDelay,
	)

	return nil
}

func (m *MockClient) SendBulkTemplate(ctx context.Context, templateMessages []*TemplateMessage) error {
	for _, message := range templateMessages {
		if err := m.SendTemplate(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockClient) ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func (m *MockClient) GetStats(ctx context.Context) (Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		Sent:      m.stats.sent,
		Delivered: m.stats.delivered,
		Bounced:   m.stats.bounced,
		Provider:  "mock",
		Metadata: map[string]string{
			"total_emails": fmt.Sprintf("%d", len(m.sentEmails)),
			"fail_rate":    fmt.Sprintf("%.2f", m.config.MockFailRate),
			"delay":        m.config.MockDelay.String(),
		},
	}, nil
}

func (m *MockClient) Close() error {
	m.logger.Info("Mock email client closed",
		"total_sent", m.stats.sent,
		"total_delivered", m.stats.delivered,
		"total_failed", m.stats.failed,
	)
	return nil
}

// Testing helper methods

// GetSentEmails returns all sent emails for testing verification
func (m *MockClient) GetSentEmails() []MockSentEmail {
	m.mu.RLock()
	defer m.mu.RUnlock()

	emails := make([]MockSentEmail, len(m.sentEmails))
	copy(emails, m.sentEmails)
	return emails
}

// GetLastSentEmail returns the last sent email for testing verification
func (m *MockClient) GetLastSentEmail() *MockSentEmail {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sentEmails) == 0 {
		return nil
	}
	return &m.sentEmails[len(m.sentEmails)-1]
}

// ClearSentEmails clears the sent emails history
func (m *MockClient) ClearSentEmails() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentEmails = make([]MockSentEmail, 0)
	m.stats = &mockStats{}
}

// SetFailRate sets the failure rate for testing
func (m *MockClient) SetFailRate(rate float64) {
	m.config.MockFailRate = rate
}

// SetDelay sets the processing delay for testing
func (m *MockClient) SetDelay(delay time.Duration) {
	m.config.MockDelay = delay
}

// Helper methods

func (m *MockClient) shouldFail() bool {
	if m.config.MockFailRate <= 0 {
		return false
	}
	return rand.Float64() < m.config.MockFailRate
}

func (m *MockClient) recordSentEmail(email *MockSentEmail) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = append(m.sentEmails, *email)
}

func (m *MockClient) validateMessage(message *Message) error {
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
	from := m.getFromAddress(message.From)
	if err := m.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := m.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (m *MockClient) validateTemplateMessage(message *TemplateMessage) error {
	if len(message.To) == 0 {
		return ErrMissingRecipients
	}
	if message.TemplateID == "" {
		return fmt.Errorf("template ID is required")
	}

	// Validate email addresses
	from := m.getFromAddress(message.From)
	if err := m.ValidateEmail(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	for _, to := range message.To {
		if err := m.ValidateEmail(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}

	return nil
}

func (m *MockClient) getFromAddress(from string) string {
	if from == "" {
		return m.config.DefaultFrom
	}
	return from
}
