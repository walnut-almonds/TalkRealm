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
	mc        *minio.Client // internal client – used for admin ops (bucket, stat, delete)
	presignMC *minio.Client // presign client – signs URLs against the public endpoint
	bucket    string
	cfg       *config.MinioConfig
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

	// Build a dedicated presign client whose endpoint is the public URL that
	// the browser will PUT/GET against.  AWS v4 signs the Host header, so the
	// signing endpoint must equal the host the browser sends the request to.
	//
	// Region + BucketLookupPath prevent the SDK from making any outbound
	// network requests during presign (it would otherwise call the public host
	// for a bucket-region probe, which can time out inside the container).
	presignMC := mc
	if cfg.PublicEndpoint != "" {
		pub, parseErr := url.Parse(cfg.PublicEndpoint)
		if parseErr != nil {
			return nil, fmt.Errorf(
				"minio: invalid public_endpoint %q: %w",
				cfg.PublicEndpoint,
				parseErr,
			)
		}

		presignMC, err = minio.New(pub.Host, &minio.Options{
			Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure:       pub.Scheme == "https",
			Region:       "us-east-1",            // skip region auto-detect (no outbound call)
			BucketLookup: minio.BucketLookupPath, // path-style; no per-bucket DNS lookup
		})
		if err != nil {
			return nil, fmt.Errorf("minio: presign client init failed: %w", err)
		}

		logger.Info("Minio presign client", "public_endpoint", cfg.PublicEndpoint)
	}

	return &Client{mc: mc, presignMC: presignMC, bucket: cfg.Bucket, cfg: cfg}, nil
}

func (c *Client) PresignPutURL(key, contentType string, expiry int) (string, error) {
	u, err := c.presignMC.PresignedPutObject(
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

func (c *Client) PresignGetURL(key string, expiry int) (string, error) {
	u, err := c.presignMC.PresignedGetObject(
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
