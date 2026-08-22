package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var safeFilenameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func ParseSHA256Checksum(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	checksum, err := hex.DecodeString(value)
	if err != nil || len(checksum) != sha256.Size {
		return nil, errors.New("invalid sha256 checksum")
	}

	return checksum, nil
}

func ValidateSafeFilename(filename string) error {
	switch {
	case filename == "":
		return errors.New("no filename provided")
	case filepath.Base(filename) != filename:
		return errors.New("path detected in filename")
	case strings.Contains(filename, ".."):
		return errors.New("invalid filename: path traversal detected")
	case !safeFilenameRegex.MatchString(filename):
		return errors.New("invalid filename: contains invalid characters")
	default:
		return nil
	}
}
