package api

import "testing"

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
			fileBytes: []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f', 0, 0, 0, 0},
			want:      "image/avif",
		},
		{
			name:      "detects heic",
			fileBytes: []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c', 0, 0, 0, 0},
			want:      "image/heic",
		},
		{
			name:      "detects heif",
			fileBytes: []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'm', 'i', 'f', '1', 0, 0, 0, 0},
			want:      "image/heif",
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
