package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"
)

func TestCampaignBiddingMapping(t *testing.T) {
	r := &campaignResource{}
	obj := r.campaignObj(campaignModel{Name: types.StringValue("c"), Status: types.StringValue("PAUSED"), AdvertisingChannelType: types.StringValue("SEARCH"), CampaignBudget: types.StringValue("customers/1/campaignBudgets/2"), BiddingStrategyType: types.StringValue("TARGET_CPA"), TargetCPAMicros: types.Int64Value(2500000)})
	if _, ok := obj["targetCpa"]; !ok {
		t.Fatalf("expected targetCpa in %#v", obj)
	}
	if _, ok := obj["manualCpc"]; ok {
		t.Fatalf("did not expect manualCpc in %#v", obj)
	}
}

func TestKeywordMapping(t *testing.T) {
	r := &adGroupKeywordResource{}
	obj := r.obj(adGroupKeywordModel{AdGroup: types.StringValue("customers/1/adGroups/2"), Text: types.StringValue("terraform automation"), MatchType: types.StringValue("PHRASE"), Negative: types.BoolValue(true)})
	kw := obj["keyword"].(map[string]any)
	if kw["matchType"] != "PHRASE" || obj["negative"] != true {
		t.Fatalf("bad keyword mapping: %#v", obj)
	}
}

func TestAdGroupAdImportReadAssetExtraction(t *testing.T) {
	finalURLs := stringsFromAny([]any{"https://example.com", "https://example.org"})
	if len(finalURLs) != 2 || finalURLs[0] != "https://example.com" || finalURLs[1] != "https://example.org" {
		t.Fatalf("unexpected final urls: %#v", finalURLs)
	}

	assets := textValuesFromAssets([]any{
		map[string]any{"text": "Direct REST asset shape"},
		map[string]any{"asset": map[string]any{"text": "Nested create payload shape"}},
	})
	if len(assets) != 2 || assets[0] != "Direct REST asset shape" || assets[1] != "Nested create payload shape" {
		t.Fatalf("unexpected asset text values: %#v", assets)
	}
}
