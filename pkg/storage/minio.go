package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

type Client struct {
	mc     *minio.Client // internal client – all MinIO operations
	bucket string
	cfg    *config.MinioConfig
}

func NewClient(cfg *config.MinioConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
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

// rewritePublicURL replaces the scheme+host of an internally-signed presigned
// URL with the configured public endpoint.
//
// AWS v4 signature validation on the MinIO side is made possible by setting
// MINIO_SERVER_URL=<public_endpoint> in MinIO's environment: MinIO will
// normalise the Host header against its server URL before verifying the
// signature, so a URL signed with "minio:9000" but served via
// "media.qrumi.org" will still pass validation.
func (c *Client) rewritePublicURL(u *url.URL) string {
	if c.cfg.PublicEndpoint == "" {
		return u.String()
	}

	pub, err := url.Parse(c.cfg.PublicEndpoint)
	if err != nil {
		return u.String()
	}

	out := *u
	out.Scheme = pub.Scheme
	out.Host = pub.Host

	return out.String()
}

func (c *Client) PresignPutURL(key, contentType string, expiry int) (string, error) {
	// Do NOT include Content-Type in signed headers.
	// If content-type is in X-Amz-SignedHeaders, MinIO validates the header value
	// against the signature during the OPTIONS preflight (CORS) and the actual PUT,
	// which causes signature mismatches because the signing host (minio:9000) differs
	// from the public host (media.qrumi.org).  Only signing 'host' mirrors how
	// PresignedGetObject works and is sufficient for authorisation.
	// The client still sends Content-Type in the PUT request so MinIO stores it.
	u, err := c.mc.PresignHeader(
		context.Background(),
		http.MethodPut,
		c.bucket,
		key,
		time.Duration(expiry)*time.Minute,
		nil,
		nil,
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
