package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/httpdss/terraform-provider-googleads/internal/googleads"
)

type adGroupResource struct{ baseResource }
type adGroupModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceName types.String `tfsdk:"resource_name"`
	Campaign     types.String `tfsdk:"campaign"`
	Name         types.String `tfsdk:"name"`
	Status       types.String `tfsdk:"status"`
	Type         types.String `tfsdk:"type"`
	CPCBidMicros types.Int64  `tfsdk:"cpc_bid_micros"`
}

func NewAdGroupResource() resource.Resource { return &adGroupResource{} }
func (r *adGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group"
}
func (r *adGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads ad group. Delete uses AdGroupService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{
		"campaign":       schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{resourceNamePlanModifier("campaigns")}},
		"name":           schema.StringAttribute{Required: true},
		"status":         schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{statusEnum}, PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"type":           schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{adGroupTypeEnum}, PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"cpc_bid_micros": schema.Int64Attribute{Optional: true},
	})}
}
func (r *adGroupResource) normalize(d *adGroupModel) {
	customerID := clientCustomerID(r.client)
	normalizeResourceState(customerID, "campaigns", &d.Campaign)
	normalizeResourceState(customerID, "adGroups", &d.ResourceName)
	normalizeResourceState(customerID, "adGroups", &d.ID)
	normalizeEnumState(&d.Status)
	normalizeEnumState(&d.Type)
}
func (r *adGroupResource) obj(d adGroupModel) map[string]any {
	r.normalize(&d)
	o := map[string]any{"campaign": stringValue(d.Campaign), "name": stringValue(d.Name), "status": def(d.Status, "PAUSED"), "type": def(d.Type, "SEARCH_STANDARD")}
	if int64Value(d.CPCBidMicros) > 0 {
		o["cpcBidMicros"] = int64Value(d.CPCBidMicros)
	}
	return o
}
func (r *adGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d adGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	out, err := r.client.Mutate(ctx, "adGroups", createOp(r.obj(d)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create ad group failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create ad group failed", err.Error())
		return
	}
	setCreated(rn, &d.ID, &d.ResourceName)
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d adGroupModel
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
func (r *adGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var d adGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&d)
	o := r.obj(d)
	o["resourceName"] = stringValue(d.ResourceName)
	_, err := r.client.Mutate(ctx, "adGroups", updateOp(o, []string{"name", "status", "type", "cpc_bid_micros"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update ad group failed", err)
		return
	}
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "adGroups", removeOp(stringValue(d.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove ad group failed", err)
	}
}
func (r *adGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *adGroupResource) read(ctx context.Context, d *adGroupModel, diags *diag.Diagnostics) bool {
	r.normalize(d)
	q := fmt.Sprintf("SELECT ad_group.resource_name, ad_group.campaign, ad_group.name, ad_group.status, ad_group.type, ad_group.cpc_bid_micros FROM ad_group WHERE ad_group.resource_name = '%s'", stringValue(d.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read ad group failed", err.Error())
		return false
	}
	ag, ok := googleads.First(res, "adGroup")
	if !ok {
		return false
	}
	setCreated(googleads.String(ag, "resourceName"), &d.ID, &d.ResourceName)
	d.Campaign = types.StringValue(googleads.String(ag, "campaign"))
	d.Name = types.StringValue(googleads.String(ag, "name"))
	d.Status = types.StringValue(googleads.String(ag, "status"))
	d.Type = types.StringValue(googleads.String(ag, "type"))
	d.CPCBidMicros = types.Int64Value(googleads.Int64(ag, "cpcBidMicros"))
	return true
}
