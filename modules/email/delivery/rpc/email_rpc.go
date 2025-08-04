package rpc

import (
	"context"
	"encoding/json"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/log"
	"go-clean-arch/proto/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EmailRPCServer struct {
	pb.UnimplementedEmailServiceServer
	usecase domain.EmailUsecase
	logger  log.Logger
}

func NewEmailRPCServer(usecase domain.EmailUsecase, logger log.Logger) *EmailRPCServer {
	return &EmailRPCServer{
		usecase: usecase,
		logger:  logger,
	}
}

// Email sending operations
func (s *EmailRPCServer) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	s.logger.Debug("RPC SendEmail called", log.Any("to", req.To), log.String("subject", req.Subject))

	// Convert attachments
	var attachments []*domain.EmailAttachment
	for _, att := range req.Attachments {
		attachments = append(attachments, &domain.EmailAttachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
			Inline:      att.Inline,
			ContentID:   att.ContentId,
		})
	}

	// Create domain request
	domainReq := &domain.SendEmailRequest{
		To:          req.To,
		CC:          req.Cc,
		BCC:         req.Bcc,
		Subject:     req.Subject,
		Content:     req.Content,
		ContentType: req.ContentType,
		Attachments: attachments,
		Headers:     req.Headers,
		Provider:    domain.EmailProvider(req.Provider),
		RequestID:   req.RequestId,
	}

	emailLog, err := s.usecase.SendEmail(ctx, domainReq)
	if err != nil {
		s.logger.Error("Failed to send email", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &pb.SendEmailResponse{
		EmailLog: s.convertEmailLogToProto(emailLog),
	}, nil
}

func (s *EmailRPCServer) SendEmailWithTemplate(ctx context.Context, req *pb.SendEmailWithTemplateRequest) (*pb.SendEmailWithTemplateResponse, error) {
	s.logger.Debug("RPC SendEmailWithTemplate called",
		log.String("template_code", req.TemplateCode),
		log.Any("to", req.To),
	)

	// Parse template data
	var templateData map[string]interface{}
	if req.Data != "" {
		if err := json.Unmarshal([]byte(req.Data), &templateData); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid template data: %v", err)
		}
	}

	// Convert attachments
	var attachments []*domain.EmailAttachment
	for _, att := range req.Attachments {
		attachments = append(attachments, &domain.EmailAttachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
			Inline:      att.Inline,
			ContentID:   att.ContentId,
		})
	}

	// Create domain request
	domainReq := &domain.SendEmailWithTemplateRequest{
		To:           req.To,
		CC:           req.Cc,
		BCC:          req.Bcc,
		TemplateCode: domain.EmailCode(req.TemplateCode),
		Locale:       req.Locale,
		Data:         templateData,
		Attachments:  attachments,
		Headers:      req.Headers,
		Provider:     domain.EmailProvider(req.Provider),
		RequestID:    req.RequestId,
	}

	emailLog, err := s.usecase.SendEmailWithTemplate(ctx, domainReq)
	if err != nil {
		s.logger.Error("Failed to send template email", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to send template email: %v", err)
	}

	return &pb.SendEmailWithTemplateResponse{
		EmailLog: s.convertEmailLogToProto(emailLog),
	}, nil
}

func (s *EmailRPCServer) SendBulkEmail(ctx context.Context, req *pb.SendBulkEmailRequest) (*pb.SendBulkEmailResponse, error) {
	s.logger.Debug("RPC SendBulkEmail called",
		log.String("template_code", req.TemplateCode),
		log.Int("recipient_count", len(req.Recipients)),
	)

	// Convert recipients
	var recipients []*domain.BulkEmailRecipient
	for _, r := range req.Recipients {
		var recipientData map[string]interface{}
		if r.Data != "" {
			if err := json.Unmarshal([]byte(r.Data), &recipientData); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid recipient data: %v", err)
			}
		}

		recipients = append(recipients, &domain.BulkEmailRecipient{
			To:   r.To,
			Data: recipientData,
		})
	}

	// Create domain request
	domainReq := &domain.SendBulkEmailRequest{
		Recipients:   recipients,
		TemplateCode: domain.EmailCode(req.TemplateCode),
		Locale:       req.Locale,
		Provider:     domain.EmailProvider(req.Provider),
		RequestID:    req.RequestId,
	}

	emailLogs, err := s.usecase.SendBulkEmail(ctx, domainReq)
	if err != nil {
		s.logger.Error("Failed to send bulk email", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to send bulk email: %v", err)
	}

	// Convert email logs
	var protoLogs []*pb.EmailLog
	for _, log := range emailLogs {
		protoLogs = append(protoLogs, s.convertEmailLogToProto(log))
	}

	return &pb.SendBulkEmailResponse{
		EmailLogs: protoLogs,
	}, nil
}

func (s *EmailRPCServer) ResendEmail(ctx context.Context, req *pb.ResendEmailRequest) (*pb.ResendEmailResponse, error) {
	s.logger.Debug("RPC ResendEmail called", log.String("email_log_id", req.EmailLogId))

	emailLog, err := s.usecase.ResendEmail(ctx, req.EmailLogId)
	if err != nil {
		s.logger.Error("Failed to resend email", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to resend email: %v", err)
	}

	return &pb.ResendEmailResponse{
		EmailLog: s.convertEmailLogToProto(emailLog),
	}, nil
}

// Email template operations
func (s *EmailRPCServer) CreateEmailTemplate(ctx context.Context, req *pb.CreateEmailTemplateRequest) (*pb.CreateEmailTemplateResponse, error) {
	s.logger.Debug("RPC CreateEmailTemplate called",
		log.String("code", req.Code),
		log.String("name", req.Name),
	)

	// Create domain request
	domainReq := &domain.CreateEmailTemplateRequest{
		Code:        domain.EmailCode(req.Code),
		Name:        req.Name,
		Subject:     req.Subject,
		Content:     req.Content,
		Description: req.Description,
		Locale:      req.Locale,
	}

	if req.IsActive != nil {
		isActive := *req.IsActive
		domainReq.IsActive = &isActive
	}

	template, err := s.usecase.CreateTemplate(ctx, domainReq)
	if err != nil {
		s.logger.Error("Failed to create email template", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create email template: %v", err)
	}

	return &pb.CreateEmailTemplateResponse{
		Template: s.convertEmailTemplateToProto(template),
	}, nil
}

func (s *EmailRPCServer) GetEmailTemplate(ctx context.Context, req *pb.GetEmailTemplateRequest) (*pb.GetEmailTemplateResponse, error) {
	s.logger.Debug("RPC GetEmailTemplate called", log.String("template_id", req.TemplateId))

	template, err := s.usecase.FindTemplateByID(ctx, req.TemplateId)
	if err != nil {
		s.logger.Error("Failed to get email template", log.Error(err))
		return nil, status.Errorf(codes.NotFound, "email template not found: %v", err)
	}

	return &pb.GetEmailTemplateResponse{
		Template: s.convertEmailTemplateToProto(template),
	}, nil
}

func (s *EmailRPCServer) GetEmailTemplateByCode(ctx context.Context, req *pb.GetEmailTemplateByCodeRequest) (*pb.GetEmailTemplateByCodeResponse, error) {
	s.logger.Debug("RPC GetEmailTemplateByCode called",
		log.String("code", req.Code),
		log.String("locale", req.Locale),
	)

	template, err := s.usecase.FindTemplate(ctx, domain.EmailCode(req.Code), req.Locale)
	if err != nil {
		s.logger.Error("Failed to get email template by code", log.Error(err))
		return nil, status.Errorf(codes.NotFound, "email template not found: %v", err)
	}

	return &pb.GetEmailTemplateByCodeResponse{
		Template: s.convertEmailTemplateToProto(template),
	}, nil
}

func (s *EmailRPCServer) UpdateEmailTemplate(ctx context.Context, req *pb.UpdateEmailTemplateRequest) (*pb.UpdateEmailTemplateResponse, error) {
	s.logger.Debug("RPC UpdateEmailTemplate called", log.String("template_id", req.TemplateId))

	// Create domain request
	domainReq := &domain.UpdateEmailTemplateRequest{}

	if req.Name != nil {
		domainReq.Name = req.Name
	}
	if req.Subject != nil {
		domainReq.Subject = req.Subject
	}
	if req.Content != nil {
		domainReq.Content = req.Content
	}
	if req.Description != nil {
		domainReq.Description = req.Description
	}
	if req.IsActive != nil {
		domainReq.IsActive = req.IsActive
	}

	template, err := s.usecase.UpdateTemplate(ctx, req.TemplateId, domainReq)
	if err != nil {
		s.logger.Error("Failed to update email template", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update email template: %v", err)
	}

	return &pb.UpdateEmailTemplateResponse{
		Template: s.convertEmailTemplateToProto(template),
	}, nil
}

func (s *EmailRPCServer) DeleteEmailTemplate(ctx context.Context, req *pb.DeleteEmailTemplateRequest) (*pb.DeleteEmailTemplateResponse, error) {
	s.logger.Debug("RPC DeleteEmailTemplate called", log.String("template_id", req.TemplateId))

	err := s.usecase.DeleteTemplate(ctx, req.TemplateId)
	if err != nil {
		s.logger.Error("Failed to delete email template", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete email template: %v", err)
	}

	return &pb.DeleteEmailTemplateResponse{
		Success: true,
	}, nil
}

func (s *EmailRPCServer) ListEmailTemplates(ctx context.Context, req *pb.ListEmailTemplatesRequest) (*pb.ListEmailTemplatesResponse, error) {
	s.logger.Debug("RPC ListEmailTemplates called")

	// Create filter
	filter := &domain.EmailTemplateFilter{}
	if req.Code != nil {
		code := domain.EmailCode(*req.Code)
		filter.Code = &code
	}
	if req.Locale != nil {
		filter.Locale = req.Locale
	}
	if req.IsActive != nil {
		filter.IsActive = req.IsActive
	}
	if req.Search != nil {
		filter.SearchTerm = req.Search
	}

	// Create pagination option
	option := &domain.FindPageOption{
		Page:    int(req.Page),
		PerPage: int(req.PerPage),
	}

	templates, pagination, err := s.usecase.FindPageTemplates(ctx, filter, option)
	if err != nil {
		s.logger.Error("Failed to list email templates", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list email templates: %v", err)
	}

	// Convert templates
	var protoTemplates []*pb.EmailTemplate
	for _, template := range templates {
		protoTemplates = append(protoTemplates, s.convertEmailTemplateToProto(template))
	}

	return &pb.ListEmailTemplatesResponse{
		Templates:  protoTemplates,
		TotalItems: int32(pagination.TotalItems),
		Page:       int32(pagination.Page),
		PerPage:    int32(pagination.PerPage),
		TotalPages: int32(pagination.TotalPages),
	}, nil
}

// Email log operations
func (s *EmailRPCServer) GetEmailLog(ctx context.Context, req *pb.GetEmailLogRequest) (*pb.GetEmailLogResponse, error) {
	s.logger.Debug("RPC GetEmailLog called", log.String("email_log_id", req.EmailLogId))

	emailLog, err := s.usecase.GetEmailLog(ctx, req.EmailLogId)
	if err != nil {
		s.logger.Error("Failed to get email log", log.Error(err))
		return nil, status.Errorf(codes.NotFound, "email log not found: %v", err)
	}

	return &pb.GetEmailLogResponse{
		EmailLog: s.convertEmailLogToProto(emailLog),
	}, nil
}

func (s *EmailRPCServer) ListEmailLogs(ctx context.Context, req *pb.ListEmailLogsRequest) (*pb.ListEmailLogsResponse, error) {
	s.logger.Debug("RPC ListEmailLogs called")

	// Create filter
	filter := &domain.EmailLogFilter{}
	if req.Status != nil {
		status := domain.EmailStatus(*req.Status)
		filter.Status = &status
	}
	if req.Provider != nil {
		provider := domain.EmailProvider(*req.Provider)
		filter.Provider = &provider
	}
	if req.Template != nil {
		filter.Template = req.Template
	}
	if req.AnyRecipient != nil {
		filter.AnyRecipient = req.AnyRecipient
	}
	if req.Search != nil {
		filter.SearchTerm = req.Search
	}
	if req.SentAfter != nil {
		filter.SentAfter = req.SentAfter
	}
	if req.SentBefore != nil {
		filter.SentBefore = req.SentBefore
	}

	// Create pagination option
	option := &domain.FindPageOption{
		Page:    int(req.Page),
		PerPage: int(req.PerPage),
	}

	emailLogs, pagination, err := s.usecase.GetEmailLogs(ctx, filter, option)
	if err != nil {
		s.logger.Error("Failed to list email logs", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list email logs: %v", err)
	}

	// Convert email logs
	var protoLogs []*pb.EmailLog
	for _, log := range emailLogs {
		protoLogs = append(protoLogs, s.convertEmailLogToProto(log))
	}

	return &pb.ListEmailLogsResponse{
		EmailLogs:  protoLogs,
		TotalItems: int32(pagination.TotalItems),
		Page:       int32(pagination.Page),
		PerPage:    int32(pagination.PerPage),
		TotalPages: int32(pagination.TotalPages),
	}, nil
}

func (s *EmailRPCServer) GetEmailStats(ctx context.Context, req *pb.GetEmailStatsRequest) (*pb.GetEmailStatsResponse, error) {
	s.logger.Debug("RPC GetEmailStats called")

	// Create filter
	filter := &domain.EmailStatsFilter{}
	if req.Provider != nil {
		provider := domain.EmailProvider(*req.Provider)
		filter.Provider = &provider
	}
	if req.Template != nil {
		filter.Template = req.Template
	}
	if req.Status != nil {
		status := domain.EmailStatus(*req.Status)
		filter.Status = &status
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}
	if req.GroupBy != nil {
		filter.GroupBy = *req.GroupBy
	}
	if req.IncludeTotal != nil {
		filter.IncludeTotal = *req.IncludeTotal
	}

	stats, err := s.usecase.GetEmailStats(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get email stats", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get email stats: %v", err)
	}

	return &pb.GetEmailStatsResponse{
		Stats: s.convertEmailStatsToProto(stats),
	}, nil
}

// Helper conversion methods
func (s *EmailRPCServer) convertEmailLogToProto(emailLog *domain.EmailLog) *pb.EmailLog {
	dataStr := ""
	if emailLog.Data != nil {
		if dataBytes, err := json.Marshal(emailLog.Data); err == nil {
			dataStr = string(dataBytes)
		}
	}

	headersStr := ""
	if emailLog.Headers != nil {
		if headersBytes, err := json.Marshal(emailLog.Headers); err == nil {
			headersStr = string(headersBytes)
		}
	}

	return &pb.EmailLog{
		Id:              emailLog.ID,
		To:              emailLog.GetToEmails(),
		Cc:              emailLog.GetCCEmails(),
		Bcc:             emailLog.GetBCCEmails(),
		Subject:         emailLog.Subject,
		Content:         emailLog.Content,
		Template:        emailLog.Template,
		Data:            dataStr,
		Status:          string(emailLog.Status),
		ErrorMsg:        emailLog.ErrorMsg,
		SentAt:          emailLog.SentAt,
		RequestId:       emailLog.RequestID,
		RetryCount:      int32(emailLog.RetryCount),
		Headers:         headersStr,
		Response:        emailLog.Response,
		Provider:        string(emailLog.Provider),
		TotalRecipients: int32(emailLog.TotalRecipients),
		ContentType:     emailLog.ContentType,
		AttachmentCount: int32(emailLog.AttachmentCount),
		MessageSize:     emailLog.MessageSize,
		CreatedAt:       emailLog.CreatedAt,
		UpdatedAt:       emailLog.UpdatedAt,
		DeletedAt:       emailLog.DeletedAt,
	}
}

func (s *EmailRPCServer) convertEmailTemplateToProto(template *domain.EmailTemplate) *pb.EmailTemplate {
	return &pb.EmailTemplate{
		Id:          template.ID,
		Code:        string(template.Code),
		Name:        template.Name,
		Subject:     template.Subject,
		Content:     template.Content,
		IsActive:    template.IsActive,
		Description: template.Description,
		Locale:      template.Locale,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
		DeletedAt:   template.DeletedAt,
	}
}

func (s *EmailRPCServer) convertEmailStatsToProto(stats *domain.EmailStats) *pb.EmailStats {
	protoStats := &pb.EmailStats{
		TotalSent:     stats.TotalSent,
		TotalSuccess:  stats.TotalSuccess,
		TotalFailed:   stats.TotalFailed,
		TotalPending:  stats.TotalPending,
		SuccessRate:   stats.SuccessRate,
		FailureRate:   stats.FailureRate,
		AvgRetryCount: stats.AvgRetryCount,
	}

	// Convert grouped stats
	for _, group := range stats.GroupedStats {
		protoStats.GroupedStats = append(protoStats.GroupedStats, &pb.EmailStatsGroup{
			Key:     group.Key,
			Label:   group.Label,
			Count:   group.Count,
			Sent:    group.Sent,
			Success: group.Success,
			Failed:  group.Failed,
			Pending: group.Pending,
			Rate:    group.Rate,
		})
	}

	// Convert provider stats
	for _, provider := range stats.ProviderStats {
		protoStats.ProviderStats = append(protoStats.ProviderStats, &pb.EmailProviderStats{
			Provider:      string(provider.Provider),
			TotalSent:     provider.TotalSent,
			TotalSuccess:  provider.TotalSuccess,
			TotalFailed:   provider.TotalFailed,
			SuccessRate:   provider.SuccessRate,
			AvgRetryCount: provider.AvgRetryCount,
		})
	}

	// Convert template stats
	for _, template := range stats.TemplateStats {
		protoStats.TemplateStats = append(protoStats.TemplateStats, &pb.EmailTemplateStats{
			Template:     template.Template,
			Code:         string(template.Code),
			TotalSent:    template.TotalSent,
			TotalSuccess: template.TotalSuccess,
			TotalFailed:  template.TotalFailed,
			SuccessRate:  template.SuccessRate,
			LastUsed:     template.LastUsed,
		})
	}

	// Convert date range
	if stats.DateRange != nil {
		protoStats.DateRange = &pb.EmailStatsDateRange{
			From: stats.DateRange.From,
			To:   stats.DateRange.To,
		}
	}

	return protoStats
}
