package media

import (
	"net/url"
	"strings"
	"testing"
)

// TestS3URLModes proves the three URL shapes: a configured public base,
// an unsigned direct URL, and a presigned URL for a private bucket.
func TestS3URLModes(t *testing.T) {
	publicBase, err := NewS3Storage(S3Options{
		Endpoint: "s3.example:9000", Bucket: "cinderfish", Region: "us-east-1", AccessKey: "a", SecretKey: "s",
		PublicBaseURL: "https://cdn.example/media",
	})
	if err != nil {
		t.Fatalf("public storage: %v", err)
	}
	if got := publicBase.URL("ab/cd/key"); got != "https://cdn.example/media/ab/cd/key" {
		t.Errorf("public URL = %q", got)
	}

	presigned, err := NewS3Storage(S3Options{
		Endpoint: "s3.example:9000", Bucket: "cinderfish", Region: "us-east-1", AccessKey: "a", SecretKey: "s",
		Presign: true,
	})
	if err != nil {
		t.Fatalf("presign storage: %v", err)
	}
	raw := presigned.URL("ab/cd/key")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse presigned URL %q: %v", raw, err)
	}
	if !strings.HasSuffix(u.Path, "/cinderfish/ab/cd/key") {
		t.Errorf("presigned path = %q", u.Path)
	}
	if u.Query().Get("X-Amz-Signature") == "" || u.Query().Get("X-Amz-Credential") == "" {
		t.Errorf("presigned URL has no signature: %q", raw)
	}

	unsigned, err := NewS3Storage(S3Options{
		Endpoint: "s3.example:9000", Bucket: "cinderfish", Region: "us-east-1", AccessKey: "a", SecretKey: "s",
	})
	if err != nil {
		t.Fatalf("unsigned storage: %v", err)
	}
	if got := unsigned.URL("ab/cd/key"); got != "http://s3.example:9000/cinderfish/ab/cd/key" {
		t.Errorf("unsigned URL = %q", got)
	}
}
