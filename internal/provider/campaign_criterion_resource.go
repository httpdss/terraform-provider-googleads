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

type campaignCriterionResource struct{ baseResource }
type campaignCriterionModel struct {
	ID                        types.String `tfsdk:"id"`
	ResourceName              types.String `tfsdk:"resource_name"`
	Campaign                  types.String `tfsdk:"campaign"`
	Type                      types.String `tfsdk:"type"`
	LocationGeoTargetConstant types.String `tfsdk:"location_geo_target_constant"`
	LanguageConstant          types.String `tfsdk:"language_constant"`
	Negative                  types.Bool   `tfsdk:"negative"`
}

func NewCampaignCriterionResource() resource.Resource { return &campaignCriterionResource{} }
func (r *campaignCriterionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_criterion"
}
func (r *campaignCriterionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads campaign criterion supporting location and language targeting. Delete uses CampaignCriterionService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{
		"campaign":                     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{resourceNamePlanModifier("campaigns")}},
		"type":                         schema.StringAttribute{Required: true, Description: "LOCATION or LANGUAGE.", PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"location_geo_target_constant": schema.StringAttribute{Optional: true, Description: "Geo target constant resource name, e.g. geoTargetConstants/2840 for United States. Numeric IDs such as 2840 are accepted.", PlanModifiers: []planmodifier.String{constantNamePlanModifier("geoTargetConstants")}},
		"language_constant":            schema.StringAttribute{Optional: true, Description: "Language constant resource name, e.g. languageConstants/1000 for English. Numeric IDs such as 1000 are accepted.", PlanModifiers: []planmodifier.String{constantNamePlanModifier("languageConstants")}},
		"negative":                     schema.BoolAttribute{Optional: true},
	})}
}
func (r *campaignCriterionResource) normalize(d *campaignCriterionModel) {
	customerID := clientCustomerID(r.client)
	normalizeResourceState(customerID, "campaigns", &d.Campaign)
	normalizeResourceState(customerID, "campaignCriteria", &d.ResourceName)
	normalizeResourceState(customerID, "campaignCriteria", &d.ID)
	normalizeEnumState(&d.Type)
	normalizeConstantState("geoTargetConstants", &d.LocationGeoTargetConstant)
	normalizeConstantState("languageConstants", &d.LanguageConstant)
}
func (r *campaignCriterionResource) obj(d campaignCriterionModel) map[string]any {
	r.normalize(&d)
	o := map[string]any{"campaign": stringValue(d.Campaign), "negative": boolValue(d.Negative)}
	if stringValue(d.Type) == "LOCATION" {
		o["location"] = map[string]any{"geoTargetConstant": stringValue(d.LocationGeoTargetConstant)}
	} else {
		o["language"] = map[string]any{"languageConstant": stringValue(d.LanguageConstant)}
	}
	return o
}
func (r *campaignCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d campaignCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	out, err := r.client.Mutate(ctx, "campaignCriteria", createOp(r.obj(d)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create campaign criterion failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create campaign criterion failed", err.Error())
		return
	}
	setCreated(rn, &d.ID, &d.ResourceName)
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *campaignCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d campaignCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.read(ctx, &d, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *campaignCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var d campaignCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	o := r.obj(d)
	o["resourceName"] = stringValue(d.ResourceName)
	_, err := r.client.Mutate(ctx, "campaignCriteria", updateOp(o, []string{"negative", "location", "language"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update campaign criterion failed", err)
		return
	}
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *campaignCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d campaignCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "campaignCriteria", removeOp(stringValue(d.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove campaign criterion failed", err)
	}
}
func (r *campaignCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *campaignCriterionResource) read(ctx context.Context, d *campaignCriterionModel, diags *diag.Diagnostics) bool {
	r.normalize(d)
	q := fmt.Sprintf("SELECT campaign_criterion.resource_name, campaign_criterion.campaign, campaign_criterion.negative, campaign_criterion.type, campaign_criterion.location.geo_target_constant, campaign_criterion.language.language_constant FROM campaign_criterion WHERE campaign_criterion.resource_name = '%s'", stringValue(d.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read campaign criterion failed", err.Error())
		return false
	}
	cc, ok := googleads.First(res, "campaignCriterion")
	if !ok {
		return false
	}
	setCreated(googleads.String(cc, "resourceName"), &d.ID, &d.ResourceName)
	d.Campaign = types.StringValue(googleads.String(cc, "campaign"))
	d.Type = types.StringValue(googleads.String(cc, "type"))
	d.Negative = types.BoolValue(googleads.Bool(cc, "negative"))
	if loc, ok := cc["location"].(map[string]any); ok {
		d.LocationGeoTargetConstant = types.StringValue(googleads.String(loc, "geoTargetConstant"))
	}
	if lang, ok := cc["language"].(map[string]any); ok {
		d.LanguageConstant = types.StringValue(googleads.String(lang, "languageConstant"))
	}
	return true
}
