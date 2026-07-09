package api

import (
	"encoding/binary"
	"testing"
)

func TestIsSupportedImageUploadMediaType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		mediaType string
		want      bool
	}{
		{name: "avif", mediaType: "image/avif", want: true},
		{name: "bmp", mediaType: "image/bmp", want: true},
		{name: "jpeg", mediaType: "image/jpeg", want: true},
		{name: "png with spaces and case", mediaType: " Image/PNG ", want: true},
		{name: "gif", mediaType: "image/gif", want: true},
		{name: "heic", mediaType: "image/heic", want: true},
		{name: "heif", mediaType: "image/heif", want: true},
		{name: "svg", mediaType: "image/svg+xml", want: true},
		{name: "tiff", mediaType: "image/tiff", want: true},
		{name: "webp", mediaType: "image/webp", want: true},
		{name: "pdf is not allowed", mediaType: "application/pdf", want: false},
		{name: "empty is not allowed", mediaType: "", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isSupportedImageUploadMediaType(tc.mediaType); got != tc.want {
				t.Fatalf("expected %t for media type %q, got %t", tc.want, tc.mediaType, got)
			}
		})
	}
}

func TestDetectSupportedImageUploadMediaType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		fileBytes []byte
		want      string
	}{
		{
			name:      "detects svg",
			fileBytes: []byte(`<?xml version="1.0"?><svg viewBox="0 0 24 24"></svg>`),
			want:      "image/svg+xml",
		},
		{
			name:      "detects bmp",
			fileBytes: []byte{'B', 'M', 0, 0, 0, 0},
			want:      "image/bmp",
		},
		{
			name:      "detects tiff",
			fileBytes: []byte{'I', 'I', 42, 0, 8, 0},
			want:      "image/tiff",
		},
		{
			name:      "detects avif",
			fileBytes: testISOBMFFFileTypeBox("avif"),
			want:      "image/avif",
		},
		{
			name:      "detects heic",
			fileBytes: testISOBMFFFileTypeBox("heic"),
			want:      "image/heic",
		},
		{
			name:      "detects heif",
			fileBytes: testISOBMFFFileTypeBox("mif1"),
			want:      "image/heif",
		},
		{
			name:      "detects heic when file type box follows a leading box",
			fileBytes: append([]byte{0, 0, 0, 8, 'f', 'r', 'e', 'e'}, testISOBMFFFileTypeBox("test", "heic")...),
			want:      "image/heic",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := detectSupportedImageUploadMediaType(tc.fileBytes)
			if !ok {
				t.Fatalf("expected media type %q to be detected", tc.want)
			}
			if got != tc.want {
				t.Fatalf("expected media type %q, got %q", tc.want, got)
			}
		})
	}
}

func testISOBMFFFileTypeBox(majorBrand string, compatibleBrands ...string) []byte {
	boxSize := 16 + len(compatibleBrands)*4
	fileBytes := make([]byte, boxSize)
	binary.BigEndian.PutUint32(fileBytes[:4], uint32(boxSize))
	copy(fileBytes[4:8], "ftyp")
	copy(fileBytes[8:12], majorBrand)

	for index, brand := range compatibleBrands {
		copy(fileBytes[16+index*4:20+index*4], brand)
	}

	return fileBytes
}
