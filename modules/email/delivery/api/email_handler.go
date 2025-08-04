package api

import (
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/middleware"
	"go-clean-arch/pkg/log"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	usecase          domain.EmailUsecase
	templateRenderer TemplateRenderer
	logger           log.Logger
	middlewares      middleware.Middlewares
}

// Add TemplateRenderer interface for preview functionality
type TemplateRenderer interface {
	RenderTemplate(template *domain.EmailTemplate, data map[string]interface{}) (subject, content string, err error)
	ValidateTemplate(template *domain.EmailTemplate) error
	GetRequiredFields(template *domain.EmailTemplate) ([]string, error)
}

func NewEmailHandler(usecase domain.EmailUsecase, templateRenderer TemplateRenderer, logger log.Logger, middlewares middleware.Middlewares) *EmailHandler {
	return &EmailHandler{
		usecase:          usecase,
		templateRenderer: templateRenderer,
		logger:           logger,
		middlewares:      middlewares,
	}
}

func (h *EmailHandler) RegisterRoutes(rg *gin.RouterGroup) {
	email := rg.Group("/emails")

	// Apply authentication middleware for all email routes
	email.Use(h.middlewares.Authenticator())
	// Require admin or super admin roles for all email operations
	email.Use(h.middlewares.RequireAnyRoles(domain.RoleIDAdmin, domain.RoleIDSuperAdmin))
	// Apply admin-specific rate limiting
	email.Use(h.middlewares.AdminRateLimits())

	// Email sending operations
	{
		email.POST("", h.SendEmail)
		email.POST("/template", h.SendEmailWithTemplate)
		email.POST("/bulk", h.SendBulkEmail)
		email.POST("/resend/:id", h.ResendEmail)
	}

	// Email template operations
	templates := email.Group("/templates")
	{
		templates.POST("", h.CreateTemplate)
		templates.GET("/:id", h.GetTemplateByID)
		templates.GET("", h.ListTemplates)
		templates.PUT("/:id", h.UpdateTemplate)
		templates.DELETE("/:id", h.DeleteTemplate)
		templates.GET("/code/:code", h.GetTemplateByCode)
		templates.POST("/preview", h.PreviewTemplate)
	}

	// Email log operations
	logs := email.Group("/logs")
	{
		logs.GET("/:id", h.GetEmailLog)
		logs.GET("", h.GetEmailLogs)
		logs.GET("/stats", h.GetEmailStats)
	}
}

// Email sending operations
func (h *EmailHandler) SendEmail(c *gin.Context) {
	var req domain.SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind send email request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	// Set request ID from context if available
	if reqID := c.GetHeader("X-Request-ID"); reqID != "" {
		req.RequestID = reqID
	}

	emailLog, err := h.usecase.SendEmail(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to send email",
			log.Error(err),
			log.String("to", req.To[0]),
			log.String("subject", req.Subject),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Email sent successfully",
		log.String("email_log_id", emailLog.ID),
		log.String("to", req.To[0]),
		log.String("subject", req.Subject),
	)

	common.ResponseCreated(c, emailLog, "Email sent successfully")
}

func (h *EmailHandler) SendEmailWithTemplate(c *gin.Context) {
	var req domain.SendEmailWithTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind send template email request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	// Set request ID from context if available
	if reqID := c.GetHeader("X-Request-ID"); reqID != "" {
		req.RequestID = reqID
	}

	emailLog, err := h.usecase.SendEmailWithTemplate(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to send template email",
			log.Error(err),
			log.String("template_code", string(req.TemplateCode)),
			log.String("to", req.To[0]),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Template email sent successfully",
		log.String("email_log_id", emailLog.ID),
		log.String("template_code", string(req.TemplateCode)),
		log.String("to", req.To[0]),
	)

	common.ResponseCreated(c, emailLog, "Template email sent successfully")
}

func (h *EmailHandler) SendBulkEmail(c *gin.Context) {
	var req domain.SendBulkEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind bulk email request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	// Set request ID from context if available
	if reqID := c.GetHeader("X-Request-ID"); reqID != "" {
		req.RequestID = reqID
	}

	emailLogs, err := h.usecase.SendBulkEmail(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to send bulk email",
			log.Error(err),
			log.String("template_code", string(req.TemplateCode)),
			log.Int("recipient_count", len(req.Recipients)),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Bulk email sent successfully",
		log.String("template_code", string(req.TemplateCode)),
		log.Int("recipient_count", len(req.Recipients)),
		log.Int("sent_count", len(emailLogs)),
	)

	common.ResponseCreated(c, emailLogs, "Bulk email sent successfully")
}

func (h *EmailHandler) ResendEmail(c *gin.Context) {
	emailLogID := c.Param("id")
	if emailLogID == "" {
		common.ResponseBadRequest(c, "Email log ID is required")
		return
	}

	emailLog, err := h.usecase.ResendEmail(c.Request.Context(), emailLogID)
	if err != nil {
		h.logger.Error("Failed to resend email",
			log.Error(err),
			log.String("original_email_log_id", emailLogID),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Email resent successfully",
		log.String("original_email_log_id", emailLogID),
		log.String("new_email_log_id", emailLog.ID),
	)

	common.ResponseCreated(c, emailLog, "Email resent successfully")
}

// Email template operations
func (h *EmailHandler) CreateTemplate(c *gin.Context) {
	var req domain.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create template request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	template, err := h.usecase.CreateTemplate(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create email template",
			log.Error(err),
			log.String("code", string(req.Code)),
			log.String("name", req.Name),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Email template created successfully",
		log.String("template_id", template.ID),
		log.String("code", string(req.Code)),
		log.String("name", req.Name),
	)

	common.ResponseCreated(c, template, "Email template created successfully")
}

func (h *EmailHandler) GetTemplateByID(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		common.ResponseBadRequest(c, "Template ID is required")
		return
	}

	template, err := h.usecase.FindTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		h.logger.Error("Failed to get email template",
			log.Error(err),
			log.String("template_id", templateID),
		)
		common.ResponseError(c, err)
		return
	}

	common.ResponseOK(c, template, "Email template retrieved successfully")
}

func (h *EmailHandler) GetTemplateByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		common.ResponseBadRequest(c, "Template code is required")
		return
	}

	locale := c.Query("locale")
	if locale == "" {
		locale = "en"
	}

	template, err := h.usecase.FindTemplate(c.Request.Context(), domain.EmailCode(code), locale)
	if err != nil {
		h.logger.Error("Failed to get email template by code",
			log.Error(err),
			log.String("code", code),
			log.String("locale", locale),
		)
		common.ResponseError(c, err)
		return
	}

	common.ResponseOK(c, template, "Email template retrieved successfully")
}

func (h *EmailHandler) ListTemplates(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &domain.EmailTemplateFilter{}

	if code := c.Query("code"); code != "" {
		emailCode := domain.EmailCode(code)
		filter.Code = &emailCode
	}
	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}
	if locale := c.Query("locale"); locale != "" {
		filter.Locale = &locale
	}
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}
	if searchTerm := c.Query("search"); searchTerm != "" {
		filter.SearchTerm = &searchTerm
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	option := &domain.FindPageOption{
		Page:    page,
		PerPage: perPage,
	}

	templates, pagination, err := h.usecase.FindPageTemplates(c.Request.Context(), filter, option)
	if err != nil {
		h.logger.Error("Failed to list email templates", log.Error(err))
		common.ResponseError(c, err)
		return
	}

	response := map[string]interface{}{
		"templates":  templates,
		"pagination": pagination,
	}

	common.ResponseOK(c, response, "Email templates retrieved successfully")
}

func (h *EmailHandler) UpdateTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		common.ResponseBadRequest(c, "Template ID is required")
		return
	}

	var req domain.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update template request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	template, err := h.usecase.UpdateTemplate(c.Request.Context(), templateID, &req)
	if err != nil {
		h.logger.Error("Failed to update email template",
			log.Error(err),
			log.String("template_id", templateID),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Email template updated successfully",
		log.String("template_id", templateID),
	)

	common.ResponseOK(c, template, "Email template updated successfully")
}

func (h *EmailHandler) DeleteTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		common.ResponseBadRequest(c, "Template ID is required")
		return
	}

	err := h.usecase.DeleteTemplate(c.Request.Context(), templateID)
	if err != nil {
		h.logger.Error("Failed to delete email template",
			log.Error(err),
			log.String("template_id", templateID),
		)
		common.ResponseError(c, err)
		return
	}

	h.logger.Info("Email template deleted successfully",
		log.String("template_id", templateID),
	)

	common.ResponseNoContent(c, "Email template deleted successfully")
}

func (h *EmailHandler) PreviewTemplate(c *gin.Context) {
	var req domain.EmailPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind preview template request", log.Error(err))
		common.ResponseBadRequest(c, err.Error())
		return
	}

	// Get template
	template, err := h.usecase.FindTemplate(c.Request.Context(), req.TemplateCode, req.Locale)
	if err != nil {
		h.logger.Error("Failed to get template for preview",
			log.Error(err),
			log.String("template_code", string(req.TemplateCode)),
		)
		common.ResponseError(c, err)
		return
	}

	// Render template with provided data
	subject, content, err := h.templateRenderer.RenderTemplate(template, req.Data)
	if err != nil {
		h.logger.Error("Failed to render template for preview",
			log.Error(err),
			log.String("template_code", string(req.TemplateCode)),
		)
		common.ResponseError(c, domain.ErrBadRequest.WithError("failed to render template").WithWrap(err))
		return
	}

	response := &domain.EmailPreviewResponse{
		Subject: subject,
		Content: content,
		Data:    req.Data,
	}

	common.ResponseOK(c, response, "Email template preview generated successfully")
}

// Email log operations
func (h *EmailHandler) GetEmailLog(c *gin.Context) {
	emailLogID := c.Param("id")
	if emailLogID == "" {
		common.ResponseBadRequest(c, "Email log ID is required")
		return
	}

	emailLog, err := h.usecase.GetEmailLog(c.Request.Context(), emailLogID)
	if err != nil {
		h.logger.Error("Failed to get email log",
			log.Error(err),
			log.String("email_log_id", emailLogID),
		)
		common.ResponseError(c, err)
		return
	}

	common.ResponseOK(c, emailLog, "Email log retrieved successfully")
}

func (h *EmailHandler) GetEmailLogs(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &domain.EmailLogFilter{}

	if to := c.Query("to"); to != "" {
		filter.To = &to
	}
	if cc := c.Query("cc"); cc != "" {
		filter.CC = &cc
	}
	if bcc := c.Query("bcc"); bcc != "" {
		filter.BCC = &bcc
	}
	if anyRecipient := c.Query("any_recipient"); anyRecipient != "" {
		filter.AnyRecipient = &anyRecipient
	}
	if status := c.Query("status"); status != "" {
		emailStatus := domain.EmailStatus(status)
		filter.Status = &emailStatus
	}
	if provider := c.Query("provider"); provider != "" {
		emailProvider := domain.EmailProvider(provider)
		filter.Provider = &emailProvider
	}
	if template := c.Query("template"); template != "" {
		filter.Template = &template
	}
	if requestID := c.Query("request_id"); requestID != "" {
		filter.RequestID = &requestID
	}
	if searchTerm := c.Query("search"); searchTerm != "" {
		filter.SearchTerm = &searchTerm
	}

	// Parse date filters
	if sentAfterStr := c.Query("sent_after"); sentAfterStr != "" {
		if sentAfter, err := strconv.ParseInt(sentAfterStr, 10, 64); err == nil {
			filter.SentAfter = &sentAfter
		}
	}
	if sentBeforeStr := c.Query("sent_before"); sentBeforeStr != "" {
		if sentBefore, err := strconv.ParseInt(sentBeforeStr, 10, 64); err == nil {
			filter.SentBefore = &sentBefore
		}
	}
	if createdAfterStr := c.Query("created_after"); createdAfterStr != "" {
		if createdAfter, err := strconv.ParseInt(createdAfterStr, 10, 64); err == nil {
			filter.CreatedAfter = &createdAfter
		}
	}
	if createdBeforeStr := c.Query("created_before"); createdBeforeStr != "" {
		if createdBefore, err := strconv.ParseInt(createdBeforeStr, 10, 64); err == nil {
			filter.CreatedBefore = &createdBefore
		}
	}

	// Parse additional filters
	if minRetryStr := c.Query("min_retry_count"); minRetryStr != "" {
		if minRetry, err := strconv.Atoi(minRetryStr); err == nil {
			filter.MinRetryCount = &minRetry
		}
	}
	if maxRetryStr := c.Query("max_retry_count"); maxRetryStr != "" {
		if maxRetry, err := strconv.Atoi(maxRetryStr); err == nil {
			filter.MaxRetryCount = &maxRetry
		}
	}
	if minRecipientsStr := c.Query("min_recipients"); minRecipientsStr != "" {
		if minRecipients, err := strconv.Atoi(minRecipientsStr); err == nil {
			filter.MinRecipients = &minRecipients
		}
	}
	if maxRecipientsStr := c.Query("max_recipients"); maxRecipientsStr != "" {
		if maxRecipients, err := strconv.Atoi(maxRecipientsStr); err == nil {
			filter.MaxRecipients = &maxRecipients
		}
	}
	if contentType := c.Query("content_type"); contentType != "" {
		filter.ContentType = &contentType
	}
	if hasAttachmentsStr := c.Query("has_attachments"); hasAttachmentsStr != "" {
		if hasAttachments, err := strconv.ParseBool(hasAttachmentsStr); err == nil {
			filter.HasAttachments = &hasAttachments
		}
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	option := &domain.FindPageOption{
		Page:    page,
		PerPage: perPage,
	}

	emailLogs, pagination, err := h.usecase.GetEmailLogs(c.Request.Context(), filter, option)
	if err != nil {
		h.logger.Error("Failed to get email logs", log.Error(err))
		common.ResponseError(c, err)
		return
	}

	response := map[string]interface{}{
		"email_logs": emailLogs,
		"pagination": pagination,
	}

	common.ResponseOK(c, response, "Email logs retrieved successfully")
}

func (h *EmailHandler) GetEmailStats(c *gin.Context) {
	// Parse query parameters for stats filtering
	filter := &domain.EmailStatsFilter{}

	if provider := c.Query("provider"); provider != "" {
		emailProvider := domain.EmailProvider(provider)
		filter.Provider = &emailProvider
	}
	if template := c.Query("template"); template != "" {
		filter.Template = &template
	}
	if status := c.Query("status"); status != "" {
		emailStatus := domain.EmailStatus(status)
		filter.Status = &emailStatus
	}
	if groupBy := c.Query("group_by"); groupBy != "" {
		filter.GroupBy = groupBy
	}
	if includeTotalStr := c.Query("include_total"); includeTotalStr != "" {
		if includeTotal, err := strconv.ParseBool(includeTotalStr); err == nil {
			filter.IncludeTotal = includeTotal
		}
	}

	// Parse date range
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if dateFrom, err := strconv.ParseInt(dateFromStr, 10, 64); err == nil {
			filter.DateFrom = &dateFrom
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if dateTo, err := strconv.ParseInt(dateToStr, 10, 64); err == nil {
			filter.DateTo = &dateTo
		}
	}

	stats, err := h.usecase.GetEmailStats(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get email stats", log.Error(err))
		common.ResponseError(c, err)
		return
	}

	common.ResponseOK(c, stats, "Email statistics retrieved successfully")
}
