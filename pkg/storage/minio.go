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

type Client struct {
	mc     *minio.Client
	bucket string
	cfg    *config.MinioConfig
}

func NewClient(cfg *config.MinioConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: connect failed: %w", err)
	}

	ctx := context.Background()

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

// rewritePublicURL replaces the host (and scheme) of a presigned URL with the
// configured public endpoint so that browser clients always receive an
// externally-reachable HTTPS URL even when the MinIO SDK signs against an
// internal address (e.g. "minio:9000").
func (c *Client) rewritePublicURL(u *url.URL) string {
	if c.cfg.PublicEndpoint == "" {
		return u.String()
	}

	pub, err := url.Parse(c.cfg.PublicEndpoint)
	if err != nil {
		return u.String()
	}

	rewritten := *u
	rewritten.Scheme = pub.Scheme
	rewritten.Host = pub.Host

	return rewritten.String()
}

func (c *Client) PresignPutURL(key, contentType string, expiry int) (string, error) {
	params := url.Values{}
	if contentType != "" {
		params.Set("Content-Type", contentType)
	}

	u, err := c.mc.PresignedPutObject(
		context.Background(),
		c.bucket,
		key,
		time.Duration(expiry)*time.Minute,
	)
	if err != nil {
		return "", fmt.Errorf("minio: presign put failed: %w", err)
	}

	return c.rewritePublicURL(u), nil
}

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

	return c.rewritePublicURL(u), nil
}

func (c *Client) DeleteObject(key string) error {
	err := c.mc.RemoveObject(context.Background(), c.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio: delete object failed: %w", err)
	}

	return nil
}

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
