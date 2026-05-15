package execx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func Run(ctx context.Context, timeout time.Duration, name string, args ...string) (Result, error) {
	return RunAllowExitCodes(ctx, timeout, nil, name, args...)
}

func RunAllowExitCodes(ctx context.Context, timeout time.Duration, allowedExitCodes []int, name string, args ...string) (Result, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			for _, allowed := range allowedExitCodes {
				if result.ExitCode == allowed {
					return result, nil
				}
			}
		}
		return result, fmt.Errorf("%s failed: %w", name, err)
	}
	return result, nil
}
