package provider

import (
	"os"
	"testing"
)

// Acceptance tests require a real Google Ads account and credentials. This file
// documents the test entry point without running destructive API calls by default.
// Future resource-level acceptance tests should be guarded by TF_ACC=1 and should
// create PAUSED campaigns with small budgets.
func TestAccProviderRequiresCredentials(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 and Google Ads credentials to run acceptance tests")
	}
	for _, env := range []string{"GOOGLEADS_DEVELOPER_TOKEN", "GOOGLEADS_CUSTOMER_ID"} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for acceptance tests", env)
		}
	}
}
