package client

import (
	"context"
	"encoding/json"
	"go-clean-arch/domain"
	"go-clean-arch/proto/pb"

	"google.golang.org/grpc"
)

type EmailRPCClient struct {
	client pb.EmailServiceClient
}

func NewEmailRPCClient(conn *grpc.ClientConn) *EmailRPCClient {
	return &EmailRPCClient{
		client: pb.NewEmailServiceClient(conn),
	}
}

func (c *EmailRPCClient) SendEmailWithTemplate(ctx context.Context, req *domain.SendEmailWithTemplateRequest) (*domain.EmailLog, error) {
	// Convert template data to JSON string
	var dataStr string
	if req.Data != nil {
		dataBytes, err := json.Marshal(req.Data)
		if err != nil {
			return nil, err
		}
		dataStr = string(dataBytes)
	}

	// Convert attachments
	var protoAttachments []*pb.EmailAttachment
	for _, att := range req.Attachments {
		protoAttachments = append(protoAttachments, &pb.EmailAttachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
			Inline:      att.Inline,
			ContentId:   att.ContentID,
		})
	}

	// Create RPC request
	rpcReq := &pb.SendEmailWithTemplateRequest{
		To:           req.To,
		Cc:           req.CC,
		Bcc:          req.BCC,
		TemplateCode: string(req.TemplateCode),
		Locale:       req.Locale,
		Data:         dataStr,
		Attachments:  protoAttachments,
		Headers:      req.Headers,
		Provider:     string(req.Provider),
		RequestId:    req.RequestID,
	}

	// Call RPC service
	resp, err := c.client.SendEmailWithTemplate(ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	// Convert response back to domain model
	return c.convertProtoToEmailLog(resp.EmailLog), nil
}

func (c *EmailRPCClient) convertProtoToEmailLog(protoLog *pb.EmailLog) *domain.EmailLog {
	emailLog := &domain.EmailLog{
		Subject:         protoLog.Subject,
		Content:         protoLog.Content,
		Template:        protoLog.Template,
		Status:          domain.EmailStatus(protoLog.Status),
		ErrorMsg:        protoLog.ErrorMsg,
		SentAt:          protoLog.SentAt,
		RequestID:       protoLog.RequestId,
		RetryCount:      int(protoLog.RetryCount),
		Response:        protoLog.Response,
		Provider:        domain.EmailProvider(protoLog.Provider),
		TotalRecipients: int(protoLog.TotalRecipients),
		ContentType:     protoLog.ContentType,
		AttachmentCount: int(protoLog.AttachmentCount),
		MessageSize:     protoLog.MessageSize,
	}

	// Set base model fields
	emailLog.ID = protoLog.Id
	emailLog.CreatedAt = protoLog.CreatedAt
	emailLog.UpdatedAt = protoLog.UpdatedAt
	emailLog.DeletedAt = protoLog.DeletedAt

	// Set recipients
	emailLog.SetToEmails(protoLog.To)
	emailLog.SetCCEmails(protoLog.Cc)
	emailLog.SetBCCEmails(protoLog.Bcc)

	// Parse JSON fields
	if protoLog.Data != "" {
		var data any
		if err := json.Unmarshal([]byte(protoLog.Data), &data); err == nil {
			emailLog.Data.Scan(data)
		}
	}

	if protoLog.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(protoLog.Headers), &headers); err == nil {
			emailLog.Headers.Scan(headers)
		}
	}

	return emailLog
}
