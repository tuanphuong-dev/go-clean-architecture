package usecase

import (
	"bytes"
	"fmt"
	"go-clean-arch/domain"
	"go-clean-arch/pkg/log"
	"html/template"
	"regexp"
	"strings"
	textTemplate "text/template"
	"time"
)

type htmlTemplateRenderer struct {
	logger log.Logger
}

func NewTemplateRenderer(logger log.Logger) TemplateRenderer {
	return &htmlTemplateRenderer{
		logger: logger,
	}
}

func (r *htmlTemplateRenderer) RenderTemplate(tmpl *domain.EmailTemplate, data map[string]interface{}) (subject, content string, err error) {
	// Add default template functions
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"now":   func() string { return time.Now().Format("2006-01-02 15:04:05") },
	}

	// Add current_time to data if not provided
	if data == nil {
		data = make(map[string]interface{})
	}
	if _, exists := data["current_time"]; !exists {
		data["current_time"] = time.Now().Format("2006-01-02 15:04:05")
	}

	// Render subject
	subjectTmpl, err := textTemplate.New("subject").Funcs(template.FuncMap(funcMap)).Parse(tmpl.Subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render subject: %w", err)
	}
	subject = subjectBuf.String()

	// Render content
	contentTmpl, err := template.New("content").Funcs(funcMap).Parse(tmpl.Content)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse content template: %w", err)
	}

	var contentBuf bytes.Buffer
	if err := contentTmpl.Execute(&contentBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render content: %w", err)
	}
	content = contentBuf.String()

	r.logger.Debug("Template rendered successfully",
		log.String("template_id", tmpl.ID),
		log.Int("subject_length", len(subject)),
		log.Int("content_length", len(content)),
	)

	return subject, content, nil
}

func (r *htmlTemplateRenderer) ValidateTemplate(tmpl *domain.EmailTemplate) error {
	// Add default template functions for validation
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"now":   func() string { return time.Now().Format("2006-01-02 15:04:05") },
	}

	// Validate subject template
	_, err := textTemplate.New("subject").Funcs(template.FuncMap(funcMap)).Parse(tmpl.Subject)
	if err != nil {
		return fmt.Errorf("invalid subject template: %w", err)
	}

	// Validate content template
	_, err = template.New("content").Funcs(funcMap).Parse(tmpl.Content)
	if err != nil {
		return fmt.Errorf("invalid content template: %w", err)
	}

	r.logger.Debug("Template validation successful", log.Any("template_id", tmpl.ID))

	return nil
}

func (r *htmlTemplateRenderer) GetRequiredFields(tmpl *domain.EmailTemplate) ([]string, error) {
	var fields []string
	fieldSet := make(map[string]struct{})

	// Extract fields from subject
	subjectFields := r.extractTemplateFields(tmpl.Subject)
	for _, field := range subjectFields {
		if _, exists := fieldSet[field]; !exists {
			fields = append(fields, field)
			fieldSet[field] = struct{}{}
		}
	}

	// Extract fields from content
	contentFields := r.extractTemplateFields(tmpl.Content)
	for _, field := range contentFields {
		if _, exists := fieldSet[field]; !exists {
			fields = append(fields, field)
			fieldSet[field] = struct{}{}
		}
	}

	return fields, nil
}

func (r *htmlTemplateRenderer) extractTemplateFields(templateStr string) []string {
	// Regex to match {{.field_name}} patterns
	re := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	var fields []string
	for _, match := range matches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}

	return fields
}
