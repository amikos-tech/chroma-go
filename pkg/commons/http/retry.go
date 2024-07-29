package http

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

type Option func(*SimpleRetryStrategy) error

func WithMaxRetries(retries int) Option {
	return func(r *SimpleRetryStrategy) error {
		if retries <= 0 {
			return fmt.Errorf("retries must be a positive integer")
		}
		r.MaxRetries = retries
		return nil
	}
}

func WithFixedDelay(delay time.Duration) Option {
	return func(r *SimpleRetryStrategy) error {
		if delay <= 0 {
			return fmt.Errorf("delay must be a positive integer")
		}
		r.FixedDelay = delay
		return nil
	}
}

func WithRetryableStatusCodes(statusCodes ...int) Option {
	return func(r *SimpleRetryStrategy) error {
		r.RetryableStatusCodes = statusCodes
		return nil
	}
}

func WithExponentialBackOff() Option {
	return func(r *SimpleRetryStrategy) error {
		r.ExponentialBackOff = true
		return nil
	}
}

type SimpleRetryStrategy struct {
	MaxRetries           int
	FixedDelay           time.Duration
	ExponentialBackOff   bool
	RetryableStatusCodes []int
}

func NewSimpleRetryStrategy(opts ...Option) (*SimpleRetryStrategy, error) {
	var strategy = &SimpleRetryStrategy{
		MaxRetries:           3,
		FixedDelay:           time.Duration(1000) * time.Millisecond,
		RetryableStatusCodes: []int{},
	}
	for _, opt := range opts {
		if err := opt(strategy); err != nil {
			return nil, err
		}
	}
	return strategy, nil
}

func (r *SimpleRetryStrategy) DoWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < r.MaxRetries; i++ {
		resp, err = client.Do(req)
		if err != nil {
			break
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			break
		}
		if r.isRetryable(resp.StatusCode) {
			if r.ExponentialBackOff {
				time.Sleep(r.FixedDelay * time.Duration(math.Pow(2, float64(i))))
			} else {
				time.Sleep(r.FixedDelay)
			}
		}
	}
	return resp, err
}

func (r *SimpleRetryStrategy) isRetryable(code int) bool {
	for _, retryableCode := range r.RetryableStatusCodes {
		if code == retryableCode {
			return true
		}
	}
	return false
}
