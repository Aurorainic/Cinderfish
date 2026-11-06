package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// encodePNG builds a w×h image with a smooth gradient so the downscale has
// something real to average.
func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func decodeConfig(t *testing.T, data []byte) image.Config {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode thumbnail config: %v", err)
	}
	return cfg
}

// TestGenerateThumbnailModes covers the three supported source types and
// their output format: PNG and GIF become PNG, JPEG stays JPEG.
func TestGenerateThumbnailModes(t *testing.T) {
	pngData := encodePNG(t, 800, 400)
	thumb, ext, err := GenerateThumbnail(pngData, "image/png")
	if err != nil || thumb == nil {
		t.Fatalf("png thumbnail: %v (nil=%v)", err, thumb == nil)
	}
	if ext != ".png" {
		t.Errorf("png extension = %q, want .png", ext)
	}
	if cfg := decodeConfig(t, thumb); cfg.Width != 512 || cfg.Height != 256 {
		t.Errorf("png thumbnail = %dx%d, want 512x256", cfg.Width, cfg.Height)
	}

	// A portrait image is bounded on its long side too.
	portrait := encodePNG(t, 300, 900)
	thumb, _, err = GenerateThumbnail(portrait, "image/png")
	if err != nil {
		t.Fatalf("portrait thumbnail: %v", err)
	}
	if cfg := decodeConfig(t, thumb); cfg.Width != 170 || cfg.Height != 512 {
		t.Errorf("portrait thumbnail = %dx%d, want 170x512", cfg.Width, cfg.Height)
	}

	// Images already inside the bound are not upscaled.
	small := encodePNG(t, 100, 50)
	thumb, _, err = GenerateThumbnail(small, "image/png")
	if err != nil {
		t.Fatalf("small thumbnail: %v", err)
	}
	if cfg := decodeConfig(t, thumb); cfg.Width != 100 || cfg.Height != 50 {
		t.Errorf("small thumbnail = %dx%d, want 100x50", cfg.Width, cfg.Height)
	}

	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, image.NewRGBA(image.Rect(0, 0, 640, 480)), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	thumb, ext, err = GenerateThumbnail(jpegBuf.Bytes(), "image/jpeg")
	if err != nil || thumb == nil {
		t.Fatalf("jpeg thumbnail: %v", err)
	}
	if ext != ".jpg" {
		t.Errorf("jpeg extension = %q, want .jpg", ext)
	}
	if _, err := jpeg.Decode(bytes.NewReader(thumb)); err != nil {
		t.Errorf("jpeg thumbnail does not decode as jpeg: %v", err)
	}
	if cfg := decodeConfig(t, thumb); cfg.Width != 512 || cfg.Height != 384 {
		t.Errorf("jpeg thumbnail = %dx%d, want 512x384", cfg.Width, cfg.Height)
	}

	// The "; charset" parameter and mixed case are tolerated.
	var gifBuf bytes.Buffer
	if err := gif.Encode(&gifBuf, image.NewPaletted(image.Rect(0, 0, 300, 300), color.Palette{color.Black, color.White}), nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	thumb, ext, err = GenerateThumbnail(gifBuf.Bytes(), "Image/GIF; charset=binary")
	if err != nil || thumb == nil {
		t.Fatalf("gif thumbnail: %v", err)
	}
	if ext != ".png" {
		t.Errorf("gif extension = %q, want .png", ext)
	}
	if cfg := decodeConfig(t, thumb); cfg.Width != 300 || cfg.Height != 300 {
		t.Errorf("gif thumbnail = %dx%d, want unchanged 300x300", cfg.Width, cfg.Height)
	}
}

// TestGenerateThumbnailRefusals covers the types and inputs that must not
// become a thumbnail.
func TestGenerateThumbnailRefusals(t *testing.T) {
	// Unsupported types answer "no thumbnail" without an error: callers
	// store the upload anyway.
	for _, ct := range []string{"image/webp", "image/svg+xml", "application/pdf", ""} {
		thumb, ext, err := GenerateThumbnail(encodePNG(t, 10, 10), ct)
		if thumb != nil || ext != "" || err != nil {
			t.Errorf("%q = (%v, %q, %v), want no thumbnail and no error", ct, thumb, ext, err)
		}
	}

	if _, _, err := GenerateThumbnail([]byte("not an image"), "image/png"); err == nil {
		t.Error("corrupt png accepted, want a decode error")
	}

	// A 10000×10000 header must be refused before any pixels are decoded.
	bomb := pngWithSize(t, encodePNG(t, 1, 1), 10000, 10000)
	if _, _, err := GenerateThumbnail(bomb, "image/png"); err == nil {
		t.Error("decompression bomb accepted, want a size refusal")
	}
}

// pngWithSize rewrites the IHDR dimensions of a PNG, leaving the rest of the
// file intact, so the header advertises a huge image without the bytes.
func pngWithSize(t *testing.T, data []byte, w, h uint32) []byte {
	t.Helper()
	out := append([]byte(nil), data...)
	if len(out) < 33 || string(out[12:16]) != "IHDR" {
		t.Fatalf("unexpected png layout")
	}
	binary.BigEndian.PutUint32(out[16:20], w)
	binary.BigEndian.PutUint32(out[20:24], h)
	binary.BigEndian.PutUint32(out[29:33], crc32.ChecksumIEEE(out[12:29]))
	return out
}

func TestThumbnailKeyFor(t *testing.T) {
	cases := []struct{ key, ext, want string }{
		{"ab/cd/abcd", ".png", "ab/cd/abcd.thumb.png"},
		{"ab/cd/abcd.jpg", ".jpg", "ab/cd/abcd.thumb.jpg"},
		{"file", ".png", "file.thumb.png"},
	}
	for _, c := range cases {
		if got := ThumbnailKeyFor(c.key, c.ext); got != c.want {
			t.Errorf("ThumbnailKeyFor(%q, %q) = %q, want %q", c.key, c.ext, got, c.want)
		}
	}
}

// TestDriveSaveStoresThumbnail proves the upload path itself writes the
// thumbnail object, records its key, and removes both on delete.
func TestDriveSaveStoresThumbnail(t *testing.T) {
	ctx := context.Background()
	pool := migratePool(t)
	storage, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	drive := NewDriveService(pool, storage)

	uploaded, err := drive.Save(ctx, "u1", "photo.png", bytes.NewReader(encodePNG(t, 800, 400)), "image/png", nil)
	if err != nil {
		t.Fatalf("save png: %v", err)
	}
	if uploaded.ThumbnailKey == nil || *uploaded.ThumbnailKey == "" {
		t.Fatalf("png upload has no thumbnail key: %+v", uploaded)
	}
	if got := *uploaded.ThumbnailKey; got != ThumbnailKeyFor(uploaded.StorageKey, ".png") {
		t.Errorf("thumbnail key = %q, want the derived key", got)
	}
	if uploaded.ThumbnailURL != "/media/"+*uploaded.ThumbnailKey {
		t.Errorf("thumbnail url = %q, want the media path", uploaded.ThumbnailURL)
	}

	// The stored thumbnail decodes and is bounded.
	rc, err := storage.Get(ctx, *uploaded.ThumbnailKey)
	if err != nil {
		t.Fatalf("thumbnail object: %v", err)
	}
	var thumbBuf bytes.Buffer
	if _, err := thumbBuf.ReadFrom(rc); err != nil {
		t.Fatalf("read thumbnail: %v", err)
	}
	rc.Close()
	if cfg := decodeConfig(t, thumbBuf.Bytes()); cfg.Width != 512 || cfg.Height != 256 {
		t.Errorf("stored thumbnail = %dx%d, want 512x256", cfg.Width, cfg.Height)
	}

	// Get and List carry the resolved URL.
	got, err := drive.Get(ctx, uploaded.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ThumbnailURL != uploaded.ThumbnailURL {
		t.Errorf("get thumbnail url = %q, want %q", got.ThumbnailURL, uploaded.ThumbnailURL)
	}
	list, err := drive.List(ctx, "u1", 10, "", "", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ThumbnailURL != uploaded.ThumbnailURL {
		t.Errorf("list thumbnail = %+v, want %q", list, uploaded.ThumbnailURL)
	}

	// Non-images store no thumbnail and still succeed.
	doc, err := drive.Save(ctx, "u1", "notes.txt", bytes.NewBufferString("hello"), "text/plain", nil)
	if err != nil {
		t.Fatalf("save text: %v", err)
	}
	if doc.ThumbnailKey != nil || doc.ThumbnailURL != "" {
		t.Errorf("text upload got a thumbnail: %+v", doc)
	}

	// Deleting the image removes the thumbnail object as well.
	if err := drive.Delete(ctx, uploaded.ID, "u1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := storage.Get(ctx, *uploaded.ThumbnailKey); !errors.Is(err, ErrNotFound) {
		t.Errorf("thumbnail after delete = %v, want ErrNotFound", err)
	}
	if _, err := storage.Get(ctx, uploaded.StorageKey); !errors.Is(err, ErrNotFound) {
		t.Errorf("original after delete = %v, want ErrNotFound", err)
	}
}

// TestDriveSaveCorruptImageKeepsUpload proves a broken image is still stored,
// just without a thumbnail.
func TestDriveSaveCorruptImageKeepsUpload(t *testing.T) {
	ctx := context.Background()
	pool := migratePool(t)
	storage, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	drive := NewDriveService(pool, storage)

	file, err := drive.Save(ctx, "u1", "broken.png", bytes.NewBufferString("PNG? no"), "image/png", nil)
	if err != nil {
		t.Fatalf("save broken image: %v", err)
	}
	if file.ThumbnailKey != nil {
		t.Errorf("broken image produced a thumbnail key: %v", *file.ThumbnailKey)
	}
	if _, err := storage.Get(ctx, file.StorageKey); err != nil {
		t.Errorf("original missing: %v", err)
	}
}
