package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnumValidatorNormalizesAcceptedValues(t *testing.T) {
	t.Parallel()

	v := biddingStrategyTypeEnum
	for _, value := range []string{"MANUAL_CPC", "manual_cpc", "manual-cpc", " target-cpa "} {
		var resp frameworkvalidator.StringResponse
		v.ValidateString(context.Background(), frameworkvalidator.StringRequest{Path: path.Root("bidding_strategy_type"), ConfigValue: types.StringValue(value)}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected %q to be accepted, got diagnostics: %v", value, resp.Diagnostics)
		}
	}
}

func TestEnumValidatorRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	var resp frameworkvalidator.StringResponse
	keywordMatchTypeEnum.ValidateString(context.Background(), frameworkvalidator.StringRequest{Path: path.Root("match_type"), ConfigValue: types.StringValue("phrase-ish")}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid match_type to return an error diagnostic")
	}
	if got := resp.Diagnostics[0].Detail(); !strings.Contains(got, "BROAD") || !strings.Contains(got, "EXACT") || !strings.Contains(got, "PHRASE") {
		t.Fatalf("expected diagnostic to list allowed values, got: %s", got)
	}
}

func TestDateValidatorRequiresYYYYMMDD(t *testing.T) {
	t.Parallel()

	v := dateValidator()
	var valid frameworkvalidator.StringResponse
	v.ValidateString(context.Background(), frameworkvalidator.StringRequest{Path: path.Root("start_date"), ConfigValue: types.StringValue("2026-07-01")}, &valid)
	if valid.Diagnostics.HasError() {
		t.Fatalf("expected valid date, got diagnostics: %v", valid.Diagnostics)
	}

	for _, value := range []string{"2026-7-1", "2026-02-30", "07/01/2026"} {
		var resp frameworkvalidator.StringResponse
		v.ValidateString(context.Background(), frameworkvalidator.StringRequest{Path: path.Root("start_date"), ConfigValue: types.StringValue(value)}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected invalid date %q to return an error diagnostic", value)
		}
	}
}

func TestValidateCampaignBiddingConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data campaignModel
		want bool
	}{
		{name: "target cpa requires target_cpa_micros", data: campaignModel{BiddingStrategyType: types.StringValue("target-cpa")}, want: true},
		{name: "target cpa accepts positive target_cpa_micros", data: campaignModel{BiddingStrategyType: types.StringValue("TARGET_CPA"), TargetCPAMicros: types.Int64Value(1000000)}, want: false},
		{name: "target roas requires target_roas", data: campaignModel{BiddingStrategyType: types.StringValue("TARGET_ROAS")}, want: true},
		{name: "manual cpc does not require target fields", data: campaignModel{BiddingStrategyType: types.StringValue("MANUAL_CPC")}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateCampaignBiddingConfig(tc.data, &diags)
			if got := diags.HasError(); got != tc.want {
				t.Fatalf("HasError() = %v, want %v; diagnostics: %v", got, tc.want, diags)
			}
		})
	}
}

func TestValidateCampaignCriterionConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data campaignCriterionModel
		want bool
	}{
		{name: "location requires geo target", data: campaignCriterionModel{Type: types.StringValue("location")}, want: true},
		{name: "location accepts geo target", data: campaignCriterionModel{Type: types.StringValue("LOCATION"), LocationGeoTargetConstant: types.StringValue("geoTargetConstants/2840")}, want: false},
		{name: "language requires language constant", data: campaignCriterionModel{Type: types.StringValue("LANGUAGE")}, want: true},
		{name: "language accepts language constant", data: campaignCriterionModel{Type: types.StringValue("language"), LanguageConstant: types.StringValue("languageConstants/1000")}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateCampaignCriterionConfig(tc.data, &diags)
			if got := diags.HasError(); got != tc.want {
				t.Fatalf("HasError() = %v, want %v; diagnostics: %v", got, tc.want, diags)
			}
		})
	}
}
