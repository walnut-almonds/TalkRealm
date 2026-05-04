package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// FileHandler 檔案處理器
type FileHandler struct {
	fileService service.FileService
}

// NewFileHandler 建立 FileHandler
func NewFileHandler(fileService service.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

// PresignUpload 產生 Pre-signed 上傳 URL
//
//	@Summary		申請上傳 URL
//	@Description	驗證檔案類型與 quota，回傳 Minio Pre-signed PUT URL
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.PresignUploadRequest	true	"上傳申請"
//	@Success		200		{object}	service.PresignUploadResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		429		{object}	map[string]string
//	@Router			/api/v1/files/presign [post]
func (h *FileHandler) PresignUpload(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req service.PresignUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.fileService.PresignUpload(userID.(uint), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnsupportedFileType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
		case errors.Is(err, service.ErrFileTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds limit"})
		case errors.Is(err, service.ErrDailyUploadExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "daily upload count limit exceeded"})
		case errors.Is(err, service.ErrDailyBytesExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "daily upload bytes limit exceeded"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}

		return
	}

	c.JSON(http.StatusOK, resp)
}

// ConfirmUpload 確認檔案已上傳至 Minio
//
//	@Summary		確認上傳完成
//	@Description	客戶端上傳完成後呼叫，驗證物件存在並將狀態改為 active
//	@Tags			files
//	@Produce		json
//	@Param			id	path		int	true	"File ID"
//	@Success		200	{object}	model.File
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/files/{id}/confirm [post]
func (h *FileHandler) ConfirmUpload(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	file, err := h.fileService.ConfirmUpload(userID.(uint), uint(fileID))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		case errors.Is(err, service.ErrFileForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrUploadNotConfirmed):
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "file not found in storage; please upload first"},
			)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}

		return
	}

	c.JSON(http.StatusOK, file)
}

// GetFile 取得檔案 metadata
//
//	@Summary	取得檔案資訊
//	@Tags		files
//	@Produce	json
//	@Param		id	path		int	true	"File ID"
//	@Success	200	{object}	model.File
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/files/{id} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	file, err := h.fileService.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		case errors.Is(err, service.ErrFileForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}

		return
	}

	c.JSON(http.StatusOK, file)
}

// GetDownloadURL 產生下載 Pre-signed URL
//
//	@Summary	取得下載 URL
//	@Tags		files
//	@Produce	json
//	@Param		id	path		int	true	"File ID"
//	@Success	200	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/files/{id}/url [get]
func (h *FileHandler) GetDownloadURL(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	url, err := h.fileService.GetDownloadURL(userID.(uint), uint(fileID))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		case errors.Is(err, service.ErrFileForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// DeleteFile 刪除檔案
//
//	@Summary	刪除檔案
//	@Tags		files
//	@Produce	json
//	@Param		id	path		int	true	"File ID"
//	@Success	204	{object}	nil
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	if err = h.fileService.DeleteFile(userID.(uint), uint(fileID)); err != nil {
		switch {
		case errors.Is(err, service.ErrFileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		case errors.Is(err, service.ErrFileForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}

		return
	}

	c.Status(http.StatusNoContent)
}
