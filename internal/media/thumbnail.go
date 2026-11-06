package media

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"path"
	"strings"

	// Register the GIF decoder so image.Decode recognises image/gif.
	_ "image/gif"
)

// Thumbnail sizing. 512 px on the long side is enough for every place the UI
// shows a preview and keeps a portrait thumbnail around 40 KB.
const (
	ThumbnailMaxSide = 512
	// ThumbnailMaxPixels refuses decompression bombs: a 20 MB PNG can decode
	// to gigabytes, so the decoded size is checked before allocating.
	ThumbnailMaxPixels = 40_000_000
)

// thumbnailFormats maps the content types we can decode to the extension used
// for the stored thumbnail. Alpha survives only in the PNG path.
var thumbnailFormats = map[string]string{
	"image/jpeg":  ".jpg",
	"image/pjpeg": ".jpg",
	"image/png":   ".png",
	"image/gif":   ".png",
}

// CanThumbnail reports whether the content type is one we can shrink.
func CanThumbnail(contentType string) bool {
	_, ok := thumbnailFormats[normalizeImageType(contentType)]
	return ok
}

// GenerateThumbnail decodes data and re-encodes a downscaled copy. The second
// return value is the file extension to store it under, which is empty when
// the content type is not a supported image (callers store no thumbnail then).
//
// PNG and GIF sources become PNG so transparency is kept; JPEG stays JPEG. The
// box filter is a plain average over the source rectangle, which is plenty for
// a preview and avoids a dependency on golang.org/x/image.
func GenerateThumbnail(data []byte, contentType string) ([]byte, string, error) {
	ct := normalizeImageType(contentType)
	ext, ok := thumbnailFormats[ct]
	if !ok {
		return nil, "", nil
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("media: decode image config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, "", fmt.Errorf("media: image has no size")
	}
	if int64(cfg.Width)*int64(cfg.Height) > ThumbnailMaxPixels {
		return nil, "", fmt.Errorf("media: image is too large to thumbnail")
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("media: decode image: %w", err)
	}

	dst := downscale(src, ThumbnailMaxSide)
	var buf bytes.Buffer
	if ext == ".png" {
		err = png.Encode(&buf, dst)
	} else {
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82})
	}
	if err != nil {
		return nil, "", fmt.Errorf("media: encode thumbnail: %w", err)
	}
	return buf.Bytes(), ext, nil
}

// downscale shrinks src so its longest side is at most maxSide, averaging the
// source rectangle behind each destination pixel. Images already small enough
// are returned unchanged.
func downscale(src image.Image, maxSide int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}

	newW, newH := w, h
	if w >= h {
		newW = maxSide
		newH = h * maxSide / w
	} else {
		newH = maxSide
		newW = w * maxSide / h
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		y0 := bounds.Min.Y + y*h/newH
		y1 := bounds.Min.Y + (y+1)*h/newH
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < newW; x++ {
			x0 := bounds.Min.X + x*w/newW
			x1 := bounds.Min.X + (x+1)*w/newW
			if x1 <= x0 {
				x1 = x0 + 1
			}
			dst.Set(x, y, averageRect(src, x0, y0, x1, y1))
		}
	}
	return dst
}

// averageRect averages the pixels of src in [x0,x1)×[y0,y1). Colours are
// summed in 16-bit space so wide images do not overflow.
func averageRect(src image.Image, x0, y0, x1, y1 int) color4 {
	var r, g, b, a uint64
	var n uint64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cr, cg, cb, ca := src.At(x, y).RGBA()
			r += uint64(cr)
			g += uint64(cg)
			b += uint64(cb)
			a += uint64(ca)
			n++
		}
	}
	if n == 0 {
		return color4{}
	}
	return color4{
		r: uint8(r / n >> 8),
		g: uint8(g / n >> 8),
		b: uint8(b / n >> 8),
		a: uint8(a / n >> 8),
	}
}

type color4 struct{ r, g, b, a uint8 }

// RGBA implements color.Color without importing image/color at every call site.
func (c color4) RGBA() (r, g, b, a uint32) {
	return uint32(c.r) * 0x101, uint32(c.g) * 0x101, uint32(c.b) * 0x101, uint32(c.a) * 0x101
}

// ThumbnailKeyFor derives the storage key of a thumbnail from the original's.
func ThumbnailKeyFor(key, ext string) string {
	base := strings.TrimSuffix(key, path.Ext(key))
	return base + ".thumb" + ext
}

// normalizeImageType lowercases a content type and drops parameters such as
// the "; charset" suffix some browsers send.
func normalizeImageType(contentType string) string {
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}
