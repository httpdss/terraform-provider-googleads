package googleads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	defaultAPIVersion       = "v23"
	defaultRetryMaxAttempts = 3
	defaultRetryBaseDelay   = 250 * time.Millisecond
	defaultRetryMaxDelay    = 2 * time.Second
)

var customerDigits = regexp.MustCompile(`[^0-9]`)

type Config struct {
	DeveloperToken  string
	CustomerID      string
	LoginCustomerID string
	ClientID        string
	ClientSecret    string
	RefreshToken    string
	TokenFile       string
	CredentialsFile string
	APIVersion      string
	ValidateOnly    bool
}

type Client struct {
	cfg              Config
	httpClient       *http.Client
	baseURL          string
	retryMaxAttempts int
	retryBaseDelay   time.Duration
	retryMaxDelay    time.Duration
}

type GoogleAdsError struct {
	Status          int
	GoogleAdsStatus string
	Message         string
	Body            string
	Details         []GoogleAdsAPIErrorDetail
}

func (e *GoogleAdsError) Error() string {
	return e.DiagnosticDetail()
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	cfg.CustomerID = NormalizeCustomerID(cfg.CustomerID)
	cfg.LoginCustomerID = NormalizeCustomerID(cfg.LoginCustomerID)
	if cfg.APIVersion == "" {
		cfg.APIVersion = defaultAPIVersion
	}
	if cfg.DeveloperToken == "" {
		return nil, errors.New("developer_token is required")
	}
	if cfg.CustomerID == "" {
		return nil, errors.New("customer_id is required")
	}

	oauthCfg, tok, err := oauthConfigAndToken(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg:              cfg,
		httpClient:       oauthCfg.Client(ctx, tok),
		baseURL:          "https://googleads.googleapis.com/" + cfg.APIVersion,
		retryMaxAttempts: defaultRetryMaxAttempts,
		retryBaseDelay:   defaultRetryBaseDelay,
		retryMaxDelay:    defaultRetryMaxDelay,
	}, nil
}

func NormalizeCustomerID(s string) string { return customerDigits.ReplaceAllString(s, "") }
func (c *Client) CustomerID() string      { return c.cfg.CustomerID }
func (c *Client) ValidateOnly() bool      { return c.cfg.ValidateOnly }

func oauthConfigAndToken(ctx context.Context, cfg Config) (*oauth2.Config, *oauth2.Token, error) {
	scopes := []string{"https://www.googleapis.com/auth/adwords"}
	var oc *oauth2.Config
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, nil, fmt.Errorf("read credentials_file: %w", err)
		}
		parsed, err := google.ConfigFromJSON(b, scopes...)
		if err != nil {
			return nil, nil, fmt.Errorf("parse credentials_file: %w", err)
		}
		oc = parsed
	} else {
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, nil, errors.New("client_id and client_secret or credentials_file are required")
		}
		oc = &oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, Endpoint: google.Endpoint, Scopes: scopes, RedirectURL: "urn:ietf:wg:oauth:2.0:oob"}
	}
	if cfg.TokenFile != "" {
		b, err := os.ReadFile(cfg.TokenFile)
		if err != nil {
			return nil, nil, fmt.Errorf("read token_file: %w", err)
		}
		var tok oauth2.Token
		if err := json.Unmarshal(b, &tok); err != nil {
			return nil, nil, fmt.Errorf("parse token_file: %w", err)
		}
		if tok.RefreshToken == "" && cfg.RefreshToken != "" {
			tok.RefreshToken = cfg.RefreshToken
		}
		return oc, &tok, nil
	}
	if cfg.RefreshToken == "" {
		return nil, nil, errors.New("refresh_token or token_file is required")
	}
	return oc, &oauth2.Token{RefreshToken: cfg.RefreshToken}, nil
}

func (c *Client) Mutate(ctx context.Context, collection string, operations any) (map[string]any, error) {
	payload := map[string]any{"operations": operations, "partialFailure": false, "validateOnly": c.cfg.ValidateOnly}
	var out map[string]any
	return out, c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/customers/%s/%s:mutate", c.cfg.CustomerID, collection), payload, &out)
}

func (c *Client) Search(ctx context.Context, query string) ([]map[string]any, error) {
	payload := map[string]any{"query": query, "pageSize": 1000}
	var out struct {
		Results   []map[string]any `json:"results"`
		FieldMask string           `json:"fieldMask"`
	}
	err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/customers/%s/googleAds:search", c.cfg.CustomerID), payload, &out)
	return out.Results, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var payloadBytes []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadBytes = b
	}

	attempts := c.retryMaxAttempts
	if attempts <= 0 {
		attempts = defaultRetryMaxAttempts
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		err := c.doJSONOnce(ctx, method, path, payloadBytes, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == attempts || !isRetryableError(err) {
			return err
		}
		if err := sleepWithContext(ctx, c.retryDelay(attempt)); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *Client) doJSONOnce(ctx context.Context, method, path string, payloadBytes []byte, out any) error {
	var body io.Reader
	if len(payloadBytes) > 0 {
		body = bytes.NewReader(payloadBytes)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("developer-token", c.cfg.DeveloperToken)
	if c.cfg.LoginCustomerID != "" {
		req.Header.Set("login-customer-id", c.cfg.LoginCustomerID)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, googleAdsStatus, details := parseGoogleAdsError(b)
		return &GoogleAdsError{Status: resp.StatusCode, GoogleAdsStatus: googleAdsStatus, Message: msg, Body: string(b), Details: details}
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode google ads response: %w: %s", err, string(b))
		}
	}
	return nil
}

func (c *Client) retryDelay(attempt int) time.Duration {
	base := c.retryBaseDelay
	if base < 0 {
		base = 0
	}
	if base == 0 {
		return 0
	}
	maxDelay := c.retryMaxDelay
	if maxDelay <= 0 {
		maxDelay = defaultRetryMaxDelay
	}
	delay := base << (attempt - 1)
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := 0.5 + rand.Float64()
	return time.Duration(float64(delay) * jitter)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func isRetryableError(err error) bool {
	var apiErr *GoogleAdsError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable()
	}
	return false
}

func ResourceNameFromMutate(resp map[string]any) (string, error) {
	results, ok := resp["results"].([]any)
	if !ok || len(results) == 0 {
		return "", fmt.Errorf("mutate response did not contain results")
	}
	m, ok := results[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected mutate result format")
	}
	if rn, ok := m["resourceName"].(string); ok && rn != "" {
		return rn, nil
	}
	return "", fmt.Errorf("mutate result did not contain resourceName")
}

func First(results []map[string]any, field string) (map[string]any, bool) {
	if len(results) == 0 {
		return nil, false
	}
	v, ok := results[0][field].(map[string]any)
	return v, ok
}

func String(v map[string]any, key string) string {
	if x, ok := v[key].(string); ok {
		return x
	}
	return ""
}
func Bool(v map[string]any, key string) bool {
	if x, ok := v[key].(bool); ok {
		return x
	}
	return false
}
func Float(v map[string]any, key string) float64 {
	if x, ok := v[key].(float64); ok {
		return x
	}
	return 0
}
func Int64(v map[string]any, key string) int64 {
	switch x := v[key].(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case json.Number:
		i, _ := x.Int64()
		return i
	}
	return 0
}

func SaveToken(path string, tok *oauth2.Token) error {
	b, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func AuthCodeURL(cfg *oauth2.Config) string {
	return cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}
func Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return cfg.Exchange(ctx, code)
}
