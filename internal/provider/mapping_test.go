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
