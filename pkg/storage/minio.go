package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

// Client 封裝 Minio 操作
type Client struct {
	mc     *minio.Client
	bucket string
	cfg    *config.MinioConfig
}

// NewClient 建立 Minio Client，並確保 bucket 存在
func NewClient(cfg *config.MinioConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: connect failed: %w", err)
	}

	ctx := context.Background()

	// 確保 bucket 存在，不存在時自動建立
	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: check bucket failed: %w", err)
	}

	if !exists {
		if err = mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio: create bucket failed: %w", err)
		}

		logger.Info("Minio bucket created", "bucket", cfg.Bucket)
	}

	logger.Info("Minio connected", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket)

	return &Client{mc: mc, bucket: cfg.Bucket, cfg: cfg}, nil
}

// PresignPutURL 產生 Pre-signed 上傳 URL（PUT）
// expiry 單位：分鐘
func (c *Client) PresignPutURL(key string, expiry int) (string, error) {
	_ = url.Values{} // ensure url import is used via other methods

	u, err := c.mc.PresignedPutObject(
		context.Background(),
		c.bucket,
		key,
		time.Duration(expiry)*time.Minute,
	)
	if err != nil {
		return "", fmt.Errorf("minio: presign put failed: %w", err)
	}

	return u.String(), nil
}

// PresignGetURL 產生 Pre-signed 下載 URL（GET）
func (c *Client) PresignGetURL(key string, expiry int) (string, error) {
	u, err := c.mc.PresignedGetObject(
		context.Background(),
		c.bucket,
		key,
		time.Duration(expiry)*time.Minute,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("minio: presign get failed: %w", err)
	}

	return u.String(), nil
}

// DeleteObject 刪除 Minio 上的物件
func (c *Client) DeleteObject(key string) error {
	err := c.mc.RemoveObject(context.Background(), c.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio: delete object failed: %w", err)
	}

	return nil
}

// ObjectExists 確認物件是否存在（upload confirm 時使用）
func (c *Client) ObjectExists(key string) (bool, int64, error) {
	info, err := c.mc.StatObject(context.Background(), c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, 0, nil
		}

		return false, 0, fmt.Errorf("minio: stat object failed: %w", err)
	}

	return true, info.Size, nil
}
