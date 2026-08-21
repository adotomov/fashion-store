package imageopt

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/gen2brain/webp"
)

// sampleJPEG builds a non-trivial gradient image and returns its JPEG bytes.
func sampleJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8((x + y) / 2), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode sample jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestConvertToWebP_TranscodesJPEG(t *testing.T) {
	src := sampleJPEG(t)

	out, ok := ConvertToWebP(src, "image/jpeg")
	if !ok {
		t.Fatal("expected JPEG to be converted to WebP")
	}
	// The output must be a valid WebP that decodes back to the right dimensions.
	img, err := webp.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("re-decode webp: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
		t.Fatalf("unexpected decoded size: %v", b.Size())
	}
}

func TestConvertToWebP_SkipsWebP(t *testing.T) {
	if _, ok := ConvertToWebP([]byte("not really webp"), "image/webp"); ok {
		t.Fatal("already-webp input should be passed through unchanged")
	}
}

func TestConvertToWebP_SkipsNonImage(t *testing.T) {
	// An SVG (or any non-raster payload) is not decodable and must pass through.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	out, ok := ConvertToWebP(svg, "image/svg+xml")
	if ok {
		t.Fatal("SVG should not be converted")
	}
	if !bytes.Equal(out, svg) {
		t.Fatal("non-image input should be returned unchanged")
	}
}

func TestOptimize_SwapsExtensionAndType(t *testing.T) {
	data, ct, name := Optimize(sampleJPEG(t), "image/jpeg", "Photo.JPG")
	if ct != "image/webp" {
		t.Fatalf("content type = %q, want image/webp", ct)
	}
	if name != "Photo.webp" {
		t.Fatalf("filename = %q, want Photo.webp", name)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty webp output")
	}
}
