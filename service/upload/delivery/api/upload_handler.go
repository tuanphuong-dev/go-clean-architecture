package api

import (
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/middleware"
	"go-clean-arch/pkg/log"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	usecase     domain.UploadUsecase
	logger      log.Logger
	middlewares middleware.Middlewares
}

type UploadHandlerDeps struct {
	Usecase     domain.UploadUsecase
	Logger      log.Logger
	Middlewares middleware.Middlewares
}

func NewUploadHandler(deps *UploadHandlerDeps) *UploadHandler {
	return &UploadHandler{
		usecase:     deps.Usecase,
		logger:      deps.Logger,
		middlewares: deps.Middlewares,
	}
}

func (h *UploadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	upload := rg.Group("/upload")
	// upload.Use(h.middlewares.Authenticator())
	{
		upload.POST("", h.UploadFiles)
	}
}

func (h *UploadHandler) UploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		common.ResponseError(c, domain.ErrUploadInvalidContentType.WithWrap(err))
		return
	}

	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		common.ResponseError(c, domain.ErrUploadFilesRequired)
		return
	}

	fileWithContents, err := domain.NewFileWithContents(fileHeaders)
	if err != nil {
		common.ResponseError(c, domain.ErrUploadFilesFailed.WithWrap(err))
		return
	}

	files, err := h.usecase.UploadFiles(c.Request.Context(), fileWithContents)
	if err != nil {
		common.ResponseError(c, domain.ErrUploadFilesFailed.WithWrap(err))
		return
	}

	common.ResponseCreated(c, files, "Files uploaded successfully")
}
