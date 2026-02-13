package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tengjizhang/feed/internal/store"
)

const (
	exitInvalidInput = 2
	exitNotFound     = 3
	exitInternal     = 1
)

func ErrorExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, store.ErrInvalidInput):
		return exitInvalidInput
	case errors.Is(err, store.ErrNotFound):
		return exitNotFound
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid id") || strings.Contains(msg, "invalid output format") {
			return exitInvalidInput
		}
		return exitInternal
	}
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, store.ErrInvalidInput):
		return fmt.Sprintf("Error [invalid-input]: %v", err)
	case errors.Is(err, store.ErrNotFound):
		return fmt.Sprintf("Error [not-found]: %v", err)
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid id") || strings.Contains(msg, "invalid output format") {
			return fmt.Sprintf("Error [invalid-input]: %v", err)
		}
		return fmt.Sprintf("Error [internal]: %v", err)
	}
}

func PrintError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, FormatError(err))
}
