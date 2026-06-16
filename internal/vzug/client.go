package vzug

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	retries    int
	retryDelay time.Duration
}

type Options struct {
	BaseURL          string
	AllowInsecureTLS bool
	Timeout          time.Duration
	Retries          int
	RetryDelay       time.Duration
}

func New(opts Options) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(opts.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.AllowInsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout:   opts.Timeout,
			Transport: transport,
		},
		retries:    opts.Retries,
		retryDelay: opts.RetryDelay,
	}, nil
}

func (c *Client) SetDisplayClock(ctx context.Context, visible bool) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(c.retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		if err := c.setDisplayClockOnce(ctx, visible); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Client) setDisplayClockOnce(ctx context.Context, visible bool) error {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/hh"
	query := endpoint.Query()
	query.Set("command", "setDisplayXclock")
	query.Set("value", strconv.FormatBool(visible))
	query.Set("_", strconv.FormatInt(time.Now().UnixMilli(), 10))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	message := strings.TrimSpace(string(body))
	if message != "" {
		return fmt.Errorf("v-zug returned HTTP %d: %s", res.StatusCode, message)
	}
	return fmt.Errorf("v-zug returned HTTP %d", res.StatusCode)
}
