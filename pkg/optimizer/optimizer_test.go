package optimizer

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConvertImageURLToWebP(t *testing.T) {
	// Create a dummy 100x100 PNG image in memory
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode dummy png: %v", err)
	}

	// Serve the dummy PNG image via mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	testURL := ts.URL + "/hero-8.png"
	res, err := ConvertImageURLToWebP(testURL, 80)
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}

	if res.Filename != "hero-8.webp" {
		t.Errorf("expected filename hero-8.webp, got %s", res.Filename)
	}
	if res.OriginalBytes == 0 {
		t.Errorf("expected original bytes > 0")
	}
	if res.OptimizedBytes == 0 {
		t.Errorf("expected optimized bytes > 0")
	}
	if res.OptimizedWebPBase64 == "" {
		t.Errorf("expected webp base64 non-empty")
	}

	// Test ZIP creation
	zipBytes, err := CreateZIPArchive([]*ConversionResult{res})
	if err != nil {
		t.Fatalf("CreateZIPArchive failed: %v", err)
	}
	if len(zipBytes) == 0 {
		t.Errorf("expected non-empty zip bytes")
	}
}

func TestConvertImageURLToWebPAuth(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "uat" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	if _, err := ConvertImageURLToWebP(ts.URL+"/a.png", 80); err == nil {
		t.Fatal("expected error without auth")
	}
	res, err := ConvertImageURLToWebPAuth(ts.URL+"/a.png", 80, "uat", "secret")
	if err != nil {
		t.Fatalf("with auth: %v", err)
	}
	if res.OptimizedBytes == 0 {
		t.Fatal("expected webp bytes")
	}
}
