package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Options configures the S3 compatible backend.
type S3Options struct {
	Endpoint      string
	Bucket        string
	Region        string
	AccessKey     string
	SecretKey     string
	UseSSL        bool
	PublicBaseURL string
	// Presign makes the backend issue short lived signed URLs instead of
	// relying on a public bucket.
	Presign bool
	// PresignTTL is how long a signed URL stays valid.
	PresignTTL time.Duration
}

// S3Storage stores files in an S3 compatible object store. It works with
// AWS S3, Cloudflare R2, Backblaze B2, MinIO and any other service that
// speaks the S3 API.
type S3Storage struct {
	client *minio.Client
	opts   S3Options
}

// NewS3Storage returns an S3Storage for the given options.
func NewS3Storage(opts S3Options) (*S3Storage, error) {
	if opts.Endpoint == "" || opts.Bucket == "" {
		return nil, errors.New("media: s3 endpoint and bucket are required")
	}
	if opts.PresignTTL <= 0 {
		opts.PresignTTL = 5 * time.Minute
	}

	client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: opts.UseSSL,
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("media: create s3 client: %w", err)
	}

	return &S3Storage{client: client, opts: opts}, nil
}

// Backend names the implementation.
func (s *S3Storage) Backend() string { return "s3" }

// Put uploads the content under key.
func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	if _, err := s.client.PutObject(ctx, s.opts.Bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return "", fmt.Errorf("media: put object: %w", err)
	}
	return s.URL(key), nil
}

// Get downloads the object stored under key.
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.opts.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("media: get object: %w", err)
	}
	// The object is fetched lazily; Stat forces the request so a missing
	// key is reported here rather than on the first read.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("media: stat object: %w", err)
	}
	return obj, nil
}

// Delete removes the object stored under key.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.opts.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("media: delete object: %w", err)
	}
	return nil
}

// URL returns the URL clients use to fetch the object.
//
// A public base URL is used when configured. Otherwise a presigned URL
// is issued, which means the URL is only valid for a short while and
// has to be requested per use.
func (s *S3Storage) URL(key string) string {
	if s.opts.PublicBaseURL != "" {
		return strings.TrimSuffix(s.opts.PublicBaseURL, "/") + "/" + strings.TrimPrefix(key, "/")
	}
	if !s.opts.Presign {
		return s.publicURL(key)
	}
	return s.presign(key)
}

// presign issues a signed URL for the key.
func (s *S3Storage) presign(key string) string {
	u, err := s.client.PresignedGetObject(context.Background(), s.opts.Bucket, key, s.opts.PresignTTL, url.Values{})
	if err != nil {
		return s.publicURL(key)
	}
	return u.String()
}

// publicURL builds the unsigned object URL.
func (s *S3Storage) publicURL(key string) string {
	scheme := "http"
	if s.opts.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.opts.Endpoint, s.opts.Bucket, strings.TrimPrefix(key, "/"))
}
