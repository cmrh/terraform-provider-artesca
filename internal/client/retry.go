package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// sendWithTransientRetry sends the request built by newReq and resends it
// after a transient gateway status (502/503/504), with exponential backoff
// starting at backoff. newReq runs once per attempt, so signed requests carry
// a fresh date. It returns the last response's body and status.
func sendWithTransientRetry(ctx context.Context, hc *http.Client, backoff time.Duration, newReq func() (*http.Request, error)) ([]byte, int, error) {
	for attempt := 0; ; attempt++ {
		req, err := newReq()
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}
		resp, err := hc.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("executing request: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		_ = resp.Body.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("reading response: %w", err)
		}
		if !isTransientGatewayStatus(resp.StatusCode) || attempt == transientGatewayMaxAttempts-1 {
			return body, resp.StatusCode, nil
		}
		if err := sleepCtx(ctx, backoff); err != nil {
			return body, resp.StatusCode, err
		}
		if backoff < transientGatewayBackoffMax {
			backoff *= 2
		}
	}
}
