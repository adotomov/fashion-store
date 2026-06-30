package storage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// storageScope is the OAuth2 scope needed for real GCS object/bucket
// read-write calls.
const storageScope = "https://www.googleapis.com/auth/devstorage.read_write"

// Client talks to a GCS-compatible JSON API (real GCS or, for local
// devbox, fsouza/fake-gcs-server) over plain HTTP requests rather than the
// official Cloud Storage SDK, since the fake server's self-signed TLS cert
// and lack of real signed-URL support make the full SDK more friction than
// it's worth for this scope. Business logic depends on the application
// layer's MediaStorage port, not on this client directly.
type Client struct {
	httpClient *http.Client
	baseURL    string
	useAuth    bool

	tokenOnce   sync.Once
	tokenSource oauth2.TokenSource
	tokenErr    error
}

func NewClient(endpoint string, insecureSkipTLS bool) *Client {
	transport := &http.Transport{}
	if insecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // local devbox FakeGCS only
	}
	return &Client{
		httpClient: &http.Client{Transport: transport},
		baseURL:    strings.TrimSuffix(endpoint, "/"),
		// FakeGCS (devbox) accepts unauthenticated requests; real GCS
		// requires a bearer token on every call. insecureSkipTLS is also
		// how the local-vs-real endpoint is already distinguished, so
		// reuse it rather than adding a third config flag.
		useAuth: !insecureSkipTLS,
	}
}

// authorize attaches an Application Default Credentials bearer token to req
// when talking to real GCS. ADC picks up Cloud Run's attached service
// account with no key file needed. No-op against the local FakeGCS server.
func (c *Client) authorize(ctx context.Context, req *http.Request) error {
	if !c.useAuth {
		return nil
	}
	c.tokenOnce.Do(func() {
		c.tokenSource, c.tokenErr = google.DefaultTokenSource(ctx, storageScope)
	})
	if c.tokenErr != nil {
		return fmt.Errorf("resolve storage credentials: %w", c.tokenErr)
	}
	token, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("get storage access token: %w", err)
	}
	token.SetAuthHeader(req)
	return nil
}

// EnsureBucket creates the bucket if it doesn't already exist. Idempotent.
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	body := strings.NewReader(fmt.Sprintf(`{"name":%q}`, bucket))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/storage/v1/b", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ensure bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("ensure bucket: unexpected status %d", resp.StatusCode)
	}
	return nil
}

type uploadResponse struct {
	Size string `json:"size"`
}

// Upload writes content to bucket/objectKey and returns the stored size in
// bytes (as reported by the server, not trusted from the caller).
func (c *Client) Upload(ctx context.Context, bucket, objectKey, contentType string, content io.Reader) (int64, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return 0, fmt.Errorf("read upload content: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		c.baseURL, url.PathEscape(bucket), url.QueryEscape(objectKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, strings.NewReader(string(data)))
	if err != nil {
		return 0, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if err := c.authorize(ctx, req); err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("upload object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("upload object: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	var parsed uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return int64(len(data)), nil //nolint:nilerr // size fallback, upload itself succeeded
	}

	size, err := strconv.ParseInt(parsed.Size, 10, 64)
	if err != nil {
		return int64(len(data)), nil //nolint:nilerr // size fallback, upload itself succeeded
	}
	return size, nil
}

// Open returns a reader for the object's content and its content type.
// Callers must close the returned reader.
func (c *Client) Open(ctx context.Context, bucket, objectKey string) (io.ReadCloser, string, error) {
	downloadURL := fmt.Sprintf("%s/storage/v1/b/%s/o/%s?alt=media",
		c.baseURL, url.PathEscape(bucket), url.PathEscape(objectKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", err
	}
	if err := c.authorize(ctx, req); err != nil {
		return nil, "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("open object: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("open object: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func (c *Client) Delete(ctx context.Context, bucket, objectKey string) error {
	deleteURL := fmt.Sprintf("%s/storage/v1/b/%s/o/%s",
		c.baseURL, url.PathEscape(bucket), url.PathEscape(objectKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	if err := c.authorize(ctx, req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("delete object: unexpected status %d", resp.StatusCode)
	}
	return nil
}
