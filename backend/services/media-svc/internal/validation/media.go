package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const MaxFileSize = 10 * 1024 * 1024 // 10MB

var (
	ErrUnsupportedType = errors.New("unsupported content type")
	ErrFileTooLarge    = errors.New("file too large")
	ErrInvalidMagic    = errors.New("file content does not match declared type")
)

var allowedTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
}

var magicBytes = map[string][]byte{
	"image/jpeg": {0xFF, 0xD8, 0xFF},
	"image/png":  {0x89, 0x50, 0x4E, 0x47},
	"image/gif":  {0x47, 0x49, 0x46},
	"image/webp": {0x52, 0x49, 0x46, 0x46},
}

var typeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

func ValidateFile(contentType string, data []byte) error {
	if _, ok := allowedTypes[contentType]; !ok {
		return ErrUnsupportedType
	}
	if len(data) > MaxFileSize {
		return ErrFileTooLarge
	}
	magic, ok := magicBytes[contentType]
	if ok && len(data) >= len(magic) {
		match := true
		for i, b := range magic {
			if data[i] != b {
				match = false
				break
			}
		}
		if !match {
			// WebP has magic at offset 8 for "WEBP"
			if contentType == "image/webp" && len(data) >= 12 {
				if string(data[8:12]) == "WEBP" {
					return nil
				}
			}
			return ErrInvalidMagic
		}
	}
	return nil
}

func GenerateObjectKey(userID, contentType string) string {
	ext := typeToExt[contentType]
	if ext == "" {
		parts := strings.SplitN(contentType, "/", 2)
		if len(parts) == 2 {
			ext = parts[1]
		} else {
			ext = "bin"
		}
	}
	return fmt.Sprintf("%s/%s.%s", userID, uuid.New().String(), ext)
}
