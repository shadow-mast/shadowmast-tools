package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	ExitOK          = 0
	ExitCheckFailed = 1
	ExitUsageError  = 2
)

const (
	StatusPass = "PASS"
	StatusWarn = "WARN"
	StatusFail = "FAIL"
)

func WriteJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func ExitCodeForStatus(status string) int {
	if strings.EqualFold(status, StatusFail) {
		return ExitCheckFailed
	}
	return ExitOK
}

func UsageError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
