package httpapi_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
)

// uploadTyped uploads a file declaring its real content type, unlike
// uploadFile which always sends application/octet-stream.
func uploadTyped(t *testing.T, baseURL, token, name, contentType string, content []byte) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/drive/files/create", &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload %s: %d %s", name, resp.StatusCode, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	return out
}

// TestMediaServeContentType proves /media serves uploads with their declared
// type instead of octet-stream, and thumbnails with the encoded format.
func TestMediaServeContentType(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	token, _ := alice["token"].(string)

	img := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	file := uploadTyped(t, ts.URL, token, "photo.png", "image/png", pngBuf.Bytes())
	url, _ := file["url"].(string)
	thumbURL, _ := file["thumbnailUrl"].(string)
	if url == "" || thumbURL == "" {
		t.Fatalf("upload has no url/thumbnailUrl: %v", file)
	}

	get := func(path string) (int, string, []byte) {
		resp, err := ts.Client().Get(ts.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		buf, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), buf
	}

	status, contentType, body := get(url)
	if status != http.StatusOK {
		t.Fatalf("original = %d, want 200", status)
	}
	if contentType != "image/png" {
		t.Errorf("original content type = %q, want image/png", contentType)
	}
	if !bytes.Equal(body, pngBuf.Bytes()) {
		t.Errorf("original bytes changed: %d vs %d", len(body), pngBuf.Len())
	}

	status, contentType, body = get(thumbURL)
	if status != http.StatusOK {
		t.Fatalf("thumbnail = %d, want 200", status)
	}
	if contentType != "image/png" {
		t.Errorf("thumbnail content type = %q, want image/png", contentType)
	}
	if len(body) == 0 || len(body) >= pngBuf.Len() {
		t.Errorf("thumbnail bytes = %d, want a smaller non-empty image", len(body))
	}

	// A non-image keeps its declared type too.
	doc := uploadTyped(t, ts.URL, token, "notes.txt", "text/plain", []byte("hello"))
	docURL, _ := doc["url"].(string)
	if _, contentType, _ = get(docURL); contentType != "text/plain" {
		t.Errorf("text content type = %q, want text/plain", contentType)
	}
	if doc["thumbnailUrl"] != nil {
		t.Errorf("text upload got a thumbnail: %v", doc["thumbnailUrl"])
	}
}
