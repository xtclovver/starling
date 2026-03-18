package validation

import (
	"strings"
	"testing"
)

func TestValidateFile(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
		wantErr     error
	}{
		{
			name:        "valid JPEG",
			contentType: "image/jpeg",
			data:        []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantErr:     nil,
		},
		{
			name:        "valid PNG",
			contentType: "image/png",
			data:        []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantErr:     nil,
		},
		{
			name:        "valid GIF",
			contentType: "image/gif",
			data:        []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
			wantErr:     nil,
		},
		{
			name:        "valid WebP with RIFF header",
			contentType: "image/webp",
			data: func() []byte {
				d := make([]byte, 12)
				copy(d[0:4], []byte{0x52, 0x49, 0x46, 0x46}) // RIFF
				copy(d[8:12], []byte("WEBP"))
				return d
			}(),
			wantErr: nil,
		},
		{
			name:        "valid WebP fallback via WEBP marker at offset 8",
			contentType: "image/webp",
			data: func() []byte {
				d := make([]byte, 12)
				// wrong RIFF magic but WEBP at bytes 8-12
				copy(d[0:4], []byte{0x00, 0x00, 0x00, 0x00})
				copy(d[8:12], []byte("WEBP"))
				return d
			}(),
			wantErr: nil,
		},
		{
			name:        "unsupported content type",
			contentType: "application/pdf",
			data:        []byte{0x25, 0x50, 0x44, 0x46},
			wantErr:     ErrUnsupportedType,
		},
		{
			name:        "unsupported content type text/plain",
			contentType: "text/plain",
			data:        []byte("hello world"),
			wantErr:     ErrUnsupportedType,
		},
		{
			name:        "file too large",
			contentType: "image/jpeg",
			data: func() []byte {
				d := make([]byte, MaxFileSize+1)
				copy(d[0:3], []byte{0xFF, 0xD8, 0xFF})
				return d
			}(),
			wantErr: ErrFileTooLarge,
		},
		{
			name:        "file exactly at max size is allowed",
			contentType: "image/jpeg",
			data: func() []byte {
				d := make([]byte, MaxFileSize)
				copy(d[0:3], []byte{0xFF, 0xD8, 0xFF})
				return d
			}(),
			wantErr: nil,
		},
		{
			name:        "wrong magic bytes - claim JPEG but send PNG bytes",
			contentType: "image/jpeg",
			data:        []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A},
			wantErr:     ErrInvalidMagic,
		},
		{
			name:        "content-type spoofing - claim PNG but send JPEG bytes",
			contentType: "image/png",
			data:        []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantErr:     ErrInvalidMagic,
		},
		{
			name:        "wrong magic bytes - claim GIF but send JPEG bytes",
			contentType: "image/gif",
			data:        []byte{0xFF, 0xD8, 0xFF},
			wantErr:     ErrInvalidMagic,
		},
		{
			name:        "WebP without RIFF header and without WEBP marker",
			contentType: "image/webp",
			data:        []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr:     ErrInvalidMagic,
		},
		{
			name:        "empty data with valid content type",
			contentType: "image/jpeg",
			data:        []byte{},
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFile(tt.contentType, tt.data)
			if err != tt.wantErr {
				t.Errorf("ValidateFile(%q) = %v, want %v", tt.contentType, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateObjectKey(t *testing.T) {
	knownTypes := []struct {
		name        string
		contentType string
		wantExt     string
	}{
		{"JPEG extension", "image/jpeg", "jpg"},
		{"PNG extension", "image/png", "png"},
		{"GIF extension", "image/gif", "gif"},
		{"WebP extension", "image/webp", "webp"},
	}

	for _, tt := range knownTypes {
		t.Run(tt.name, func(t *testing.T) {
			userID := "user-123"
			key := GenerateObjectKey(userID, tt.contentType)

			if !strings.HasPrefix(key, userID+"/") {
				t.Errorf("key %q does not start with %q/", key, userID)
			}
			if !strings.HasSuffix(key, "."+tt.wantExt) {
				t.Errorf("key %q does not end with .%s", key, tt.wantExt)
			}
		})
	}

	t.Run("unknown content type uses second part of MIME", func(t *testing.T) {
		userID := "user-456"
		key := GenerateObjectKey(userID, "video/mp4")

		if !strings.HasPrefix(key, userID+"/") {
			t.Errorf("key %q does not start with %q/", key, userID)
		}
		if !strings.HasSuffix(key, ".mp4") {
			t.Errorf("key %q does not end with .mp4", key)
		}
	})

	t.Run("key format is userID/uuid.ext", func(t *testing.T) {
		userID := "user-789"
		key := GenerateObjectKey(userID, "image/png")

		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			t.Fatalf("expected key with one slash, got %q", key)
		}
		if parts[0] != userID {
			t.Errorf("first part = %q, want %q", parts[0], userID)
		}
		fileParts := strings.SplitN(parts[1], ".", 2)
		if len(fileParts) != 2 {
			t.Fatalf("expected filename with dot, got %q", parts[1])
		}
		// UUID should be 36 characters (8-4-4-4-12 with hyphens)
		if len(fileParts[0]) != 36 {
			t.Errorf("UUID part length = %d, want 36, value = %q", len(fileParts[0]), fileParts[0])
		}
		if fileParts[1] != "png" {
			t.Errorf("extension = %q, want %q", fileParts[1], "png")
		}
	})

	t.Run("each call generates a unique key", func(t *testing.T) {
		key1 := GenerateObjectKey("user", "image/jpeg")
		key2 := GenerateObjectKey("user", "image/jpeg")
		if key1 == key2 {
			t.Errorf("expected unique keys, both were %q", key1)
		}
	})
}
