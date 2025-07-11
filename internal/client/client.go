package client

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

const healthURL = "health/alive"

type Client struct {
	client *http.Client
	root   string
}

func New(root string) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,              // Total idle connections
		MaxIdleConnsPerHost: 100,              // Idle connections per host
		MaxConnsPerHost:     100,              // Total connections per host
		IdleConnTimeout:     90 * time.Second, // Timeout for idle connections
	}
	return &Client{
		client: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
		root: root,
	}
}

func (c *Client) Get(ctx *stopper.Context, path string) ([]byte, error) {
	endpoint, err := url.JoinPath(c.root, path)
	if err != nil {
		return nil, errors.Wrap(err, "endpoint is invalid")
	}
	var resp *http.Response
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = c.client.Get(endpoint)
		if err == nil && statusOK(resp) {
			break
		}
		if attempt < 3 {
			if err != nil {
				log.Printf("retrying request %s %s", endpoint, err.Error())
			} else {
				log.Printf("retrying request %s %s", endpoint, resp.Status)
			}
			select {
			case <-ctx.Stopping():
				return nil, ctx.Err()
			case <-ticker.C:

			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	if !statusOK(resp) {
		return nil, errors.Newf("request failed with status code %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) HealthCheck(ctx *stopper.Context) error {
	endpoint, err := url.JoinPath(c.root, healthURL)
	if err != nil {
		return errors.Wrapf(err, "endpoint is invalid %s", endpoint)
	}
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return errors.Wrapf(err, "health check failed %s", endpoint)
	}
	if statusOK(resp) {
		return nil
	}
	return errors.Newf("healthC check failed with error code %s", resp.Status)
}

func (c *Client) PostForm(ctx *stopper.Context, path string, data url.Values) ([]byte, error) {
	return c.submit(
		ctx,
		http.MethodPost,
		path,
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		[]byte(data.Encode()),
	)
}

func (c *Client) PostJson(ctx *stopper.Context, path string, data []byte) ([]byte, error) {
	return c.submit(
		ctx,
		http.MethodPost,
		path,
		map[string]string{"Content-Type": "application/json"},
		data,
	)
}

func (c *Client) PutJson(ctx *stopper.Context, path string, data []byte) ([]byte, error) {
	return c.submit(
		ctx,
		http.MethodPut,
		path,
		map[string]string{"Content-Type": "application/json"},
		data,
	)
}
func (c *Client) submit(ctx *stopper.Context, method string, path string, headers map[string]string, data []byte) ([]byte, error) {
	endpoint, err := url.JoinPath(c.root, path)
	if err != nil {
		return nil, errors.Wrap(err, "endpoint is invalid")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Wrap(err, "fail to create request")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	var resp *http.Response
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = c.client.Do(req)
		if err == nil && statusOK(resp) {
			break
		}
		if attempt < 3 {
			if err != nil {
				log.Printf("retrying request %s %s", endpoint, err.Error())
				if resp != nil {
					detail, _ := io.ReadAll(resp.Body)
					log.Print(string(detail))
				}
			} else {
				log.Printf("retrying request %s %s", endpoint, resp.Status)
			}
			select {
			case <-ctx.Stopping():
				return nil, ctx.Err()
			case <-ticker.C:

			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	if !statusOK(resp) {
		return nil, errors.Newf("request failed with status code %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func statusOK(res *http.Response) bool {
	if res == nil {
		return false
	}
	if res.StatusCode == http.StatusAccepted ||
		res.StatusCode == http.StatusOK ||
		res.StatusCode == http.StatusCreated {
		return true
	}
	return false
}
