package httpapi_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
)

// TestIUpdateBanner covers the profile banner: set it from an owned drive file,
// clear it with null, and refuse another account's file.
func TestIUpdateBanner(t *testing.T) {
	ts := newTestServer(t, "open")
	aliceID, aliceToken := createUser(t, ts, "alice")
	_, bobToken := createUser(t, ts, "bob")

	img := image.NewRGBA(image.Rect(0, 0, 640, 320))
	for y := 0; y < 320; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	file := uploadTyped(t, ts.URL, aliceToken, "banner.png", "image/png", buf.Bytes())
	fileID, _ := file["id"].(string)
	fileURL, _ := file["url"].(string)
	if fileID == "" || fileURL == "" {
		t.Fatalf("upload returned no id/url: %v", file)
	}

	resp, me := postWithToken(t, ts, "/api/i/update", aliceToken, map[string]any{"bannerId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update: status %d body %v", resp.StatusCode, me)
	}
	if me["bannerUrl"] != fileURL {
		t.Errorf("MeDetailed bannerUrl = %v, want %q", me["bannerUrl"], fileURL)
	}

	_, show := post(t, ts, "/api/users/show", map[string]any{"userId": aliceID})
	if show["bannerUrl"] != fileURL {
		t.Errorf("users/show bannerUrl = %v, want %q", show["bannerUrl"], fileURL)
	}

	// null clears it.
	if resp, body := postWithToken(t, ts, "/api/i/update", aliceToken, map[string]any{"bannerId": nil}); resp.StatusCode != http.StatusOK {
		t.Fatalf("clear banner: status %d body %v", resp.StatusCode, body)
	}
	_, show = post(t, ts, "/api/users/show", map[string]any{"userId": aliceID})
	if show["bannerUrl"] != nil {
		t.Errorf("bannerUrl after clear = %v, want null", show["bannerUrl"])
	}

	// Another account's file is refused before it reaches the profile.
	bobFile := uploadTyped(t, ts.URL, bobToken, "bob.png", "image/png", buf.Bytes())
	resp, body := postWithToken(t, ts, "/api/i/update", aliceToken, map[string]any{"bannerId": bobFile["id"]})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign file status = %d, want 403 (%v)", resp.StatusCode, body)
	}
	// A missing file id is a client error.
	if resp, _ := postWithToken(t, ts, "/api/i/update", aliceToken, map[string]any{"bannerId": "00000000000000000000"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing file status = %d, want 400", resp.StatusCode)
	}
}
