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

type adGroupKeywordResource struct{ baseResource }
type adGroupKeywordModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceName types.String `tfsdk:"resource_name"`
	AdGroup      types.String `tfsdk:"ad_group"`
	Text         types.String `tfsdk:"text"`
	MatchType    types.String `tfsdk:"match_type"`
	Status       types.String `tfsdk:"status"`
	CPCBidMicros types.Int64  `tfsdk:"cpc_bid_micros"`
	Negative     types.Bool   `tfsdk:"negative"`
}

func NewAdGroupKeywordResource() resource.Resource { return &adGroupKeywordResource{} }
func (r *adGroupKeywordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_keyword"
}
func (r *adGroupKeywordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads ad group keyword criterion. Negative keywords are represented with criterion.negative=true. Delete uses AdGroupCriterionService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{
		"ad_group":       schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{resourceNamePlanModifier("adGroups")}},
		"text":           schema.StringAttribute{Required: true},
		"match_type":     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"status":         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"cpc_bid_micros": schema.Int64Attribute{Optional: true},
		"negative":       schema.BoolAttribute{Optional: true},
	})}
}
func (r *adGroupKeywordResource) normalize(d *adGroupKeywordModel) {
	customerID := clientCustomerID(r.client)
	normalizeResourceState(customerID, "adGroups", &d.AdGroup)
	normalizeResourceState(customerID, "adGroupCriteria", &d.ResourceName)
	normalizeResourceState(customerID, "adGroupCriteria", &d.ID)
	normalizeEnumState(&d.MatchType)
	normalizeEnumState(&d.Status)
}
func (r *adGroupKeywordResource) obj(d adGroupKeywordModel) map[string]any {
	r.normalize(&d)
	o := map[string]any{"adGroup": stringValue(d.AdGroup), "status": def(d.Status, "ENABLED"), "negative": boolValue(d.Negative), "keyword": map[string]any{"text": stringValue(d.Text), "matchType": stringValue(d.MatchType)}}
	if int64Value(d.CPCBidMicros) > 0 {
		o["cpcBidMicros"] = int64Value(d.CPCBidMicros)
	}
	return o
}
func (r *adGroupKeywordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d adGroupKeywordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	out, err := r.client.Mutate(ctx, "adGroupCriteria", createOp(r.obj(d)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create keyword failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create keyword failed", err.Error())
		return
	}
	setCreated(rn, &d.ID, &d.ResourceName)
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupKeywordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d adGroupKeywordModel
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
func (r *adGroupKeywordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var d adGroupKeywordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	o := r.obj(d)
	o["resourceName"] = stringValue(d.ResourceName)
	_, err := r.client.Mutate(ctx, "adGroupCriteria", updateOp(o, []string{"status", "cpc_bid_micros", "negative", "keyword"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update keyword failed", err)
		return
	}
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupKeywordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d adGroupKeywordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "adGroupCriteria", removeOp(stringValue(d.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove keyword failed", err)
	}
}
func (r *adGroupKeywordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *adGroupKeywordResource) read(ctx context.Context, d *adGroupKeywordModel, diags *diag.Diagnostics) bool {
	r.normalize(d)
	q := fmt.Sprintf("SELECT ad_group_criterion.resource_name, ad_group_criterion.ad_group, ad_group_criterion.status, ad_group_criterion.cpc_bid_micros, ad_group_criterion.negative, ad_group_criterion.keyword.text, ad_group_criterion.keyword.match_type FROM ad_group_criterion WHERE ad_group_criterion.resource_name = '%s'", stringValue(d.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read keyword failed", err.Error())
		return false
	}
	c, ok := googleads.First(res, "adGroupCriterion")
	if !ok {
		return false
	}
	setCreated(googleads.String(c, "resourceName"), &d.ID, &d.ResourceName)
	d.AdGroup = types.StringValue(googleads.String(c, "adGroup"))
	d.Status = types.StringValue(googleads.String(c, "status"))
	d.CPCBidMicros = types.Int64Value(googleads.Int64(c, "cpcBidMicros"))
	d.Negative = types.BoolValue(googleads.Bool(c, "negative"))
	if kw, ok := c["keyword"].(map[string]any); ok {
		d.Text = types.StringValue(googleads.String(kw, "text"))
		d.MatchType = types.StringValue(googleads.String(kw, "matchType"))
	}
	return true
}
