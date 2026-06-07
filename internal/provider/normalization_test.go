package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeCustomerResourceName(t *testing.T) {
	cases := []struct {
		name       string
		customerID string
		collection string
		input      string
		want       string
	}{
		{name: "full resource strips customer dashes", collection: "campaigns", input: "customers/123-456-7890/campaigns/111", want: "customers/1234567890/campaigns/111"},
		{name: "numeric expands with customer", customerID: "123-456-7890", collection: "adGroups", input: "222", want: "customers/1234567890/adGroups/222"},
		{name: "collection id expands with customer", customerID: "1234567890", collection: "campaignBudgets", input: "campaignBudgets/333", want: "customers/1234567890/campaignBudgets/333"},
		{name: "numeric without customer stays numeric", collection: "campaigns", input: "444", want: "444"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeCustomerResourceName(tc.customerID, tc.collection, tc.input)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeEnumsAndConstants(t *testing.T) {
	if got := normalizeEnum("manual-cpc"); got != "MANUAL_CPC" {
		t.Fatalf("enum got %q", got)
	}
	if got := normalizeConstantName("geoTargetConstants", "2840"); got != "geoTargetConstants/2840" {
		t.Fatalf("constant got %q", got)
	}
}

func TestResourceNamePlanModifierSuppressesNumericDiffWithState(t *testing.T) {
	mod := resourceNamePlanModifier("campaigns")
	req := planmodifier.StringRequest{
		PlanValue:  types.StringValue("111"),
		StateValue: types.StringValue("customers/1234567890/campaigns/111"),
	}
	var resp planmodifier.StringResponse
	mod.PlanModifyString(context.Background(), req, &resp)
	if resp.PlanValue.ValueString() != "customers/1234567890/campaigns/111" {
		t.Fatalf("got %q", resp.PlanValue.ValueString())
	}
}
