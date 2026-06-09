package googleads

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeCustomerID(t *testing.T) {
	got := NormalizeCustomerID("123-456-7890")
	if got != "1234567890" {
		t.Fatalf("NormalizeCustomerID() = %q", got)
	}
}

func TestResourceNameFromMutate(t *testing.T) {
	rn, err := ResourceNameFromMutate(map[string]any{"results": []any{map[string]any{"resourceName": "customers/1/campaigns/2"}}})
	if err != nil {
		t.Fatal(err)
	}
	if rn != "customers/1/campaigns/2" {
		t.Fatalf("resource name = %q", rn)
	}
}

func TestParseGoogleAdsFailureDetails(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 400,
			"message": "Request contains an invalid argument.",
			"status": "INVALID_ARGUMENT",
			"details": [{
				"@type": "type.googleapis.com/google.ads.googleads.v23.errors.GoogleAdsFailure",
				"errors": [{
					"errorCode": {"criterionError": "INVALID_KEYWORD_TEXT"},
					"message": "Keyword text should not be empty.",
					"trigger": {"stringValue": ""},
					"location": {"fieldPathElements": [
						{"fieldName": "operations", "index": 0},
						{"fieldName": "create"},
						{"fieldName": "keyword"},
						{"fieldName": "text"}
					]}
				}]
			}]
		}
	}`)

	msg, status, details := parseGoogleAdsError(body)
	if msg != "Request contains an invalid argument." {
		t.Fatalf("message = %q", msg)
	}
	if status != "INVALID_ARGUMENT" {
		t.Fatalf("status = %q", status)
	}
	if len(details) != 1 {
		t.Fatalf("details length = %d", len(details))
	}
	d := details[0]
	if d.ErrorCode != "criterionError.INVALID_KEYWORD_TEXT" {
		t.Fatalf("error code = %q", d.ErrorCode)
	}
	if d.Message != "Keyword text should not be empty." {
		t.Fatalf("detail message = %q", d.Message)
	}
	if d.FieldPath != "operations[0].create.keyword.text" {
		t.Fatalf("field path = %q", d.FieldPath)
	}
	if d.OperationIndex == nil || *d.OperationIndex != 0 {
		t.Fatalf("operation index = %v", d.OperationIndex)
	}
}

func TestGoogleAdsErrorDiagnosticDetailIncludesActionableFields(t *testing.T) {
	operationIndex := 2
	err := &GoogleAdsError{
		Status:  400,
		Message: "Request contains an invalid argument.",
		Details: []GoogleAdsAPIErrorDetail{{
			ErrorCode:      "fieldError.REQUIRED",
			Message:        "The required field was not present.",
			Trigger:        "campaign_budget",
			FieldPath:      "operations[2].create.campaignBudget",
			OperationIndex: &operationIndex,
		}},
	}

	detail := err.DiagnosticDetail()
	for _, want := range []string{
		"HTTP 400",
		"fieldError.REQUIRED",
		"The required field was not present.",
		"campaign_budget",
		"operations[2].create.campaignBudget",
		"operation_index: 2",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("diagnostic detail missing %q:\n%s", want, detail)
		}
	}
}

func TestGoogleAdsErrorDiagnosticDetailRedactsSecrets(t *testing.T) {
	err := &GoogleAdsError{
		Status:  400,
		Message: "developer_token: abc123 client_secret=shh",
		Details: []GoogleAdsAPIErrorDetail{{Message: "refresh_token: xyz", Trigger: "access_token=secret"}},
	}
	detail := err.DiagnosticDetail()
	for _, secretValue := range []string{"abc123", "shh", "xyz"} {
		if strings.Contains(detail, secretValue) {
			t.Fatalf("diagnostic leaked secret value %q: %s", secretValue, detail)
		}
	}
	if got := strings.Count(detail, "[REDACTED]"); got < 4 {
		t.Fatalf("expected redacted placeholders, got %d in: %s", got, detail)
	}
}

func TestDoJSONRetriesHTTPTransientStatus(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if got := r.Header.Get("developer-token"); got != "dev-token" {
			t.Fatalf("developer-token header = %q", got)
		}
		if attempts < 3 {
			http.Error(w, `{"error":{"message":"rate limited","status":"RESOURCE_EXHAUSTED"}}`, http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer ts.Close()

	client := testRetryClient(ts)
	var out map[string]any
	if err := client.doJSON(context.Background(), http.MethodPost, "/test", map[string]any{"x": "y"}, &out); err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if out["ok"] != true {
		t.Fatalf("decoded response = %#v", out)
	}
}

func TestDoJSONRetriesGoogleAdsTransientStatus(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"backend unavailable","status":"UNAVAILABLE"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	client := testRetryClient(ts)
	var out map[string]any
	if err := client.doJSON(context.Background(), http.MethodPost, "/test", nil, &out); err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestDoJSONDoesNotRetryValidationErrors(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid field","status":"INVALID_ARGUMENT"}}`))
	}))
	defer ts.Close()

	client := testRetryClient(ts)
	err := client.doJSON(context.Background(), http.MethodPost, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected non-retryable error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestGoogleAdsErrorRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  *GoogleAdsError
		want bool
	}{
		{name: "too many requests", err: &GoogleAdsError{Status: http.StatusTooManyRequests}, want: true},
		{name: "server error", err: &GoogleAdsError{Status: http.StatusBadGateway}, want: true},
		{name: "resource exhausted", err: &GoogleAdsError{Status: http.StatusBadRequest, GoogleAdsStatus: "RESOURCE_EXHAUSTED"}, want: true},
		{name: "unavailable", err: &GoogleAdsError{Status: http.StatusBadRequest, GoogleAdsStatus: "UNAVAILABLE"}, want: true},
		{name: "deadline exceeded", err: &GoogleAdsError{Status: http.StatusBadRequest, GoogleAdsStatus: "DEADLINE_EXCEEDED"}, want: true},
		{name: "invalid argument", err: &GoogleAdsError{Status: http.StatusBadRequest, GoogleAdsStatus: "INVALID_ARGUMENT"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Retryable(); got != tc.want {
				t.Fatalf("Retryable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func testRetryClient(ts *httptest.Server) *Client {
	return &Client{
		cfg:              Config{DeveloperToken: "dev-token", CustomerID: "1234567890"},
		httpClient:       ts.Client(),
		baseURL:          ts.URL,
		retryMaxAttempts: 3,
		retryBaseDelay:   0,
		retryMaxDelay:    0,
	}
}
