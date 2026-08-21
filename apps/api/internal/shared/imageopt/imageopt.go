// Package imageopt transcodes uploaded raster images to WebP so the store
// serves small, fast-loading files instead of the multi-megabyte JPEG/PNG
// originals shoppers' browsers would otherwise download. Conversion is
// best-effort: anything that isn't a decodable raster image (SVG, PDFs, an
// already-WebP file) is passed through untouched, so callers can route every
// upload through it safely.
package imageopt

import (
	"bytes"
	"image"
	// Register the standard raster decoders so image.Decode understands the
	// formats browsers actually upload. WebP sources are short-circuited
	// before decode, so no WebP decoder import is needed here.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path"
	"strings"

	"github.com/gen2brain/webp"
)

// webpQuality trades visual fidelity against file size for photographic
// content. 82 is near-visually-lossless for product photography while
// typically cutting a 5–6 MB JPEG down to a few hundred KB.
const webpQuality = 82

// webpMethod is libwebp's quality/speed trade-off (0=fast … 6=slowest/best).
// 4 is the library default — a good balance for a synchronous upload path.
const webpMethod = 4

// ConvertToWebP transcodes raw image bytes to lossy WebP. It returns the WebP
// bytes and true on success; if data isn't a decodable raster image, or is
// already WebP, it returns the input unchanged with false.
func ConvertToWebP(data []byte, contentType string) ([]byte, bool) {
	if strings.EqualFold(strings.TrimSpace(contentType), "image/webp") {
		return data, false
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, false // not a raster image we understand (e.g. SVG)
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: webpQuality, Method: webpMethod}); err != nil {
		return data, false
	}
	return buf.Bytes(), true
}

// webpFilename swaps a filename's extension to .webp so the stored object key
// matches its new content.
func webpFilename(filename string) string {
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if strings.TrimSpace(base) == "" {
		base = "image"
	}
	return base + ".webp"
}

// Optimize converts raster image bytes to WebP, returning the new bytes,
// content type ("image/webp") and filename (extension swapped to .webp). When
// the input can't be converted it returns everything unchanged, so it's safe
// to call on every upload regardless of type.
func Optimize(data []byte, contentType, filename string) (outData []byte, outContentType, outFilename string) {
	converted, ok := ConvertToWebP(data, contentType)
	if !ok {
		return data, contentType, filename
	}
	return converted, "image/webp", webpFilename(filename)
}
