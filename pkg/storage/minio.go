package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

type Client struct {
	mc        *minio.Client // internal client – bucket ops, stat, delete
	presignMC *minio.Client // presign client – points to public endpoint so SigV4 host matches
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

	// 若設定為 public_read，套用 bucket public-read policy，
	// 讓物件可透過穩定 URL 直接存取，無需 pre-signed signature。
	if cfg.PublicRead {
		if err = applyPublicReadPolicy(ctx, mc, cfg.Bucket); err != nil {
			return nil, fmt.Errorf("minio: set public-read policy failed: %w", err)
		}

		logger.Info("Minio bucket set to public-read", "bucket", cfg.Bucket)
	}

	// Build a second client that signs requests with the public endpoint as host.
	// SigV4 embeds the Host in the canonical request; if the signing host differs
	// from the host MinIO sees (e.g. minio:9000 vs media.qrumi.org), validation
	// fails with SignatureDoesNotMatch.  By pointing this client directly at the
	// public endpoint we ensure the signed host always matches the incoming Host
	// header, making MINIO_SERVER_URL unnecessary for presign correctness.
	presignMC := mc // fallback: reuse internal client when no public endpoint

	if cfg.PublicEndpoint != "" {
		pub, parseErr := url.Parse(cfg.PublicEndpoint)
		if parseErr == nil {
			useSSL := strings.EqualFold(pub.Scheme, "https")

			presignClient, newErr := minio.New(pub.Host, &minio.Options{
				Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
				Secure:       useSSL,
				Region:       "us-east-1",
				BucketLookup: minio.BucketLookupPath,
			})
			if newErr == nil {
				presignMC = presignClient

				logger.Info("Minio presign client using public endpoint", "endpoint", pub.Host)
			} else {
				logger.Warn(
					"Minio presign client init failed, falling back to internal",
					"err",
					newErr,
				)
			}
		}
	}

	return &Client{mc: mc, presignMC: presignMC, bucket: cfg.Bucket, cfg: cfg}, nil
}

// applyPublicReadPolicy 設定 bucket 為 public-read（允許任何人 GET 物件）。
func applyPublicReadPolicy(ctx context.Context, mc *minio.Client, bucket string) error {
	type statement struct {
		Effect    string   `json:"Effect"`
		Principal string   `json:"Principal"`
		Action    []string `json:"Action"`
		Resource  []string `json:"Resource"`
	}

	type policy struct {
		Version   string      `json:"Version"`
		Statement []statement `json:"Statement"`
	}

	p := policy{
		Version: "2012-10-17",
		Statement: []statement{
			{
				Effect:    "Allow",
				Principal: "*",
				Action:    []string{"s3:GetObject"},
				Resource:  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			},
		},
	}

	policyJSON, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return mc.SetBucketPolicy(ctx, bucket, string(policyJSON))
}

// PublicFileURL 回傳物件的公開永久 URL（需 bucket 已設為 public-read）。
// 格式：{publicEndpoint}/{bucket}/{key}
func (c *Client) PublicFileURL(key string) string {
	base := c.cfg.PublicEndpoint
	if base == "" {
		scheme := "http"
		if c.cfg.UseSSL {
			scheme = "https"
		}

		base = fmt.Sprintf("%s://%s", scheme, c.cfg.Endpoint)
	}

	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(base, "/"), c.cfg.Bucket, key)
}

func (c *Client) PresignPutURL(key, _ string, expiry int) (string, error) {
	// Use presignMC (public endpoint) so the SigV4 canonical host matches
	// media.qrumi.org – the host MinIO sees when the request arrives via nginx.
	// Content-Type is intentionally excluded from signed headers; the client
	// still sends it in the actual PUT so MinIO stores it correctly.
	u, err := c.presignMC.PresignHeader(
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
