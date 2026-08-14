package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/storage"
	"gorm.io/gorm"
)

// 允許上傳的 MIME type 前綴 / 副檔名
var allowedExtensions = map[string]bool{
	// 圖片
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true,
	// 文件
	".pdf": true, ".txt": true, ".md": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	// 壓縮
	".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
	// 媒體
	".mp4": true, ".webm": true, ".mp3": true, ".wav": true, ".ogg": true,
}

// ErrFileTooLarge 檔案超過大小限制
var ErrFileTooLarge = errors.New("file size exceeds limit")

// ErrDailyUploadExceeded 超過每日上傳次數
var ErrDailyUploadExceeded = errors.New("daily upload count limit exceeded")

// ErrDailyBytesExceeded 超過每日上傳總量
var ErrDailyBytesExceeded = errors.New("daily upload bytes limit exceeded")

// ErrUnsupportedFileType 不支援的檔案類型
var ErrUnsupportedFileType = errors.New("unsupported file type")

// ErrFileNotFound 找不到檔案
var ErrFileNotFound = errors.New("file not found")

// ErrFileForbidden 無權限存取此檔案
var ErrFileForbidden = errors.New("forbidden")

// ErrUploadNotConfirmed 檔案尚未上傳確認
var ErrUploadNotConfirmed = errors.New("upload not confirmed in minio")

// 檔案生命週期狀態：presign 後為 pending，確認上傳完成後轉 active。
const (
	fileStatusPending = "pending"
	fileStatusActive  = "active"
)

// PresignUploadRequest 請求上傳 Pre-signed URL
type PresignUploadRequest struct {
	Filename    string `json:"filename"     binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Size        int64  `json:"size"         binding:"required,gt=0"`
}

// PresignUploadResponse 回應
type PresignUploadResponse struct {
	FileID     uint   `json:"file_id"`
	UploadURL  string `json:"upload_url"`
	StorageKey string `json:"storage_key"`
	ExpiresIn  int    `json:"expires_in"` // 秒
}

// FileService 檔案服務介面
type FileService interface {
	// PresignUpload 驗證 quota/類型/大小後，產生 Pre-signed PUT URL 並建立 pending 記錄
	PresignUpload(userID uint, req PresignUploadRequest) (*PresignUploadResponse, error)
	// ConfirmUpload 確認物件已上傳至 Minio，更新 status → active
	ConfirmUpload(userID, fileID uint) (*model.File, error)
	// GetFile 取得檔案 metadata（更新 last_accessed_at）
	GetFile(userID, fileID uint) (*model.File, error)
	// GetDownloadURL 產生 Pre-signed 下載 URL，回傳 (url, expiresInSeconds, error)
	GetDownloadURL(userID, fileID uint) (string, int, error)
	// DeleteFile 刪除檔案（Minio + DB）
	DeleteFile(userID, fileID uint) error
	// AttachToMessage 將使用者擁有且已確認的檔案關聯到訊息
	AttachToMessage(userID, messageID, fileID uint) (*model.MessageAttachment, error)
	// GetAttachmentsByMessage 取得訊息的附件清單
	GetAttachmentsByMessage(messageID uint) ([]*model.MessageAttachment, error)
	// CleanupExpired 刪除 TTL 到期的檔案（由 background job 呼叫）
	CleanupExpired() error
}

type fileService struct {
	fileRepo repository.FileRepository
	storage  *storage.Client
	rdb      *goredis.Client
	cfg      *config.MinioConfig
}

// NewFileService 建立 FileService
func NewFileService(
	fileRepo repository.FileRepository,
	storageClient *storage.Client,
	rdb *goredis.Client,
	cfg *config.MinioConfig,
) FileService {
	return &fileService{
		fileRepo: fileRepo,
		storage:  storageClient,
		rdb:      rdb,
		cfg:      cfg,
	}
}

func (s *fileService) PresignUpload(
	userID uint,
	req PresignUploadRequest,
) (*PresignUploadResponse, error) {
	// 1. 驗證副檔名
	ext := strings.ToLower(filepath.Ext(req.Filename))
	if !allowedExtensions[ext] {
		return nil, ErrUnsupportedFileType
	}

	// 2. 單檔大小限制
	maxBytes := s.cfg.MaxFileSizeMB * 1024 * 1024
	if req.Size > maxBytes {
		return nil, ErrFileTooLarge
	}

	// 3. 每日上傳次數限制（Redis 優先，無 Redis 時 fallback DB）
	if err := s.checkDailyQuota(userID, req.Size); err != nil {
		return nil, err
	}

	// 4. 計算 TTL（0 = 永久）
	var expiresAt *time.Time

	if s.cfg.FileTTLDays > 0 {
		t := time.Now().UTC().Add(time.Duration(s.cfg.FileTTLDays) * 24 * time.Hour)
		expiresAt = &t
	}

	// 5. 產生唯一 storage key: uploads/{userID}/{uuid}{ext}
	key := fmt.Sprintf("uploads/%d/%s%s", userID, uuid.New().String(), ext)

	// 6. 建立 pending 記錄
	file := &model.File{
		UserID:         userID,
		Filename:       req.Filename,
		StorageKey:     key,
		ContentType:    req.ContentType,
		Size:           req.Size,
		Status:         fileStatusPending,
		ExpiresAt:      expiresAt,
		LastAccessedAt: time.Now().UTC(),
	}
	if err := s.fileRepo.Create(file); err != nil {
		return nil, fmt.Errorf("file: create record failed: %w", err)
	}

	// 7. 產生 Pre-signed PUT URL
	uploadURL, err := s.storage.PresignPutURL(key, req.ContentType, s.cfg.PresignExpiry)
	if err != nil {
		// 清理剛建立的 pending 記錄
		_ = s.fileRepo.Delete(file.ID)

		return nil, fmt.Errorf("file: presign put url failed: %w", err)
	}

	return &PresignUploadResponse{
		FileID:     file.ID,
		UploadURL:  uploadURL,
		StorageKey: key,
		ExpiresIn:  s.cfg.PresignExpiry * 60,
	}, nil
}

func (s *fileService) ConfirmUpload(userID, fileID uint) (*model.File, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, err
	}

	if file.UserID != userID {
		return nil, ErrFileForbidden
	}

	if file.Status != fileStatusPending {
		return file, nil // 已確認，冪等
	}

	// 驗證物件是否真的在 Minio
	exists, actualSize, err := s.storage.ObjectExists(file.StorageKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrUploadNotConfirmed
	}

	// 以 Minio 回報的實際大小為準
	declaredSize := file.Size
	file.Size = actualSize
	file.Status = fileStatusActive

	if err = s.fileRepo.Update(file); err != nil {
		return nil, err
	}

	// 配額的「次數」已在 PresignUpload 計過（那才是需要擋下來的時機點），
	// 這裡再 INCR 一次會讓每個檔案吃掉兩份額度。但位元組數不能不管：
	// presign 記的是 client 自己宣告的大小，而 Minio 的 presigned PUT
	// 不強制 Content-Length，宣告 1 byte 上傳 1GB 就繞開了每日總量上限。
	// 所以次數不重計，位元組只補上實際與宣告的差額。
	s.reconcileRedisBytes(userID, actualSize-declaredSize)

	return file, nil
}

// reconcileRedisBytes 以差額修正每日位元組計數（正負皆可），不動次數計數。
func (s *fileService) reconcileRedisBytes(userID uint, delta int64) {
	if s.rdb == nil || delta == 0 {
		return
	}

	ctx := context.Background()
	bytesKey := fmt.Sprintf("upload_quota:%d:daily_bytes", userID)

	// 只在 key 還在（今天已有計數）時修正；key 已過期就讓它留在下一個視窗歸零。
	if s.rdb.Exists(ctx, bytesKey).Val() == 0 {
		return
	}

	s.rdb.IncrBy(ctx, bytesKey, delta)
}

func (s *fileService) GetFile(userID, fileID uint) (*model.File, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, err
	}

	if file.UserID != userID {
		return nil, ErrFileForbidden
	}

	_ = s.fileRepo.TouchLastAccessed(fileID)

	return file, nil
}

func (s *fileService) GetDownloadURL(userID, fileID uint) (string, int, error) {
	// Any authenticated user may download a file (e.g. viewing another user's
	// attachment in a shared channel).  We only require the file to be active.
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, ErrFileNotFound
		}

		return "", 0, err
	}

	if file.Status != fileStatusActive {
		return "", 0, ErrFileNotFound
	}

	_ = s.fileRepo.TouchLastAccessed(fileID)

	// public_read 模式：bucket 開放公開存取，URL 為永久固定網址，
	// 無 signature、無過期，瀏覽器可無限期快取。
	if s.cfg.PublicRead {
		return s.storage.PublicFileURL(file.StorageKey), -1, nil
	}

	// 2-minute safety buffer so cached URL never expires before the TTL we advertise.
	const bufferMin = 2

	cacheTTLMin := s.cfg.PresignExpiry - bufferMin
	if cacheTTLMin <= 0 {
		cacheTTLMin = 1
	}

	cacheTTL := time.Duration(cacheTTLMin) * time.Minute

	// Try to serve a cached URL from Redis – same URL → browser can actually cache
	// the image by its stable URL instead of re-downloading on every page render.
	cacheKey := fmt.Sprintf("file:dl_url:%d", fileID)

	if s.rdb != nil {
		ctx := context.Background()

		if cachedURL, redisErr := s.rdb.Get(ctx, cacheKey).Result(); redisErr == nil {
			remaining, _ := s.rdb.TTL(ctx, cacheKey).Result()
			expiresIn := int(remaining.Seconds())

			if expiresIn < 0 {
				expiresIn = 0
			}

			return cachedURL, expiresIn, nil
		}
	}

	presignedURL, err := s.storage.PresignGetURL(file.StorageKey, s.cfg.PresignExpiry)
	if err != nil {
		return "", 0, err
	}

	// Store in Redis so subsequent callers within the window get the same URL.
	if s.rdb != nil {
		_ = s.rdb.Set(context.Background(), cacheKey, presignedURL, cacheTTL).Err()
	}

	return presignedURL, cacheTTLMin * 60, nil
}

func (s *fileService) DeleteFile(userID, fileID uint) error {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileNotFound
		}

		return err
	}

	if file.UserID != userID {
		return ErrFileForbidden
	}

	// 先刪 Minio（允許 object 不存在）
	_ = s.storage.DeleteObject(file.StorageKey)

	// 清除下載 URL 快取
	if s.rdb != nil {
		_ = s.rdb.Del(context.Background(), fmt.Sprintf("file:dl_url:%d", fileID)).Err()
	}

	return s.fileRepo.Delete(fileID)
}

func (s *fileService) AttachToMessage(
	userID, messageID, fileID uint,
) (*model.MessageAttachment, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}

		return nil, err
	}

	if file.UserID != userID {
		return nil, ErrFileForbidden
	}

	if file.Status != fileStatusActive {
		return nil, ErrUploadNotConfirmed
	}

	attachment := &model.MessageAttachment{
		MessageID: messageID,
		FileID:    fileID,
	}

	if err := s.fileRepo.CreateAttachment(attachment); err != nil {
		return nil, err
	}

	return attachment, nil
}

func (s *fileService) GetAttachmentsByMessage(messageID uint) ([]*model.MessageAttachment, error) {
	return s.fileRepo.GetAttachmentsByMessageID(messageID)
}

func (s *fileService) CleanupExpired() error {
	const batchSize = 100

	files, err := s.fileRepo.FindExpired(batchSize)
	if err != nil {
		return err
	}

	for _, f := range files {
		_ = s.storage.DeleteObject(f.StorageKey)
		_ = s.fileRepo.Delete(f.ID)
	}

	return nil
}

// checkDailyQuota 檢查每日上傳次數與總量配額
func (s *fileService) checkDailyQuota(userID uint, size int64) error {
	if s.rdb != nil {
		return s.checkDailyQuotaRedis(userID, size)
	}

	return s.checkDailyQuotaDB(userID, size)
}

func (s *fileService) checkDailyQuotaRedis(userID uint, size int64) error {
	ctx := context.Background()
	now := time.Now().UTC()
	ttl := time.Until(now.Truncate(24 * time.Hour).Add(24 * time.Hour))

	countKey := fmt.Sprintf("upload_quota:%d:daily_count", userID)
	bytesKey := fmt.Sprintf("upload_quota:%d:daily_bytes", userID)

	pipe := s.rdb.Pipeline()
	incrCount := pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, countKey, ttl)
	incrBytes := pipe.IncrBy(ctx, bytesKey, size)
	pipe.Expire(ctx, bytesKey, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		// Redis 故障時 fallback DB
		return s.checkDailyQuotaDB(userID, size)
	}

	if int(incrCount.Val()) > s.cfg.DailyUploadMax {
		// 回滾 Redis 計數（下次 TTL 自動過期）
		s.rdb.DecrBy(ctx, bytesKey, size)
		s.rdb.Decr(ctx, countKey)

		return ErrDailyUploadExceeded
	}

	maxDailyBytes := s.cfg.DailyBytesMaxMB * 1024 * 1024
	if incrBytes.Val() > maxDailyBytes {
		s.rdb.DecrBy(ctx, bytesKey, size)
		s.rdb.Decr(ctx, countKey)

		return ErrDailyBytesExceeded
	}

	return nil
}

func (s *fileService) checkDailyQuotaDB(userID uint, size int64) error {
	count, err := s.fileRepo.CountByUserToday(userID)
	if err != nil {
		return err
	}

	if int(count) >= s.cfg.DailyUploadMax {
		return ErrDailyUploadExceeded
	}

	totalBytes, err := s.fileRepo.SumBytesByUserToday(userID)
	if err != nil {
		return err
	}

	maxDailyBytes := s.cfg.DailyBytesMaxMB * 1024 * 1024
	if totalBytes+size > maxDailyBytes {
		return ErrDailyBytesExceeded
	}

	return nil
}
