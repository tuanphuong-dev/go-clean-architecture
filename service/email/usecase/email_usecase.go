package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/email"
	"go-clean-arch/pkg/log"
	"time"
)

// EmailLogRepository defines the interface for email log operations
type EmailLogRepository interface {
	Create(ctx context.Context, emailLog *domain.EmailLog) error
	FindByID(ctx context.Context, emailLogID string, option *domain.FindOneOption) (*domain.EmailLog, error)
	FindOne(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindOneOption) (*domain.EmailLog, error)
	FindMany(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindManyOption) ([]*domain.EmailLog, error)
	FindPage(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindPageOption) ([]*domain.EmailLog, *domain.Pagination, error)
	Update(ctx context.Context, emailLog *domain.EmailLog) error
	UpdateFields(ctx context.Context, id string, fields map[string]any) error
	Delete(ctx context.Context, emailLogID string) error
	Count(ctx context.Context, filter *domain.EmailLogFilter) (int64, error)
	GetStats(ctx context.Context, filter *domain.EmailStatsFilter) (*domain.EmailStats, error)
}

// EmailTemplateRepository defines the interface for email template operations
type EmailTemplateRepository interface {
	Create(ctx context.Context, template *domain.EmailTemplate) error
	FindByID(ctx context.Context, templateID string, option *domain.FindOneOption) (*domain.EmailTemplate, error)
	FindByCodeAndLocale(ctx context.Context, code domain.EmailCode, locale string, option *domain.FindOneOption) (*domain.EmailTemplate, error)
	FindOne(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindOneOption) (*domain.EmailTemplate, error)
	FindMany(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindManyOption) ([]*domain.EmailTemplate, error)
	FindPage(ctx context.Context, filter *domain.EmailTemplateFilter, option *domain.FindPageOption) ([]*domain.EmailTemplate, *domain.Pagination, error)
	Update(ctx context.Context, template *domain.EmailTemplate) error
	UpdateFields(ctx context.Context, id string, fields map[string]any) error
	Delete(ctx context.Context, templateID string) error
	Count(ctx context.Context, filter *domain.EmailTemplateFilter) (int64, error)
	// Template-specific methods
	ActivateTemplate(ctx context.Context, templateID string) error
	DeactivateTemplate(ctx context.Context, templateID string) error
	GetActiveTemplatesByCode(ctx context.Context, code domain.EmailCode) ([]*domain.EmailTemplate, error)
}

// EmailClient defines the interface for email sending operations
type EmailClient interface {
	Send(ctx context.Context, message *email.Message) error
	SendBulk(ctx context.Context, messages []*email.Message) error
	SendTemplate(ctx context.Context, templateMessage *email.TemplateMessage) error
	SendBulkTemplate(ctx context.Context, templateMessages []*email.TemplateMessage) error
	ValidateEmail(email string) error
	GetStats(ctx context.Context) (email.Stats, error)
	Close() error
}

// TemplateRenderer defines the interface for rendering email templates
type TemplateRenderer interface {
	RenderTemplate(template *domain.EmailTemplate, data map[string]interface{}) (subject, content string, err error)
	ValidateTemplate(template *domain.EmailTemplate) error
	GetRequiredFields(template *domain.EmailTemplate) ([]string, error)
}

// EmailUsecase implementation
type emailUsecase struct {
	emailLogRepo     EmailLogRepository
	templateRepo     EmailTemplateRepository
	emailClient      EmailClient
	templateRenderer TemplateRenderer
	logger           log.Logger
}

// NewEmailUsecase creates a new email usecase instance
func NewEmailUsecase(
	emailLogRepo EmailLogRepository,
	templateRepo EmailTemplateRepository,
	emailClient EmailClient,
	templateRenderer TemplateRenderer,
	logger log.Logger,
) domain.EmailUsecase {
	return &emailUsecase{
		emailLogRepo:     emailLogRepo,
		templateRepo:     templateRepo,
		emailClient:      emailClient,
		templateRenderer: templateRenderer,
		logger:           logger,
	}
}

func (u *emailUsecase) SendEmail(ctx context.Context, req *domain.SendEmailRequest) (*domain.EmailLog, error) {
	u.logger.Debug("Sending email", log.Any("to", req.To), log.String("subject", req.Subject))

	// Create email message
	message := &email.Message{
		To:      req.To,
		From:    "demo@example.com", // TODO set from address properly
		CC:      req.CC,
		BCC:     req.BCC,
		Subject: req.Subject,
		Headers: req.Headers,
	}

	// Set content based on type
	if req.ContentType == "text/html" {
		message.HTML = req.Content
	} else {
		message.Text = req.Content
	}

	// Add attachments if any
	for _, attachment := range req.Attachments {
		emailAttachment := &email.Attachment{
			Filename:    attachment.Filename,
			Content:     attachment.Content,
			ContentType: attachment.ContentType,
			Inline:      attachment.Inline,
			ContentID:   attachment.ContentID,
		}
		message.Attachments = append(message.Attachments, emailAttachment)
	}

	// Create email log entry
	emailLog := &domain.EmailLog{
		Subject:         req.Subject,
		Content:         req.Content,
		Status:          domain.EmailStatusPending,
		RequestID:       req.RequestID,
		Provider:        req.Provider,
		ContentType:     req.ContentType,
		AttachmentCount: len(req.Attachments),
		MessageSize:     int64(len(req.Content)),
	}

	// Set recipient arrays
	emailLog.To = domain.NewStringSlice(req.To)
	if len(req.CC) > 0 {
		emailLog.CC = domain.NewStringSlice(req.CC)
	}
	if len(req.BCC) > 0 {
		emailLog.BCC = domain.NewStringSlice(req.BCC)
	}

	// Set headers and total recipients
	if len(req.Headers) > 0 {
		if err := emailLog.Headers.Scan(req.Headers); err != nil {
			return nil, domain.ErrEmailSendFailed.WithWrap(err)
		}
	}
	emailLog.TotalRecipients = len(req.To) + len(req.CC) + len(req.BCC)

	// Save initial log entry
	if err := u.emailLogRepo.Create(ctx, emailLog); err != nil {
		return nil, domain.ErrEmailSendFailed.WithWrap(err)
	}

	// Send email
	err := u.emailClient.Send(ctx, message)
	if err != nil {
		// Update log with failure
		emailLog.Status = domain.EmailStatusFailed
		emailLog.ErrorMsg = err.Error()
		u.emailLogRepo.Update(ctx, emailLog)

		u.logger.Error("Failed to send email",
			log.String("email_log_id", emailLog.ID),
			log.Error(err),
		)
		return emailLog, domain.ErrEmailSendFailed.WithWrap(err)
	}

	// Update log with success
	emailLog.Status = domain.EmailStatusSuccess
	emailLog.SentAt = time.Now().UnixMilli()
	if err := u.emailLogRepo.Update(ctx, emailLog); err != nil {
		u.logger.Error("Failed to update email log", log.Error(err))
	}

	u.logger.Info("Email sent successfully",
		log.String("email_log_id", emailLog.ID),
		log.Int("to_count", len(req.To)),
		log.String("subject", req.Subject),
	)

	return emailLog, nil
}

func (u *emailUsecase) SendEmailWithTemplate(
	ctx context.Context,
	req *domain.SendEmailWithTemplateRequest,

) (*domain.EmailLog, error) {
	u.logger.Debug("Sending email with template",
		log.Any("template_code", req.TemplateCode),
		log.String("locale", req.Locale),
		log.Any("to", req.To),
	)

	// Get template
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	template, err := u.templateRepo.FindByCodeAndLocale(ctx, req.TemplateCode, locale, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithError("email template not found")
	}

	if !template.IsActive {
		return nil, domain.ErrNotFound.WithError("email template is not active")
	}

	// Render template
	subject, content, err := u.templateRenderer.RenderTemplate(template, req.Data)
	if err != nil {
		return nil, domain.ErrEmailSendFailed.WithError("failed to render template").WithWrap(err)
	}

	// Create send email request
	sendReq := &domain.SendEmailRequest{
		To:          req.To,
		CC:          req.CC,
		BCC:         req.BCC,
		Subject:     subject,
		Content:     content,
		ContentType: "text/html", // Templates are typically HTML
		Attachments: req.Attachments,
		Headers:     req.Headers,
		Provider:    req.Provider,
		RequestID:   req.RequestID,
	}

	// Send email
	emailLog, err := u.SendEmail(ctx, sendReq)
	if err != nil {
		return emailLog, err
	}

	// Update log with template information
	emailLog.Template = string(req.TemplateCode)
	if len(req.Data) > 0 {
		if err := emailLog.Data.Scan(req.Data); err != nil {
			u.logger.Error("Failed to save template data", log.Error(err))
		}
	}

	if err := u.emailLogRepo.Update(ctx, emailLog); err != nil {
		u.logger.Error("Failed to update email log with template info", log.Error(err))
	}

	return emailLog, nil
}

func (u *emailUsecase) SendBulkEmail(ctx context.Context, req *domain.SendBulkEmailRequest) ([]*domain.EmailLog, error) {
	u.logger.Debug("Sending bulk email",
		log.Any("template_code", req.TemplateCode),
		log.Int("recipient_count", len(req.Recipients)),
	)

	var emailLogs []*domain.EmailLog
	var errors []string

	for _, recipient := range req.Recipients {
		// Create individual send request
		sendReq := &domain.SendEmailWithTemplateRequest{
			To:           []string{recipient.To},
			TemplateCode: req.TemplateCode,
			Locale:       req.Locale,
			Data:         recipient.Data,
			Provider:     req.Provider,
			RequestID:    req.RequestID,
		}

		emailLog, err := u.SendEmailWithTemplate(ctx, sendReq)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to send to %s: %v", recipient.To, err))
			u.logger.Error("Failed to send bulk email to recipient",
				log.String("recipient", recipient.To),
				log.Error(err),
			)
		}

		if emailLog != nil {
			emailLogs = append(emailLogs, emailLog)
		}
	}

	// If there were any errors, log them
	if len(errors) > 0 {
		u.logger.Error("Bulk email sending completed with errors", log.Int("error_count", len(errors)), log.Int("success_count", len(emailLogs)))
	} else {
		u.logger.Info("Bulk email sending completed successfully", log.Int("success_count", len(emailLogs)))
	}

	return emailLogs, nil
}

func (u *emailUsecase) ResendEmail(ctx context.Context, emailLogID string) (*domain.EmailLog, error) {
	u.logger.Debug("Resending email", log.String("email_log_id", emailLogID))

	// Get original email log
	originalLog, err := u.emailLogRepo.FindByID(ctx, emailLogID, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithWrap(err)
	}

	// Parse recipients
	toEmails := originalLog.GetToEmails()
	ccEmails := originalLog.GetCCEmails()
	bccEmails := originalLog.GetBCCEmails()

	// Check if it was a template email
	if originalLog.Template != "" {
		// Parse template data
		var templateData map[string]interface{}
		if originalLog.Data != nil {
			json.Unmarshal([]byte(fmt.Sprintf("%v", originalLog.Data)), &templateData)
		}

		// Resend with template
		req := &domain.SendEmailWithTemplateRequest{
			To:           toEmails,
			CC:           ccEmails,
			BCC:          bccEmails,
			TemplateCode: domain.EmailCode(originalLog.Template),
			Data:         templateData,
			Provider:     originalLog.Provider,
			RequestID:    originalLog.RequestID,
		}

		return u.SendEmailWithTemplate(ctx, req)
	} else {
		// Resend as regular email
		req := &domain.SendEmailRequest{
			To:          toEmails,
			CC:          ccEmails,
			BCC:         bccEmails,
			Subject:     originalLog.Subject,
			Content:     originalLog.Content,
			ContentType: originalLog.ContentType,
			Provider:    originalLog.Provider,
			RequestID:   originalLog.RequestID,
		}

		return u.SendEmail(ctx, req)
	}
}

func (u *emailUsecase) CreateTemplate(
	ctx context.Context,
	req *domain.CreateEmailTemplateRequest,

) (*domain.EmailTemplate, error) {
	u.logger.Debug("Creating email template", log.Any("code", req.Code), log.String("name", req.Name))

	// Set defaults
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Check if template already exists
	existing, err := u.templateRepo.FindByCodeAndLocale(ctx, req.Code, locale, nil)
	if err == nil && existing != nil {
		return nil, domain.ErrConflict.WithError("template with this code and locale already exists")
	}

	// Create template
	template := &domain.EmailTemplate{
		Code:        req.Code,
		Name:        req.Name,
		Subject:     req.Subject,
		Content:     req.Content,
		Description: req.Description,
		Locale:      locale,
		IsActive:    isActive,
	}

	// Validate template
	if u.templateRenderer != nil {
		if err := u.templateRenderer.ValidateTemplate(template); err != nil {
			return nil, domain.ErrBadRequest.WithError("template validation failed").WithWrap(err)
		}
	}

	if err := u.templateRepo.Create(ctx, template); err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	u.logger.Info("Email template created successfully",
		log.String("template_id", template.ID),
		log.Any("code", req.Code),
		log.String("locale", locale),
	)

	return template, nil
}

func (u *emailUsecase) FindTemplate(
	ctx context.Context,
	code domain.EmailCode,
	locale string,

) (*domain.EmailTemplate, error) {
	u.logger.Debug("Getting email template", log.Any("code", code), log.String("locale", locale))
	if locale == "" {
		locale = "en"
	}

	template, err := u.templateRepo.FindByCodeAndLocale(ctx, code, locale, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithWrap(err)
	}

	if template == nil {
		return nil, domain.ErrNotFound.WithError("email template not found")
	}

	u.logger.Debug("Email template retrieved successfully", log.String("template_id", template.ID))
	return template, nil
}

func (u *emailUsecase) FindTemplateByID(
	ctx context.Context,
	templateID string,

) (*domain.EmailTemplate, error) {
	u.logger.Debug("Getting email template by ID", log.String("template_id", templateID))

	template, err := u.templateRepo.FindByID(ctx, templateID, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithWrap(err)
	}

	if template == nil {
		return nil, domain.ErrNotFound.WithError("email template not found")
	}

	u.logger.Debug("Email template retrieved successfully", log.String("template_id", templateID))
	return template, nil
}

func (u *emailUsecase) FindPageTemplates(
	ctx context.Context,
	filter *domain.EmailTemplateFilter,
	option *domain.FindPageOption,

) ([]*domain.EmailTemplate, *domain.Pagination, error) {
	templates, pagination, err := u.templateRepo.FindPage(ctx, filter, option)
	if err != nil {
		return nil, nil, domain.ErrInternalServerError.WithWrap(err)
	}

	return templates, pagination, nil
}

func (u *emailUsecase) UpdateTemplate(
	ctx context.Context,
	templateID string,
	req *domain.UpdateEmailTemplateRequest,

) (*domain.EmailTemplate, error) {
	u.logger.Debug("Updating email template", log.String("template_id", templateID))

	// Get existing template
	template, err := u.templateRepo.FindByID(ctx, templateID, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithWrap(err)
	}

	// Update fields
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Subject != nil {
		template.Subject = *req.Subject
	}
	if req.Content != nil {
		template.Content = *req.Content
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	// Validate template
	if u.templateRenderer != nil {
		if err := u.templateRenderer.ValidateTemplate(template); err != nil {
			return nil, domain.ErrBadRequest.WithError("template validation failed").WithWrap(err)
		}
	}

	if err := u.templateRepo.Update(ctx, template); err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	u.logger.Info("Email template updated successfully", log.String("template_id", templateID))

	return template, nil
}

func (u *emailUsecase) DeleteTemplate(ctx context.Context, templateID string) error {
	u.logger.Debug("Deleting email template", log.String("template_id", templateID))

	// Check if template exists
	_, err := u.templateRepo.FindByID(ctx, templateID, nil)
	if err != nil {
		return domain.ErrNotFound.WithWrap(err)
	}

	if err := u.templateRepo.Delete(ctx, templateID); err != nil {
		return domain.ErrInternalServerError.WithWrap(err)
	}

	u.logger.Info("Email template deleted successfully", log.String("template_id", templateID))

	return nil
}

func (u *emailUsecase) GetEmailLog(ctx context.Context, emailLogID string) (*domain.EmailLog, error) {
	u.logger.Debug("Getting email log", log.String("email_log_id", emailLogID))

	emailLog, err := u.emailLogRepo.FindByID(ctx, emailLogID, nil)
	if err != nil {
		return nil, domain.ErrNotFound.WithWrap(err)
	}

	if emailLog == nil {
		return nil, domain.ErrNotFound.WithError("email log not found")
	}

	u.logger.Debug("Email log retrieved successfully", log.String("email_log_id", emailLogID))
	return emailLog, nil
}

func (u *emailUsecase) GetEmailLogs(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindPageOption) ([]*domain.EmailLog, *domain.Pagination, error) {
	u.logger.Debug("Getting email logs",
		log.Any("filter", filter),
		log.Any("option", option),
	)

	emailLogs, pagination, err := u.emailLogRepo.FindPage(ctx, filter, option)
	if err != nil {
		return nil, nil, domain.ErrInternalServerError.WithWrap(err)
	}

	u.logger.Debug("Email logs retrieved successfully",
		log.Int("count", len(emailLogs)),
		log.Any("pagination", pagination),
	)

	return emailLogs, pagination, nil
}

func (u *emailUsecase) GetEmailStats(ctx context.Context, filter *domain.EmailStatsFilter) (*domain.EmailStats, error) {
	u.logger.Debug("Getting email statistics", log.Any("filter", filter))

	stats, err := u.emailLogRepo.GetStats(ctx, filter)
	if err != nil {
		return nil, domain.ErrInternalServerError.WithWrap(err)
	}

	u.logger.Debug("Email statistics retrieved successfully",
		log.Int64("total_sent", stats.TotalSent),
		log.Float64("success_rate", stats.SuccessRate),
	)

	return stats, nil
}
