package googleads

import (
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

	msg, details := parseGoogleAdsError(body)
	if msg != "Request contains an invalid argument." {
		t.Fatalf("message = %q", msg)
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
