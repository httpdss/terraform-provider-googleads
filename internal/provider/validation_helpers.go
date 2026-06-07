package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type enumStringValidator struct {
	name   string
	values map[string]struct{}
}

func enumValidator(name string, values ...string) validator.String {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[normalizeEnum(value)] = struct{}{}
	}
	return enumStringValidator{name: name, values: allowed}
}

func (v enumStringValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Value must be one of: %s.", strings.Join(v.sortedValues(), ", "))
}

func (v enumStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v enumStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		return
	}
	value := normalizeEnum(req.ConfigValue.ValueString())
	if _, ok := v.values[value]; ok {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Google Ads enum value",
		fmt.Sprintf("%s must be one of %s. Got %q. Values are case-insensitive and hyphenated forms are accepted, for example manual-cpc is normalized to MANUAL_CPC.", v.name, strings.Join(v.sortedValues(), ", "), req.ConfigValue.ValueString()),
	)
}

func (v enumStringValidator) sortedValues() []string {
	values := make([]string, 0, len(v.values))
	for value := range v.values {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

type dateStringValidator struct{}

func dateValidator() validator.String { return dateStringValidator{} }

func (dateStringValidator) Description(ctx context.Context) string {
	return "Value must be a calendar date in YYYY-MM-DD format."
}

func (v dateStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (dateStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		return
	}
	value := strings.TrimSpace(req.ConfigValue.ValueString())
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid date format",
			fmt.Sprintf("Date must use YYYY-MM-DD format, for example 2026-07-01. Got %q.", req.ConfigValue.ValueString()),
		)
	}
}

var (
	statusEnum                 = enumValidator("status", "ENABLED", "PAUSED", "REMOVED")
	budgetStatusEnum           = enumValidator("status", "ENABLED", "REMOVED")
	budgetDeliveryMethodEnum   = enumValidator("delivery_method", "STANDARD", "ACCELERATED")
	advertisingChannelTypeEnum = enumValidator("advertising_channel_type", "SEARCH")
	biddingStrategyTypeEnum    = enumValidator("bidding_strategy_type", "MANUAL_CPC", "TARGET_CPA", "TARGET_ROAS", "MAXIMIZE_CONVERSIONS", "MAXIMIZE_CONVERSION_VALUE")
	adGroupTypeEnum            = enumValidator("type", "SEARCH_STANDARD")
	keywordMatchTypeEnum       = enumValidator("match_type", "BROAD", "PHRASE", "EXACT")
	criterionTypeEnum          = enumValidator("type", "LOCATION", "LANGUAGE")
)

type campaignConfigValidator struct{}

func (campaignConfigValidator) Description(ctx context.Context) string {
	return "Validates Google Ads campaign bidding strategy field combinations."
}
func (v campaignConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (campaignConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data campaignModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateCampaignBiddingConfig(data, &resp.Diagnostics)
}

func validateCampaignBiddingConfig(data campaignModel, diags *diag.Diagnostics) {
	if data.BiddingStrategyType.IsNull() || data.BiddingStrategyType.IsUnknown() {
		return
	}
	switch normalizeEnum(data.BiddingStrategyType.ValueString()) {
	case "TARGET_CPA":
		if data.TargetCPAMicros.IsNull() || data.TargetCPAMicros.IsUnknown() || data.TargetCPAMicros.ValueInt64() <= 0 {
			diags.AddAttributeError(path.Root("target_cpa_micros"), "Missing required bidding strategy field", "target_cpa_micros must be set to a positive value when bidding_strategy_type is TARGET_CPA.")
		}
	case "TARGET_ROAS":
		if data.TargetROAS.IsNull() || data.TargetROAS.IsUnknown() || data.TargetROAS.ValueFloat64() <= 0 {
			diags.AddAttributeError(path.Root("target_roas"), "Missing required bidding strategy field", "target_roas must be set to a positive value when bidding_strategy_type is TARGET_ROAS.")
		}
	}
}

type campaignCriterionConfigValidator struct{}

func (campaignCriterionConfigValidator) Description(ctx context.Context) string {
	return "Validates Google Ads campaign criterion required fields for location and language targeting."
}
func (v campaignCriterionConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (campaignCriterionConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data campaignCriterionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateCampaignCriterionConfig(data, &resp.Diagnostics)
}

func validateCampaignCriterionConfig(data campaignCriterionModel, diags *diag.Diagnostics) {
	if data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}
	switch normalizeEnum(data.Type.ValueString()) {
	case "LOCATION":
		if isEmptyString(data.LocationGeoTargetConstant) {
			diags.AddAttributeError(path.Root("location_geo_target_constant"), "Missing location targeting field", "location_geo_target_constant must be set when type is LOCATION.")
		}
	case "LANGUAGE":
		if isEmptyString(data.LanguageConstant) {
			diags.AddAttributeError(path.Root("language_constant"), "Missing language targeting field", "language_constant must be set when type is LANGUAGE.")
		}
	}
}

func isEmptyString(v types.String) bool {
	return v.IsNull() || v.IsUnknown() || strings.TrimSpace(v.ValueString()) == ""
}
