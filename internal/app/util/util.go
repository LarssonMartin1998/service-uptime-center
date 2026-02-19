// Package util
package util

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"service-uptime-center/internal/app/apperror"
)

func ParsePasswordFile(path string) (string, error) {
	if len(path) == 0 {
		slog.Warn("Running without a password file, this is supported but might not be what you intended to do, see --help for more info")
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	pw := strings.TrimSpace(string(data))
	if len(pw) == 0 {
		return "", apperror.ErrPasswordFileIsEmpty
	}

	const MaxPasswordLen = 255
	if len(pw) > MaxPasswordLen {
		return "", fmt.Errorf("%w (max: %d)", apperror.ErrPasswordTooLong, MaxPasswordLen)
	}

	return pw, nil
}
