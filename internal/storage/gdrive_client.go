package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TokenProvider interface {
	Acquire() (SAEntry, error)
	ReportResult(saID string, err error)
}

type GDriveClientConfig struct {
	BaseURL       string
	FileID        string
	MaxRetries    int
	BaseBackoffMs int
}

type GDriveClient struct {
	httpClient *http.Client
	cfg        GDriveClientConfig
}

func NewGDriveClient(cfg GDriveClientConfig) *GDriveClient {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BaseBackoffMs <= 0 {
		cfg.BaseBackoffMs = 50
	}
	return &GDriveClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cfg:        cfg,
	}
}

func (c *GDriveClient) FetchRange(ctx context.Context, tp TokenProvider, offset uint64, length uint32) (io.ReadCloser, int64, SAEntry, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, SAEntry{}, err
		}

		sa, err := tp.Acquire()
		if err != nil {
			lastErr = fmt.Errorf("acquire token: %w", err)
			if waitErr := c.waitBackoff(ctx, attempt); waitErr != nil {
				return nil, 0, SAEntry{}, waitErr
			}
			continue
		}

		body, contentLength, err := c.fetchOnce(ctx, sa, offset, length)
		tp.ReportResult(sa.ID, err)
		if err == nil {
			return body, contentLength, sa, nil
		}

		lastErr = err
		if waitErr := c.waitBackoff(ctx, attempt); waitErr != nil {
			return nil, 0, SAEntry{}, waitErr
		}
	}

	return nil, 0, SAEntry{}, fmt.Errorf("fetch range failed: %w", lastErr)
}

func (c *GDriveClient) fetchOnce(ctx context.Context, sa SAEntry, offset uint64, length uint32) (io.ReadCloser, int64, error) {
	if length == 0 {
		return nil, 0, errors.New("length must be greater than 0")
	}
	end := offset + uint64(length) - 1
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/" + c.cfg.FileID + "?alt=media"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Range", "bytes="+strconv.FormatUint(offset, 10)+"-"+strconv.FormatUint(end, 10))
	if sa.Token != "" {
		req.Header.Set("Authorization", "Bearer "+sa.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode == http.StatusPartialContent {
		return resp.Body, resp.ContentLength, nil
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, 0, fmt.Errorf("client error status=%d", resp.StatusCode)
	}
	return nil, 0, fmt.Errorf("upstream error status=%d", resp.StatusCode)
}

func (c *GDriveClient) backoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	maxMs := c.cfg.BaseBackoffMs * (1 << min(attempt, 6))
	jitter := rand.Intn(c.cfg.BaseBackoffMs)
	return time.Duration(maxMs+jitter) * time.Millisecond
}

func (c *GDriveClient) waitBackoff(ctx context.Context, attempt int) error {
	t := time.NewTimer(c.backoff(attempt))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
