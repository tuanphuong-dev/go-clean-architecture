package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/pkg/errors"
)

/****************************
*        Upload errors      *
****************************/

var (
	ErrUploadFilesFailed = &DetailedError{
		IDField:         "UPLOAD_FILES_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to upload files",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrDeleteFilesFailed = &DetailedError{
		IDField:         "DELETE_FILES_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to delete files",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrUploadInvalidContentType = &DetailedError{
		IDField:         "UPLOAD_INVALID_CONTENT_TYPE",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "Invalid content type for upload",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrUploadFilesRequired = &DetailedError{
		IDField:         "UPLOAD_FILES_REQUIRED",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "No files provided for upload",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrAddFileLinksFailed = &DetailedError{
		IDField:         "ADD_FILE_LINKS_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to add file relations",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrGetFilesByEntitiesAndFieldFailed = &DetailedError{
		IDField:         "GET_FILES_BY_ENTITIES_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to get files by entities",
		StatusCodeField: http.StatusInternalServerError,
	}
)

/***************************************
*       Upload entities and types      *
***************************************/

type File struct {
	SQLModel
	Name         string     `json:"name" gorm:"column:name;type:varchar(255)"`
	Mime         string     `json:"mime" gorm:"column:mime;type:varchar(128)"`
	Ext          string     `json:"ext" gorm:"column:ext;type:varchar(128)"`
	URL          string     `json:"url" gorm:"column:url;type:text"`
	ThumbnailURL string     `json:"thumbnail_url" gorm:"column:thumbnail_url;type:text"`
	Width        int64      `json:"width" gorm:"column:width"`
	Height       int64      `json:"height" gorm:"column:height"`
	Size         int64      `json:"size" gorm:"column:size"`
	Props        *FileProps `json:"-" gorm:"column:props;type:jsonb"`
}

func NewFileFromRequest(fileReq *FileRequest) *File {
	if fileReq == nil || fileReq.ID == "" {
		return nil
	}
	return &File{
		SQLModel: SQLModel{
			ID: fileReq.ID,
		},
	}
}

type FileProps struct {
	Provider         string `json:"provider"`
	StoragePath      string `json:"storage_path"`
	ThumbStoragePath string `json:"thumb_storage_path"`
}

func (p FileProps) Value() (driver.Value, error) {
	val, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(val), nil
}

func (p *FileProps) Scan(input any) error {
	b, ok := input.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, p)
}

type FileFilter struct {
	ID             *string  `json:"id" form:"id"`
	IDIn           []string `json:"id_in" form:"id_in"`
	Ext            *string  `json:"ext" form:"ext"`
	IncludeDeleted *bool    `json:"include_deleted" form:"include_deleted"`
}

type FileLink struct {
	FileID      string `json:"file_id" gorm:"primaryKey"`
	RelatedID   string `json:"related_id" gorm:"primaryKey;index:idx_related_lookup,priority:1"`
	RelatedType string `json:"related_type" gorm:"primaryKey;index:idx_related_lookup,priority:2"`
	Field       string `json:"field" gorm:"primaryKey;index:idx_related_lookup,priority:3"`
	Order       int    `json:"order" gorm:"primaryKey,type:int4"`
	File        *File  `json:"file"`
}

type FileLinkFilter struct {
	FileID      *string  `json:"file_id" form:"file_id"`
	RelatedID   *string  `json:"related_id" form:"related_id"`
	RelatedIDIn []string `json:"related_id_in" form:"related_id_in"`
	RelatedType *string  `json:"related_type" form:"related_type"`
	Field       *string  `json:"field" form:"field"`
	FieldIn     []string `json:"field_in" form:"field_in"`
}

type FileWithContent struct {
	File
	Content []byte `json:"content" gorm:"-"`
}

type FileRequest struct {
	ID string `json:"id" validate:"required"`
}

func NewFileWithContents(fileHeaders []*multipart.FileHeader) ([]*FileWithContent, error) {
	fileWithContents := make([]*FileWithContent, len(fileHeaders))
	for idx, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}

		fileContent, err := func() ([]byte, error) {
			defer file.Close()
			return io.ReadAll(file)
		}()
		if err != nil {
			return nil, err
		}

		fileWithContent := &FileWithContent{
			File: File{
				Name: fileHeader.Filename,
				Mime: fileHeader.Header.Get("Content-Type"),
			},
			Content: fileContent,
		}

		fileWithContents[idx] = fileWithContent
	}
	return fileWithContents, nil
}

/***************************************
*  Upload usecase interfaces and types  *
****************************************/

type UploadUsecase interface {
	UploadFiles(ctx context.Context, files []*FileWithContent) ([]*File, error)
	DeleteFiles(ctx context.Context, files []*File) error
	FindManyFiles(ctx context.Context, filter *FileFilter, option *FindManyOption) ([]*File, error)
	AddFileLinks(ctx context.Context, entityType string, entityID string, fieldFiles map[string][]*File) (map[string]*File, error)
	GetFilesByEntitiesAndField(ctx context.Context, entityType string, entitiesID []string, fieldName string) (map[string][]*File, error)
	GetFilesByEntities(ctx context.Context, entityType string, entitiesID []string) (map[string]map[string][]*File, error)
}
