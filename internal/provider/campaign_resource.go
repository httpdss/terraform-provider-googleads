package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/httpdss/terraform-provider-googleads/internal/googleads"
)

type campaignResource struct{ baseResource }
type networkSettingsModel struct {
	TargetGoogleSearch         types.Bool `tfsdk:"target_google_search"`
	TargetSearchNetwork        types.Bool `tfsdk:"target_search_network"`
	TargetContentNetwork       types.Bool `tfsdk:"target_content_network"`
	TargetPartnerSearchNetwork types.Bool `tfsdk:"target_partner_search_network"`
}
type campaignModel struct {
	ID                     types.String         `tfsdk:"id"`
	ResourceName           types.String         `tfsdk:"resource_name"`
	Name                   types.String         `tfsdk:"name"`
	Status                 types.String         `tfsdk:"status"`
	AdvertisingChannelType types.String         `tfsdk:"advertising_channel_type"`
	CampaignBudget         types.String         `tfsdk:"campaign_budget"`
	StartDate              types.String         `tfsdk:"start_date"`
	EndDate                types.String         `tfsdk:"end_date"`
	BiddingStrategyType    types.String         `tfsdk:"bidding_strategy_type"`
	TargetCPAMicros        types.Int64          `tfsdk:"target_cpa_micros"`
	TargetROAS             types.Float64        `tfsdk:"target_roas"`
	NetworkSettings        networkSettingsModel `tfsdk:"network_settings"`
	TrackingURLTemplate    types.String         `tfsdk:"tracking_url_template"`
	FinalURLSuffix         types.String         `tfsdk:"final_url_suffix"`
}

func NewCampaignResource() resource.Resource { return &campaignResource{} }
func (r *campaignResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign"
}
func (r *campaignResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads campaign. Delete uses CampaignService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{
		"name":                     schema.StringAttribute{Required: true},
		"status":                   schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"advertising_channel_type": schema.StringAttribute{Optional: true, Computed: true, Description: "Defaults to SEARCH.", PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"campaign_budget":          schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{resourceNamePlanModifier("campaignBudgets")}},
		"start_date":               schema.StringAttribute{Optional: true},
		"end_date":                 schema.StringAttribute{Optional: true},
		"bidding_strategy_type":    schema.StringAttribute{Optional: true, Computed: true, Description: "MANUAL_CPC, TARGET_CPA, TARGET_ROAS, etc.", PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"target_cpa_micros":        schema.Int64Attribute{Optional: true},
		"target_roas":              schema.Float64Attribute{Optional: true},
		"tracking_url_template":    schema.StringAttribute{Optional: true},
		"final_url_suffix":         schema.StringAttribute{Optional: true},
		"network_settings": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
			"target_google_search":          schema.BoolAttribute{Optional: true},
			"target_search_network":         schema.BoolAttribute{Optional: true},
			"target_content_network":        schema.BoolAttribute{Optional: true},
			"target_partner_search_network": schema.BoolAttribute{Optional: true},
		}},
	})}
}
func (r *campaignResource) normalize(data *campaignModel) {
	customerID := clientCustomerID(r.client)
	normalizeEnumState(&data.Status)
	normalizeEnumState(&data.AdvertisingChannelType)
	normalizeEnumState(&data.BiddingStrategyType)
	normalizeResourceState(customerID, "campaignBudgets", &data.CampaignBudget)
	normalizeResourceState(customerID, "campaigns", &data.ResourceName)
	normalizeResourceState(customerID, "campaigns", &data.ID)
}

func (r *campaignResource) campaignObj(data campaignModel) map[string]any {
	r.normalize(&data)
	obj := map[string]any{"name": stringValue(data.Name), "status": def(data.Status, "PAUSED"), "advertisingChannelType": def(data.AdvertisingChannelType, "SEARCH"), "campaignBudget": stringValue(data.CampaignBudget)}
	if stringValue(data.StartDate) != "" {
		obj["startDate"] = stringValue(data.StartDate)
	}
	if stringValue(data.EndDate) != "" {
		obj["endDate"] = stringValue(data.EndDate)
	}
	if stringValue(data.TrackingURLTemplate) != "" {
		obj["trackingUrlTemplate"] = stringValue(data.TrackingURLTemplate)
	}
	if stringValue(data.FinalURLSuffix) != "" {
		obj["finalUrlSuffix"] = stringValue(data.FinalURLSuffix)
	}
	ns := map[string]any{"targetGoogleSearch": boolValue(data.NetworkSettings.TargetGoogleSearch), "targetSearchNetwork": boolValue(data.NetworkSettings.TargetSearchNetwork), "targetContentNetwork": boolValue(data.NetworkSettings.TargetContentNetwork), "targetPartnerSearchNetwork": boolValue(data.NetworkSettings.TargetPartnerSearchNetwork)}
	obj["networkSettings"] = ns
	switch def(data.BiddingStrategyType, "MANUAL_CPC") {
	case "MANUAL_CPC":
		obj["manualCpc"] = map[string]any{}
	case "TARGET_CPA":
		obj["targetCpa"] = map[string]any{"targetCpaMicros": int64Value(data.TargetCPAMicros)}
	case "TARGET_ROAS":
		obj["targetRoas"] = map[string]any{"targetRoas": floatValue(data.TargetROAS)}
	case "MAXIMIZE_CONVERSIONS":
		obj["maximizeConversions"] = map[string]any{}
	case "MAXIMIZE_CONVERSION_VALUE":
		obj["maximizeConversionValue"] = map[string]any{}
	default:
		obj["manualCpc"] = map[string]any{}
	}
	return obj
}
func (r *campaignResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data campaignModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&data)
	out, err := r.client.Mutate(ctx, "campaigns", createOp(r.campaignObj(data)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create campaign failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create campaign failed", err.Error())
		return
	}
	setCreated(rn, &data.ID, &data.ResourceName)
	r.read(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.read(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data campaignModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&data)
	obj := r.campaignObj(data)
	obj["resourceName"] = stringValue(data.ResourceName)
	_, err := r.client.Mutate(ctx, "campaigns", updateOp(obj, []string{"name", "status", "campaign_budget", "start_date", "end_date", "network_settings", "tracking_url_template", "final_url_suffix", "manual_cpc", "target_cpa", "target_roas", "maximize_conversions", "maximize_conversion_value"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update campaign failed", err)
		return
	}
	r.read(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "campaigns", removeOp(stringValue(data.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove campaign failed", err)
	}
}
func (r *campaignResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *campaignResource) read(ctx context.Context, data *campaignModel, diags *diag.Diagnostics) bool {
	r.normalize(data)
	q := fmt.Sprintf("SELECT campaign.resource_name, campaign.name, campaign.status, campaign.advertising_channel_type, campaign.campaign_budget, campaign.start_date, campaign.end_date, campaign.bidding_strategy_type, campaign.network_settings.target_google_search, campaign.network_settings.target_search_network, campaign.network_settings.target_content_network, campaign.network_settings.target_partner_search_network, campaign.tracking_url_template, campaign.final_url_suffix FROM campaign WHERE campaign.resource_name = '%s'", stringValue(data.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read campaign failed", err.Error())
		return false
	}
	c, ok := googleads.First(res, "campaign")
	if !ok {
		return false
	}
	setCreated(googleads.String(c, "resourceName"), &data.ID, &data.ResourceName)
	data.Name = types.StringValue(googleads.String(c, "name"))
	data.Status = types.StringValue(googleads.String(c, "status"))
	data.AdvertisingChannelType = types.StringValue(googleads.String(c, "advertisingChannelType"))
	data.CampaignBudget = types.StringValue(googleads.String(c, "campaignBudget"))
	data.StartDate = types.StringValue(googleads.String(c, "startDate"))
	data.EndDate = types.StringValue(googleads.String(c, "endDate"))
	data.BiddingStrategyType = types.StringValue(googleads.String(c, "biddingStrategyType"))
	data.TrackingURLTemplate = types.StringValue(googleads.String(c, "trackingUrlTemplate"))
	data.FinalURLSuffix = types.StringValue(googleads.String(c, "finalUrlSuffix"))
	if ns, ok := c["networkSettings"].(map[string]any); ok {
		data.NetworkSettings.TargetGoogleSearch = types.BoolValue(googleads.Bool(ns, "targetGoogleSearch"))
		data.NetworkSettings.TargetSearchNetwork = types.BoolValue(googleads.Bool(ns, "targetSearchNetwork"))
		data.NetworkSettings.TargetContentNetwork = types.BoolValue(googleads.Bool(ns, "targetContentNetwork"))
		data.NetworkSettings.TargetPartnerSearchNetwork = types.BoolValue(googleads.Bool(ns, "targetPartnerSearchNetwork"))
	}
	return true
}
