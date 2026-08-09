package e2b

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/agentclash/agentclash/runtime/sandbox"
)

func normalizeHTTPError(statusCode int, body string, notFoundErr error, retryAfter time.Duration) error {
	switch statusCode {
	case http.StatusNotFound:
		if notFoundErr != nil {
			return notFoundErr
		}
		return fmt.Errorf("e2b resource not found: %s", body)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("e2b authentication failed: %s", body)
	case http.StatusTooManyRequests:
		return sandbox.NewAccountLimitError(retryAfter, body)
	default:
		return fmt.Errorf("e2b request failed with status %d: %s", statusCode, body)
	}
}

func parseRetryAfterHeader(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func normalizeRPCError(err error) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return err
	}
	switch connectErr.Code() {
	case connect.CodeNotFound:
		return sandbox.ErrFileNotFound
	default:
		return fmt.Errorf("e2b rpc failed: %w", err)
	}
}
