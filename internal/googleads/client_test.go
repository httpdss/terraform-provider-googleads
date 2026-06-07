package googleads

import "testing"

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
