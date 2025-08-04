package domain

import (
	"context"
	"net/http"
)

/*****************************
*        Email errors        *
*****************************/
var (
	ErrEmailSendFailed = &DetailedError{
		IDField:         "EMAIL_SEND_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to send email",
		StatusCodeField: http.StatusInternalServerError,
	}
)

/***************************************
*       Email entities and types       *
***************************************/
type EmailCode string

const (
	EmailCodeVerification  EmailCode = "verification"
	EmailCodePasswordReset EmailCode = "password_reset"
	EmailCodeWelcome       EmailCode = "welcome"
)

type EmailStatus string

const (
	EmailStatusSuccess EmailStatus = "success"
	EmailStatusFailed  EmailStatus = "failed"
	EmailStatusPending EmailStatus = "pending"
)

type EmailProvider string

const (
	EmailProviderSMTP     EmailProvider = "smtp"
	EmailProviderSendGrid EmailProvider = "sendgrid"
	EmailProviderSES      EmailProvider = "ses"
)

type EmailLog struct {
	SQLModel

	// Recipients - JSON arrays to store multiple email addresses
	To  StringSlice `json:"to" gorm:"type:jsonb;not null"` // JSON array of primary recipients
	CC  StringSlice `json:"cc" gorm:"type:jsonb"`          // JSON array of CC recipients
	BCC StringSlice `json:"bcc" gorm:"type:jsonb"`         // JSON array of BCC recipients

	Subject string `json:"subject" gorm:"type:varchar(255)"` // Email subject
	Content string `json:"content" gorm:"type:text"`         // Rendered email content

	Template string `json:"template" gorm:"type:varchar(64)"` // Template name used for rendering
	Data     JSONB  `json:"data" gorm:"type:jsonb"`           // Marshaled data for template rendering

	Status     EmailStatus   `json:"status" gorm:"type:varchar(32)"`     // "success" or "failed"
	ErrorMsg   string        `json:"error_msg" gorm:"type:text"`         // Error message if failed
	SentAt     int64         `json:"sent_at"`                            // Unix timestamp when sent
	RequestID  string        `json:"request_id" gorm:"type:varchar(64)"` // Trace/debug request ID
	RetryCount int           `json:"retry_count" gorm:"default:0"`       // Number of send attempts
	Headers    JSONB         `json:"headers" gorm:"type:jsonb"`          // Email headers (JSON string)
	Response   string        `json:"response" gorm:"type:text"`          // Raw response from email provider
	Provider   EmailProvider `json:"provider" gorm:"type:varchar(64)"`   // Email provider/service name

	// Additional tracking fields
	TotalRecipients int    `json:"total_recipients" gorm:"default:0"`    // Total number of recipients (TO + CC + BCC)
	ContentType     string `json:"content_type" gorm:"type:varchar(32)"` // "text/plain" or "text/html"
	AttachmentCount int    `json:"attachment_count" gorm:"default:0"`    // Number of attachments
	MessageSize     int64  `json:"message_size" gorm:"default:0"`        // Total message size in bytes
}

func (e *EmailLog) GetToEmails() []string {
	return []string(e.To)
}

func (e *EmailLog) SetToEmails(emails []string) {
	e.To = NewStringSlice(emails)
}

func (e *EmailLog) GetCCEmails() []string {
	return []string(e.CC)
}

func (e *EmailLog) SetCCEmails(emails []string) {
	e.CC = NewStringSlice(emails)
}

func (e *EmailLog) GetBCCEmails() []string {
	return []string(e.BCC)
}

func (e *EmailLog) SetBCCEmails(emails []string) {
	e.BCC = NewStringSlice(emails)
}

func (e *EmailLog) GetAllRecipients() []string {
	var allEmails []string
	allEmails = append(allEmails, e.GetToEmails()...)
	allEmails = append(allEmails, e.GetCCEmails()...)
	allEmails = append(allEmails, e.GetBCCEmails()...)
	return allEmails
}

func (e *EmailLog) UpdateTotalRecipients() {
	e.TotalRecipients = len(e.GetAllRecipients())
}

type EmailLogFilter struct {
	ID           *string        `json:"id,omitempty"`
	To           *string        `json:"to,omitempty"`            // Search in TO recipients
	CC           *string        `json:"cc,omitempty"`            // Search in CC recipients
	BCC          *string        `json:"bcc,omitempty"`           // Search in BCC recipients
	AnyRecipient *string        `json:"any_recipient,omitempty"` // Search across all recipient types
	Status       *EmailStatus   `json:"status,omitempty"`
	Provider     *EmailProvider `json:"provider,omitempty"`
	Template     *string        `json:"template,omitempty"`
	RequestID    *string        `json:"request_id,omitempty"`

	// Date filters
	SentAfter  *int64 `json:"sent_after,omitempty"`  // Unix timestamp
	SentBefore *int64 `json:"sent_before,omitempty"` // Unix timestamp

	CreatedAfter  *int64 `json:"created_after,omitempty"`
	CreatedBefore *int64 `json:"created_before,omitempty"`

	// Retry filters
	MinRetryCount *int `json:"min_retry_count,omitempty"`
	MaxRetryCount *int `json:"max_retry_count,omitempty"`

	// Additional filters
	MinRecipients  *int    `json:"min_recipients,omitempty"`   // Filter by minimum number of recipients
	MaxRecipients  *int    `json:"max_recipients,omitempty"`   // Filter by maximum number of recipients
	ContentType    *string `json:"content_type,omitempty"`     // Filter by content type
	HasAttachments *bool   `json:"has_attachments,omitempty"`  // Filter emails with/without attachments
	MinMessageSize *int64  `json:"min_message_size,omitempty"` // Filter by minimum message size
	MaxMessageSize *int64  `json:"max_message_size,omitempty"` // Filter by maximum message size

	SearchTerm     *string  `json:"search_term,omitempty"`
	SearchFields   []string `json:"search_fields,omitempty"` // to, cc, bcc, subject, template, content
	IncludeDeleted *bool    `json:"include_deleted" form:"include_deleted"`
}

type EmailTemplate struct {
	SQLModel
	Code        EmailCode `json:"code" gorm:"type:varchar(32);not null;index"` // Email code/type
	Name        string    `json:"name" gorm:"type:varchar(64);not null"`       // Template name
	Subject     string    `json:"subject" gorm:"type:varchar(255);not null"`   // Default subject
	Content     string    `json:"content" gorm:"type:text;not null"`           // Template body (can be HTML/text)
	IsActive    bool      `json:"is_active" gorm:"default:true"`               // Is template active/usable
	Description string    `json:"description" gorm:"type:text"`                // Optional description
	Locale      string    `json:"locale" gorm:"type:varchar(16)"`              // Language/locale code (e.g. "en", "vi")
}
type EmailTemplateFilter struct {
	ID       *string    `json:"id,omitempty"`
	Code     *EmailCode `json:"code,omitempty"`
	Name     *string    `json:"name,omitempty"`
	Locale   *string    `json:"locale,omitempty"`
	IsActive *bool      `json:"is_active,omitempty"`

	SearchTerm   *string  `json:"search_term,omitempty"`
	SearchFields []string `json:"search_fields,omitempty"` // name, subject, description

	IncludeDeleted *bool `json:"include_deleted,omitempty"`
}

/*************************************
*  Email usecase interfaces and types *
**************************************/
type EmailUsecase interface {
	// Email sending operations
	SendEmail(ctx context.Context, req *SendEmailRequest) (*EmailLog, error)
	SendEmailWithTemplate(ctx context.Context, req *SendEmailWithTemplateRequest) (*EmailLog, error)
	SendBulkEmail(ctx context.Context, req *SendBulkEmailRequest) ([]*EmailLog, error)
	ResendEmail(ctx context.Context, emailLogID string) (*EmailLog, error)

	// Email template operations
	CreateTemplate(ctx context.Context, req *CreateEmailTemplateRequest) (*EmailTemplate, error)
	FindTemplate(ctx context.Context, code EmailCode, locale string) (*EmailTemplate, error)
	FindTemplateByID(ctx context.Context, templateID string) (*EmailTemplate, error)
	UpdateTemplate(ctx context.Context, templateID string, req *UpdateEmailTemplateRequest) (*EmailTemplate, error)
	DeleteTemplate(ctx context.Context, templateID string) error
	FindPageTemplates(ctx context.Context, filter *EmailTemplateFilter, option *FindPageOption) ([]*EmailTemplate, *Pagination, error)

	// Email log operations
	GetEmailLog(ctx context.Context, emailLogID string) (*EmailLog, error)
	GetEmailLogs(ctx context.Context, filter *EmailLogFilter, option *FindPageOption) ([]*EmailLog, *Pagination, error)
	GetEmailStats(ctx context.Context, filter *EmailStatsFilter) (*EmailStats, error)
}

// Email sending request types
type SendEmailRequest struct {
	To          []string           `json:"to" validate:"required,min=1"`
	CC          []string           `json:"cc,omitempty"`
	BCC         []string           `json:"bcc,omitempty"`
	Subject     string             `json:"subject" validate:"required"`
	Content     string             `json:"content" validate:"required"`
	ContentType string             `json:"content_type" validate:"required,oneof=text/plain text/html"` // "text/plain" or "text/html"
	Attachments []*EmailAttachment `json:"attachments,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	Provider    EmailProvider      `json:"provider,omitempty"`
	RequestID   string             `json:"request_id,omitempty"`
}

type SendEmailWithTemplateRequest struct {
	To           []string               `json:"to" validate:"required,min=1"`
	CC           []string               `json:"cc,omitempty"`
	BCC          []string               `json:"bcc,omitempty"`
	TemplateCode EmailCode              `json:"template_code" validate:"required"`
	Locale       string                 `json:"locale,omitempty"` // defaults to "en"
	Data         map[string]interface{} `json:"data,omitempty"`
	Attachments  []*EmailAttachment     `json:"attachments,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Provider     EmailProvider          `json:"provider,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
}

type SendBulkEmailRequest struct {
	Recipients   []*BulkEmailRecipient `json:"recipients" validate:"required,min=1"`
	TemplateCode EmailCode             `json:"template_code" validate:"required"`
	Locale       string                `json:"locale,omitempty"`
	Provider     EmailProvider         `json:"provider,omitempty"`
	RequestID    string                `json:"request_id,omitempty"`
}

type BulkEmailRecipient struct {
	To   string                 `json:"to" validate:"required,email"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type EmailAttachment struct {
	Filename    string `json:"filename" validate:"required"`
	Content     []byte `json:"content" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Inline      bool   `json:"inline,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// Email template request types
type CreateEmailTemplateRequest struct {
	Code        EmailCode `json:"code" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=64"`
	Subject     string    `json:"subject" validate:"required,min=1,max=255"`
	Content     string    `json:"content" validate:"required"`
	Description string    `json:"description,omitempty"`
	Locale      string    `json:"locale,omitempty"` // defaults to "en"
	IsActive    *bool     `json:"is_active,omitempty"`
}

type UpdateEmailTemplateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=64"`
	Subject     *string `json:"subject,omitempty" validate:"omitempty,min=1,max=255"`
	Content     *string `json:"content,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type EmailStatsFilter struct {
	Provider     *EmailProvider `json:"provider,omitempty"`
	Template     *string        `json:"template,omitempty"`
	Status       *EmailStatus   `json:"status,omitempty"`
	DateFrom     *int64         `json:"date_from,omitempty"` // Unix timestamp
	DateTo       *int64         `json:"date_to,omitempty"`   // Unix timestamp
	GroupBy      string         `json:"group_by,omitempty"`  // "day", "week", "month", "provider", "template", "status"
	IncludeTotal bool           `json:"include_total,omitempty"`
}

// Response types
type EmailStats struct {
	TotalSent     int64                 `json:"total_sent"`
	TotalSuccess  int64                 `json:"total_success"`
	TotalFailed   int64                 `json:"total_failed"`
	TotalPending  int64                 `json:"total_pending"`
	SuccessRate   float64               `json:"success_rate"`
	FailureRate   float64               `json:"failure_rate"`
	AvgRetryCount float64               `json:"avg_retry_count"`
	GroupedStats  []*EmailStatsGroup    `json:"grouped_stats,omitempty"`
	ProviderStats []*EmailProviderStats `json:"provider_stats,omitempty"`
	TemplateStats []*EmailTemplateStats `json:"template_stats,omitempty"`
	DateRange     *EmailStatsDateRange  `json:"date_range,omitempty"`
}

type EmailStatsGroup struct {
	Key   string `json:"key"`   // Group key (date, provider, template, etc.)
	Label string `json:"label"` // Human readable label
	Count int64  `json:"count"`

	Sent    int64   `json:"sent"`
	Success int64   `json:"success"`
	Failed  int64   `json:"failed"`
	Pending int64   `json:"pending"`
	Rate    float64 `json:"success_rate"`
}

type EmailProviderStats struct {
	Provider      EmailProvider `json:"provider"`
	TotalSent     int64         `json:"total_sent"`
	TotalSuccess  int64         `json:"total_success"`
	TotalFailed   int64         `json:"total_failed"`
	SuccessRate   float64       `json:"success_rate"`
	AvgRetryCount float64       `json:"avg_retry_count"`
}

type EmailTemplateStats struct {
	Template     string    `json:"template"`
	Code         EmailCode `json:"code"`
	TotalSent    int64     `json:"total_sent"`
	TotalSuccess int64     `json:"total_success"`
	TotalFailed  int64     `json:"total_failed"`
	SuccessRate  float64   `json:"success_rate"`
	LastUsed     int64     `json:"last_used"` // Unix timestamp
}

type EmailStatsDateRange struct {
	From int64 `json:"from"` // Unix timestamp
	To   int64 `json:"to"`   // Unix timestamp
}

// Email validation types
type EmailValidationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type EmailValidationResponse struct {
	Email       string `json:"email"`
	IsValid     bool   `json:"is_valid"`
	IsReachable *bool  `json:"is_reachable,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// Email preview types (for template testing)
type EmailPreviewRequest struct {
	TemplateCode EmailCode              `json:"template_code" validate:"required"`
	Locale       string                 `json:"locale,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

type EmailPreviewResponse struct {
	Subject string                 `json:"subject"`
	Content string                 `json:"content"`
	Data    map[string]interface{} `json:"data"`
}
