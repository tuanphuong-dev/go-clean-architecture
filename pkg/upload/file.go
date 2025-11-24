package upload

import (
	"bytes"
	"image"
	"io"
	"mime/multipart"
	"path"
	"strings"

	"github.com/nfnt/resize"
	"github.com/samber/lo"
)

const (
	HashLength = 32
)

type File struct {
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Content []byte `json:"content"`
}

func (file *File) IsImage() bool {
	return strings.HasPrefix(file.Mime, "image/")
}

func (file *File) IsVideo() bool {
	return strings.HasPrefix(file.Mime, "video/")
}

type UploadedFileInfo struct {
	Name                 string   `json:"name"`
	Mime                 string   `json:"mime"`
	Ext                  string   `json:"ext"`
	URL                  string   `json:"url"`
	ThumbnailURL         string   `json:"thumbnail_url"`
	Width                int64    `json:"width"`
	Height               int64    `json:"height"`
	Size                 int64    `json:"size"`
	StoragePath          string   `json:"storage_path"`
	ThumbnailStoragePath string   `json:"thumbnail_storage_path"`
	Provider             Provider `json:"provider"`
}

func getExt(fileName string) string {
	return path.Ext(fileName)
}

func generateHash() string {
	return lo.RandomString(HashLength, lo.AlphanumericCharset)
}

func generateFileName(filename, hash string) string {
	return hash + "_" + strings.ReplaceAll(filename, " ", "-")
}

func generateThumbnailName(filename, hash string) string {
	return "thumb_" + hash + "_" + strings.ReplaceAll(filename, " ", "-")
}

func ParseFileHeaders2Files(fileHeaders []*multipart.FileHeader) ([]*File, error) {
	var files []*File
	for _, fileHeader := range fileHeaders {
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

		fileInfo := &File{
			Name:    fileHeader.Filename,
			Mime:    fileHeader.Header.Get("Content-Type"),
			Content: fileContent,
		}

		files = append(files, fileInfo)
	}
	return files, nil
}

func DecodeImgAndGenThumbnail(r io.Reader, thumbWidth uint, thumbHeight uint) (width int64, height int64, thumbnail image.Image, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, 0, nil, err
	}

	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return 0, 0, nil, err
	}

	thumbnail = resize.Resize(thumbWidth, thumbHeight, img, resize.Lanczos3)
	width = int64(img.Bounds().Dx())
	height = int64(img.Bounds().Dy())
	return width, height, thumbnail, nil
}
