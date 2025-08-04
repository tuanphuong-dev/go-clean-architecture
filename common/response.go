package common

import (
	"errors"
	"fmt"
	"go-clean-arch/domain"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ResponseT[T any] struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Data        T      `json:"data"`
	Description string `json:"description"`
}

var logger Logger

// SetLogger sets the logger for response logging
func SetLogger(l Logger) {
	logger = l
}

func Response[T any](c *gin.Context, status int, code string, data T, desc string) {
	if status >= 400 && logger != nil {
		logger.Error("API Error",
			"status", status,
			"code", code,
			"description", desc,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)
	}

	c.AbortWithStatusJSON(status, ResponseT[T]{
		Status:      status,
		Code:        code,
		Data:        data,
		Description: desc,
	})
}

// Success responses
func ResponseOK[T any](c *gin.Context, data T, desc string) {
	Response(c, http.StatusOK, "SUCCESS", data, desc)
}

func ResponseCreated[T any](c *gin.Context, data T, desc string) {
	Response(c, http.StatusCreated, "SUCCESS", data, desc)
}

func ResponseNoContent(c *gin.Context, desc string) {
	Response[any](c, http.StatusNoContent, "SUCCESS", nil, desc)
}

func ResponseBadRequest(c *gin.Context, desc string) {
	dErr := domain.ErrBadRequest.WithWrap(errors.New(desc))
	Response[any](c, dErr.StatusCode(), dErr.IDField, dErr.DetailsField, dErr.ErrorField)
}

func ResponseForbidden(c *gin.Context, desc string) {
	dErr := domain.ErrForbidden.WithWrap(errors.New(desc))
	Response[any](c, dErr.StatusCode(), dErr.IDField, dErr.DetailsField, dErr.ErrorField)
}

func ResponseNotFound(c *gin.Context, desc string) {
	dErr := domain.ErrNotFound.WithWrap(errors.New(desc))
	Response[any](c, dErr.StatusCode(), dErr.IDField, dErr.DetailsField, dErr.ErrorField)
}

func ResponseError(c *gin.Context, err error) {
	var dErr *domain.DetailedError
	if de, ok := IsDetailError(err); ok {
		dErr = de
	} else {
		dErr = domain.ErrInternalServerError.WithWrap(err)
	}

	Response[any](c, dErr.StatusCode(), dErr.IDField, dErr.DetailsField, dErr.ErrorField)
}

func ResponseRateLimitExceeded(c *gin.Context, desc string, retryAt time.Time) {
	retryAfterSeconds := int64(0)
	retryAtISO := ""

	if !retryAt.IsZero() {
		retryAfterSeconds = int64(time.Until(retryAt).Seconds())
		if retryAfterSeconds > 0 {
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
		}
		retryAtISO = retryAt.Format(time.RFC3339)
	}

	c.JSON(http.StatusTooManyRequests, ResponseT[map[string]interface{}]{
		Status:      http.StatusTooManyRequests,
		Code:        "TOO_MANY_REQUESTS",
		Description: desc,
		Data: map[string]interface{}{
			"retry_at":            retryAtISO,
			"retry_after_seconds": retryAfterSeconds,
		},
	})
	c.Abort()
}

func ResponseTooManyRequests(c *gin.Context, message string, retryAt time.Time) {
	ResponseRateLimitExceeded(c, message, retryAt)
}
