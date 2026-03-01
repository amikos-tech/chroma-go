package downloadutil

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultMaxRedirects = 10
)

// Config controls network and file constraints for downloads.
type Config struct {
	MaxBytes              int64
	DirPerm               os.FileMode
	AllowHTTP             bool
	Timeout               time.Duration
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	Accept                string
	UserAgent             string
}

var newHTTPClientFunc = newHTTPClient

// DownloadFileWithRetry downloads a file with linear retry backoff.
func DownloadFileWithRetry(filePath, sourceURL string, attempts int, cfg Config) error {
	cfg = withDefaults(cfg)
	if cfg.MaxBytes <= 0 {
		return errors.New("max download bytes must be greater than zero")
	}
	parsedURL, err := validateSourceURL(sourceURL, cfg.AllowHTTP)
	if err != nil {
		return err
	}

	if attempts < 1 {
		attempts = 1
	}
	client := newHTTPClientFunc(cfg)
	defer client.CloseIdleConnections()

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := downloadFileWithClient(filePath, parsedURL, cfg, client); err != nil {
			lastErr = err
			if attempt < attempts {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}
		return nil
	}
	if lastErr == nil {
		return errors.New("download failed after retries")
	}
	return fmt.Errorf("download failed after retries: %w", lastErr)
}

// DownloadFile downloads a file to disk via temp file + rename while enforcing max size.
func DownloadFile(filePath, sourceURL string, cfg Config) error {
	return DownloadFileWithRetry(filePath, sourceURL, 1, cfg)
}

func downloadFileWithClient(filePath string, parsedURL *url.URL, cfg Config, client *http.Client) error {
	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	if cfg.Accept != "" {
		req.Header.Set("Accept", cfg.Accept)
	}
	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response %s for URL %s", resp.Status, parsedURL.Redacted())
	}
	if resp.ContentLength > 0 && resp.ContentLength > cfg.MaxBytes {
		return fmt.Errorf(
			"downloaded artifact is too large: %d bytes exceeds max %d bytes",
			resp.ContentLength,
			cfg.MaxBytes,
		)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), cfg.DirPerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	out, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := out.Name()

	limitedBody := io.LimitReader(resp.Body, cfg.MaxBytes+1)
	written, copyErr := io.Copy(out, limitedBody)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to copy HTTP response: %w", copyErr)
	}
	if written > cfg.MaxBytes {
		_ = os.Remove(tempPath)
		return fmt.Errorf(
			"downloaded artifact exceeds max allowed size: got %d bytes, max %d bytes",
			written,
			cfg.MaxBytes,
		)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	if resp.ContentLength > 0 && written != resp.ContentLength {
		_ = os.Remove(tempPath)
		return fmt.Errorf("download incomplete: expected %d bytes, got %d bytes", resp.ContentLength, written)
	}

	_ = os.Remove(filePath)
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to finalize downloaded file: %w", err)
	}
	return nil
}

// RejectHTTPSDowngradeRedirect blocks HTTPS->HTTP redirects and too many redirect hops.
func RejectHTTPSDowngradeRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= defaultMaxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if len(via) == 0 {
		return nil
	}
	previousReq := via[len(via)-1]
	if previousReq.URL != nil && req.URL != nil &&
		strings.EqualFold(previousReq.URL.Scheme, "https") &&
		strings.EqualFold(req.URL.Scheme, "http") {
		return fmt.Errorf(
			"redirect from HTTPS to HTTP is not allowed: %s -> %s",
			previousReq.URL.Redacted(),
			req.URL.Redacted(),
		)
	}
	return nil
}

func withDefaults(cfg Config) Config {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 30 * time.Second
	}
	if cfg.TLSHandshakeTimeout <= 0 {
		cfg.TLSHandshakeTimeout = 10 * time.Second
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = 30 * time.Second
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.DirPerm == 0 {
		cfg.DirPerm = 0700
	}
	return cfg
}

func validateSourceURL(rawURL string, allowHTTP bool) (*url.URL, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid source URL %q: %w", rawURL, err)
	}
	if parsedURL.User != nil {
		return nil, fmt.Errorf("source URL must not contain credentials: %q", parsedURL.Redacted())
	}
	if !parsedURL.IsAbs() {
		return nil, fmt.Errorf("source URL must be absolute: %q", parsedURL.Redacted())
	}
	if strings.TrimSpace(parsedURL.Host) == "" {
		return nil, fmt.Errorf("source URL host cannot be empty: %q", parsedURL.Redacted())
	}
	if strings.EqualFold(parsedURL.Scheme, "https") {
		return parsedURL, nil
	}
	if allowHTTP && strings.EqualFold(parsedURL.Scheme, "http") {
		return parsedURL, nil
	}
	return nil, fmt.Errorf("only HTTPS URLs are supported: %q", parsedURL.Redacted())
}

func newHTTPClient(cfg Config) *http.Client {
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: cfg.DialTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
			ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
			IdleConnTimeout:       cfg.IdleConnTimeout,
		},
		CheckRedirect: RejectHTTPSDowngradeRedirect,
	}
}
