package repository

import (
	"context"
	"go-clean-arch/database"
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

type EmailLogRepository struct {
	db         *gorm.DB
	sqlHandler *database.SQLHandler[domain.EmailLog, domain.EmailLogFilter]
}

func NewEmailLogRepository(db *gorm.DB) *EmailLogRepository {
	sqlHandler := database.NewSQLHandler[domain.EmailLog](db, applyEmailLogFilter)
	return &EmailLogRepository{
		db:         db,
		sqlHandler: sqlHandler,
	}
}

func applyEmailLogFilter(qb *gorm.DB, filter *domain.EmailLogFilter) *gorm.DB {
	if filter == nil {
		return qb
	}

	if filter.ID != nil {
		qb = qb.Where("id = ?", *filter.ID)
	}

	// Filter by TO recipients - search in JSON array
	if filter.To != nil {
		qb = qb.Where("to::text ILIKE ?", "%\""+*filter.To+"\"%")
	}

	// Filter by CC recipients - search in JSON array
	if filter.CC != nil {
		qb = qb.Where("cc::text ILIKE ?", "%\""+*filter.CC+"\"%")
	}

	// Filter by BCC recipients - search in JSON array
	if filter.BCC != nil {
		qb = qb.Where("bcc::text ILIKE ?", "%\""+*filter.BCC+"\"%")
	}

	// Filter across all recipient types
	if filter.AnyRecipient != nil {
		recipient := *filter.AnyRecipient
		qb = qb.Where("(to::text ILIKE ? OR cc::text ILIKE ? OR bcc::text ILIKE ?)",
			"%\""+recipient+"\"%", "%\""+recipient+"\"%", "%\""+recipient+"\"%")
	}

	if filter.Status != nil {
		qb = qb.Where("status = ?", *filter.Status)
	}

	if filter.Provider != nil {
		qb = qb.Where("provider = ?", *filter.Provider)
	}

	if filter.Template != nil {
		qb = qb.Where("template = ?", *filter.Template)
	}

	if filter.RequestID != nil {
		qb = qb.Where("request_id = ?", *filter.RequestID)
	}

	// Date filters for sent_at
	if filter.SentAfter != nil {
		qb = qb.Where("sent_at >= ?", *filter.SentAfter)
	}
	if filter.SentBefore != nil {
		qb = qb.Where("sent_at <= ?", *filter.SentBefore)
	}

	// Date filters for created_at
	if filter.CreatedAfter != nil {
		qb = qb.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		qb = qb.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Retry count filters
	if filter.MinRetryCount != nil {
		qb = qb.Where("retry_count >= ?", *filter.MinRetryCount)
	}
	if filter.MaxRetryCount != nil {
		qb = qb.Where("retry_count <= ?", *filter.MaxRetryCount)
	}

	// Recipient count filters
	if filter.MinRecipients != nil {
		qb = qb.Where("total_recipients >= ?", *filter.MinRecipients)
	}
	if filter.MaxRecipients != nil {
		qb = qb.Where("total_recipients <= ?", *filter.MaxRecipients)
	}

	// Content type filter
	if filter.ContentType != nil {
		qb = qb.Where("content_type = ?", *filter.ContentType)
	}

	// Attachment filter
	if filter.HasAttachments != nil {
		if *filter.HasAttachments {
			qb = qb.Where("attachment_count > 0")
		} else {
			qb = qb.Where("attachment_count = 0")
		}
	}

	// Message size filters
	if filter.MinMessageSize != nil {
		qb = qb.Where("message_size >= ?", *filter.MinMessageSize)
	}
	if filter.MaxMessageSize != nil {
		qb = qb.Where("message_size <= ?", *filter.MaxMessageSize)
	}

	// Search functionality
	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		searchFields := filter.SearchFields
		if len(searchFields) == 0 {
			searchFields = []string{"to", "cc", "bcc", "subject", "template", "content"}
		}

		searchQuery := ""
		searchValues := []interface{}{}
		for i, field := range searchFields {
			if i > 0 {
				searchQuery += " OR "
			}

			switch field {
			case "to", "cc", "bcc":
				// Search in JSON arrays
				searchQuery += field + "::text ILIKE ?"
				searchValues = append(searchValues, "%"+*filter.SearchTerm+"%")
			case "subject", "template", "content":
				// Search in regular text fields
				searchQuery += field + " ILIKE ?"
				searchValues = append(searchValues, "%"+*filter.SearchTerm+"%")
			}
		}

		if searchQuery != "" {
			qb = qb.Where("("+searchQuery+")", searchValues...)
		}
	}

	// Default: exclude soft deleted records
	if filter.IncludeDeleted == nil || !*filter.IncludeDeleted {
		qb = qb.Where("deleted_at = 0")
	}

	return qb
}

func (r *EmailLogRepository) Create(ctx context.Context, emailLog *domain.EmailLog) error {
	return r.sqlHandler.Create(ctx, emailLog)
}

func (r *EmailLogRepository) FindByID(ctx context.Context, emailLogID string, option *domain.FindOneOption) (*domain.EmailLog, error) {
	return r.sqlHandler.FindByID(ctx, emailLogID, option)
}

func (r *EmailLogRepository) FindOne(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindOneOption) (*domain.EmailLog, error) {
	return r.sqlHandler.FindOne(ctx, filter, option)
}

func (r *EmailLogRepository) FindMany(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindManyOption) ([]*domain.EmailLog, error) {
	return r.sqlHandler.FindMany(ctx, filter, option)
}

func (r *EmailLogRepository) FindPage(ctx context.Context, filter *domain.EmailLogFilter, option *domain.FindPageOption) ([]*domain.EmailLog, *domain.Pagination, error) {
	return r.sqlHandler.FindPage(ctx, filter, option)
}

func (r *EmailLogRepository) Update(ctx context.Context, emailLog *domain.EmailLog) error {
	return r.sqlHandler.Update(ctx, emailLog)
}

func (r *EmailLogRepository) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return r.sqlHandler.UpdateFields(ctx, id, fields)
}

func (r *EmailLogRepository) Delete(ctx context.Context, emailLogID string) error {
	return r.sqlHandler.DeleteByID(ctx, emailLogID)
}

func (r *EmailLogRepository) Count(ctx context.Context, filter *domain.EmailLogFilter) (int64, error) {
	return r.sqlHandler.Count(ctx, filter)
}

func (r *EmailLogRepository) GetStats(ctx context.Context, filter *domain.EmailStatsFilter) (*domain.EmailStats, error) {
	qb := r.db.WithContext(ctx)

	// Base query for email logs
	query := qb.Model(&domain.EmailLog{})

	// Apply filters
	if filter.Provider != nil {
		query = query.Where("provider = ?", *filter.Provider)
	}
	if filter.Template != nil {
		query = query.Where("template = ?", *filter.Template)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.DateFrom != nil {
		query = query.Where("sent_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("sent_at <= ?", *filter.DateTo)
	}

	// Base statistics
	var totalSent, totalSuccess, totalFailed, totalPending int64
	var avgRetryCount float64

	// Get basic counts
	query.Count(&totalSent)

	query.Where("status = ?", domain.EmailStatusSuccess).Count(&totalSuccess)
	query.Where("status = ?", domain.EmailStatusFailed).Count(&totalFailed)
	query.Where("status = ?", domain.EmailStatusPending).Count(&totalPending)

	// Calculate average retry count
	if totalSent > 0 {
		var sumRetryCount int64
		query.Select("COALESCE(SUM(retry_count), 0)").Scan(&sumRetryCount)
		avgRetryCount = float64(sumRetryCount) / float64(totalSent)
	}

	// Calculate rates
	var successRate, failureRate float64
	if totalSent > 0 {
		successRate = float64(totalSuccess) / float64(totalSent)
		failureRate = float64(totalFailed) / float64(totalSent)
	}

	stats := &domain.EmailStats{
		TotalSent:     totalSent,
		TotalSuccess:  totalSuccess,
		TotalFailed:   totalFailed,
		TotalPending:  totalPending,
		SuccessRate:   successRate,
		FailureRate:   failureRate,
		AvgRetryCount: avgRetryCount,
	}

	// Add grouped stats if requested
	if filter.GroupBy != "" {
		groupedStats, err := r.getGroupedStats(ctx, query, filter.GroupBy)
		if err != nil {
			return nil, err
		}
		stats.GroupedStats = groupedStats
	}

	// Add provider stats
	if filter.IncludeTotal {
		providerStats, err := r.getProviderStats(ctx, query)
		if err != nil {
			return nil, err
		}
		stats.ProviderStats = providerStats

		templateStats, err := r.getTemplateStats(ctx, query)
		if err != nil {
			return nil, err
		}
		stats.TemplateStats = templateStats
	}

	// Add date range
	if filter.DateFrom != nil && filter.DateTo != nil {
		stats.DateRange = &domain.EmailStatsDateRange{
			From: *filter.DateFrom,
			To:   *filter.DateTo,
		}
	}

	return stats, nil
}

func (r *EmailLogRepository) getGroupedStats(_ context.Context, baseQuery *gorm.DB, groupBy string) ([]*domain.EmailStatsGroup, error) {
	var results []map[string]any

	var selectClause, groupClause string

	switch groupBy {
	case "day":
		selectClause = "DATE(TO_TIMESTAMP(sent_at/1000)) as group_key, COUNT(*) as count, " +
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, " +
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed, " +
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending"
		groupClause = "DATE(TO_TIMESTAMP(sent_at/1000))"
	case "provider":
		selectClause = "provider as group_key, COUNT(*) as count, " +
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, " +
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed, " +
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending"
		groupClause = "provider"
	case "template":
		selectClause = "template as group_key, COUNT(*) as count, " +
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, " +
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed, " +
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending"
		groupClause = "template"
	case "status":
		selectClause = "status as group_key, COUNT(*) as count, " +
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, " +
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed, " +
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending"
		groupClause = "status"
	default:
		return nil, nil
	}

	err := baseQuery.Select(selectClause).Group(groupClause).Find(&results).Error
	if err != nil {
		return nil, err
	}

	var groupedStats []*domain.EmailStatsGroup
	for _, result := range results {
		group := &domain.EmailStatsGroup{
			Key:     getString(result["group_key"]),
			Label:   getString(result["group_key"]),
			Count:   getInt64(result["count"]),
			Sent:    getInt64(result["count"]),
			Success: getInt64(result["success"]),
			Failed:  getInt64(result["failed"]),
			Pending: getInt64(result["pending"]),
		}

		if group.Sent > 0 {
			group.Rate = float64(group.Success) / float64(group.Sent)
		}

		groupedStats = append(groupedStats, group)
	}

	return groupedStats, nil
}

func (r *EmailLogRepository) getProviderStats(_ context.Context, baseQuery *gorm.DB) ([]*domain.EmailProviderStats, error) {
	var results []map[string]any

	selectClause := "provider, COUNT(*) as total_sent, " +
		"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as total_success, " +
		"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as total_failed, " +
		"AVG(retry_count) as avg_retry_count"

	err := baseQuery.Select(selectClause).Group("provider").Find(&results).Error
	if err != nil {
		return nil, err
	}

	var providerStats []*domain.EmailProviderStats
	for _, result := range results {
		providerStr := getString(result["provider"])
		totalSent := getInt64(result["total_sent"])
		totalSuccess := getInt64(result["total_success"])

		stats := &domain.EmailProviderStats{
			Provider:      domain.EmailProvider(providerStr),
			TotalSent:     totalSent,
			TotalSuccess:  totalSuccess,
			TotalFailed:   getInt64(result["total_failed"]),
			AvgRetryCount: getFloat64(result["avg_retry_count"]),
		}

		if totalSent > 0 {
			stats.SuccessRate = float64(totalSuccess) / float64(totalSent)
		}

		providerStats = append(providerStats, stats)
	}

	return providerStats, nil
}

func (r *EmailLogRepository) getTemplateStats(_ context.Context, baseQuery *gorm.DB) ([]*domain.EmailTemplateStats, error) {
	var results []map[string]any

	selectClause := "template, COUNT(*) as total_sent, " +
		"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as total_success, " +
		"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as total_failed, " +
		"MAX(sent_at) as last_used"

	err := baseQuery.Select(selectClause).Group("template").Find(&results).Error
	if err != nil {
		return nil, err
	}

	var templateStats []*domain.EmailTemplateStats
	for _, result := range results {
		template := getString(result["template"])
		totalSent := getInt64(result["total_sent"])
		totalSuccess := getInt64(result["total_success"])

		stats := &domain.EmailTemplateStats{
			Template:     template,
			Code:         domain.EmailCode(template), // Assume template name matches code
			TotalSent:    totalSent,
			TotalSuccess: totalSuccess,
			TotalFailed:  getInt64(result["total_failed"]),
			LastUsed:     getInt64(result["last_used"]),
		}

		if totalSent > 0 {
			stats.SuccessRate = float64(totalSuccess) / float64(totalSent)
		}

		templateStats = append(templateStats, stats)
	}

	return templateStats, nil
}

// Helper functions to safely convert interface{} to specific types
func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	}
	return 0
}

func getFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
