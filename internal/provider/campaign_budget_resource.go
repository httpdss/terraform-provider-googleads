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

type campaignBudgetResource struct{ baseResource }
type campaignBudgetModel struct {
	ID               types.String `tfsdk:"id"`
	ResourceName     types.String `tfsdk:"resource_name"`
	Name             types.String `tfsdk:"name"`
	AmountMicros     types.Int64  `tfsdk:"amount_micros"`
	DeliveryMethod   types.String `tfsdk:"delivery_method"`
	ExplicitlyShared types.Bool   `tfsdk:"explicitly_shared"`
	Status           types.String `tfsdk:"status"`
}

func NewCampaignBudgetResource() resource.Resource { return &campaignBudgetResource{} }
func (r *campaignBudgetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_budget"
}
func (r *campaignBudgetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads campaign budget. Delete uses CampaignBudgetService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{
		"name":              schema.StringAttribute{Required: true},
		"amount_micros":     schema.Int64Attribute{Required: true},
		"delivery_method":   schema.StringAttribute{Optional: true, Computed: true, Description: "STANDARD or ACCELERATED where supported; defaults to STANDARD.", PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
		"explicitly_shared": schema.BoolAttribute{Optional: true, Computed: true},
		"status":            schema.StringAttribute{Optional: true, Computed: true, Description: "ENABLED or REMOVED. Setting REMOVED is equivalent to delete.", PlanModifiers: []planmodifier.String{enumStringPlanModifier()}},
	})}
}
func (r *campaignBudgetResource) normalize(data *campaignBudgetModel) {
	normalizeEnumState(&data.DeliveryMethod)
	normalizeEnumState(&data.Status)
	normalizeResourceState(clientCustomerID(r.client), "campaignBudgets", &data.ResourceName)
	normalizeResourceState(clientCustomerID(r.client), "campaignBudgets", &data.ID)
}
func (r *campaignBudgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data campaignBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&data)
	obj := map[string]any{"name": data.Name.ValueString(), "amountMicros": data.AmountMicros.ValueInt64(), "deliveryMethod": def(data.DeliveryMethod, "STANDARD"), "explicitlyShared": boolValue(data.ExplicitlyShared)}
	out, err := r.client.Mutate(ctx, "campaignBudgets", createOp(obj))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create campaign budget failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create campaign budget failed", err.Error())
		return
	}
	setCreated(rn, &data.ID, &data.ResourceName)
	r.read(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignBudgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data campaignBudgetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found := r.read(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignBudgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data campaignBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.normalize(&data)
	rn := stringValue(data.ResourceName)
	obj := map[string]any{"resourceName": rn, "name": data.Name.ValueString(), "amountMicros": data.AmountMicros.ValueInt64(), "deliveryMethod": def(data.DeliveryMethod, "STANDARD"), "explicitlyShared": boolValue(data.ExplicitlyShared)}
	if stringValue(data.Status) != "" {
		obj["status"] = stringValue(data.Status)
	}
	_, err := r.client.Mutate(ctx, "campaignBudgets", updateOp(obj, []string{"name", "amount_micros", "delivery_method", "explicitly_shared", "status"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update campaign budget failed", err)
		return
	}
	setCreated(rn, &data.ID, &data.ResourceName)
	r.read(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *campaignBudgetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data campaignBudgetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "campaignBudgets", removeOp(stringValue(data.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove campaign budget failed", err)
	}
}
func (r *campaignBudgetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *campaignBudgetResource) read(ctx context.Context, data *campaignBudgetModel, diags *diag.Diagnostics) bool {
	r.normalize(data)
	q := fmt.Sprintf("SELECT campaign_budget.resource_name, campaign_budget.name, campaign_budget.amount_micros, campaign_budget.delivery_method, campaign_budget.explicitly_shared, campaign_budget.status FROM campaign_budget WHERE campaign_budget.resource_name = '%s'", stringValue(data.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read campaign budget failed", err.Error())
		return false
	}
	cb, ok := googleads.First(res, "campaignBudget")
	if !ok {
		return false
	}
	data.Name = types.StringValue(googleads.String(cb, "name"))
	data.AmountMicros = types.Int64Value(googleads.Int64(cb, "amountMicros"))
	data.DeliveryMethod = types.StringValue(googleads.String(cb, "deliveryMethod"))
	data.ExplicitlyShared = types.BoolValue(googleads.Bool(cb, "explicitlyShared"))
	data.Status = types.StringValue(googleads.String(cb, "status"))
	setCreated(googleads.String(cb, "resourceName"), &data.ID, &data.ResourceName)
	return true
}
func def(v types.String, d string) string {
	if stringValue(v) == "" {
		return d
	}
	return stringValue(v)
}
