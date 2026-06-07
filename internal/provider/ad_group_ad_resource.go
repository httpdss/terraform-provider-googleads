package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/httpdss/terraform-provider-googleads/internal/googleads"
)

var _ validator.String

type adGroupAdResource struct{ baseResource }
type rsaModel struct {
	Headlines    types.List `tfsdk:"headlines"`
	Descriptions types.List `tfsdk:"descriptions"`
}
type adGroupAdModel struct {
	ID                 types.String `tfsdk:"id"`
	ResourceName       types.String `tfsdk:"resource_name"`
	AdGroup            types.String `tfsdk:"ad_group"`
	Status             types.String `tfsdk:"status"`
	FinalURLs          types.List   `tfsdk:"final_urls"`
	Path1              types.String `tfsdk:"path1"`
	Path2              types.String `tfsdk:"path2"`
	ResponsiveSearchAd rsaModel     `tfsdk:"responsive_search_ad"`
}

func NewAdGroupAdResource() resource.Resource { return &adGroupAdResource{} }
func (r *adGroupAdResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_ad"
}
func (r *adGroupAdResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Google Ads ad group ad. Version 0.1 supports responsive search ads. Delete uses AdGroupAdService remove.", Attributes: mergeAttrs(idAttrs(), map[string]schema.Attribute{"ad_group": schema.StringAttribute{Required: true}, "status": schema.StringAttribute{Optional: true, Computed: true}, "final_urls": schema.ListAttribute{Required: true, ElementType: types.StringType}, "path1": schema.StringAttribute{Optional: true}, "path2": schema.StringAttribute{Optional: true}, "responsive_search_ad": schema.SingleNestedAttribute{Required: true, Attributes: map[string]schema.Attribute{"headlines": schema.ListAttribute{Required: true, ElementType: types.StringType}, "descriptions": schema.ListAttribute{Required: true, ElementType: types.StringType}}}})}
}
func (r *adGroupAdResource) obj(ctx context.Context, d adGroupAdModel, diags *diag.Diagnostics) map[string]any {
	ad := map[string]any{"finalUrls": stringList(ctx, d.FinalURLs, diags), "responsiveSearchAd": map[string]any{"headlines": textAssets(stringList(ctx, d.ResponsiveSearchAd.Headlines, diags)), "descriptions": textAssets(stringList(ctx, d.ResponsiveSearchAd.Descriptions, diags))}}
	if stringValue(d.Path1) != "" {
		ad["path1"] = stringValue(d.Path1)
	}
	if stringValue(d.Path2) != "" {
		ad["path2"] = stringValue(d.Path2)
	}
	return map[string]any{"adGroup": stringValue(d.AdGroup), "status": def(d.Status, "PAUSED"), "ad": ad}
}
func textAssets(xs []string) []map[string]any {
	out := []map[string]any{}
	for _, x := range xs {
		out = append(out, map[string]any{"text": x})
	}
	return out
}
func (r *adGroupAdResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d adGroupAdModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.Mutate(ctx, "adGroupAds", createOp(r.obj(ctx, d, &resp.Diagnostics)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create ad group ad failed", err)
		return
	}
	rn, err := googleads.ResourceNameFromMutate(out)
	if err != nil {
		resp.Diagnostics.AddError("Create ad group ad failed", err.Error())
		return
	}
	setCreated(rn, &d.ID, &d.ResourceName)
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupAdResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d adGroupAdModel
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
func (r *adGroupAdResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var d adGroupAdModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o := r.obj(ctx, d, &resp.Diagnostics)
	o["resourceName"] = stringValue(d.ResourceName)
	_, err := r.client.Mutate(ctx, "adGroupAds", updateOp(o, []string{"status", "ad.final_urls", "ad.path1", "ad.path2", "ad.responsive_search_ad"}))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update ad group ad failed", err)
		return
	}
	r.read(ctx, &d, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (r *adGroupAdResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var d adGroupAdModel
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Mutate(ctx, "adGroupAds", removeOp(stringValue(d.ResourceName)))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Remove ad group ad failed", err)
	}
}
func (r *adGroupAdResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), req.ID)...)
}
func (r *adGroupAdResource) read(ctx context.Context, d *adGroupAdModel, diags *diag.Diagnostics) bool {
	q := fmt.Sprintf("SELECT ad_group_ad.resource_name, ad_group_ad.ad_group, ad_group_ad.status, ad_group_ad.ad.final_urls, ad_group_ad.ad.path1, ad_group_ad.ad.path2, ad_group_ad.ad.responsive_search_ad.headlines, ad_group_ad.ad.responsive_search_ad.descriptions FROM ad_group_ad WHERE ad_group_ad.resource_name = '%s'", stringValue(d.ResourceName))
	res, err := r.client.Search(ctx, q)
	if err != nil {
		diags.AddError("Read ad group ad failed", err.Error())
		return false
	}
	aga, ok := googleads.First(res, "adGroupAd")
	if !ok {
		return false
	}
	setCreated(googleads.String(aga, "resourceName"), &d.ID, &d.ResourceName)
	d.AdGroup = types.StringValue(googleads.String(aga, "adGroup"))
	d.Status = types.StringValue(googleads.String(aga, "status"))
	if ad, ok := aga["ad"].(map[string]any); ok {
		d.Path1 = types.StringValue(googleads.String(ad, "path1"))
		d.Path2 = types.StringValue(googleads.String(ad, "path2"))
		d.FinalURLs = stringListValue(ctx, stringsFromAny(ad["finalUrls"]), diags)
		if rsa, ok := ad["responsiveSearchAd"].(map[string]any); ok {
			d.ResponsiveSearchAd.Headlines = stringListValue(ctx, textValuesFromAssets(rsa["headlines"]), diags)
			d.ResponsiveSearchAd.Descriptions = stringListValue(ctx, textValuesFromAssets(rsa["descriptions"]), diags)
		}
	}
	return true
}

func stringsFromAny(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func textValuesFromAssets(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		asset, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := asset["text"].(string); ok {
			out = append(out, text)
			continue
		}
		if nested, ok := asset["asset"].(map[string]any); ok {
			if text, ok := nested["text"].(string); ok {
				out = append(out, text)
			}
		}
	}
	return out
}

func stringListValue(ctx context.Context, values []string, diags *diag.Diagnostics) types.List {
	list, listDiags := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(listDiags...)
	if listDiags.HasError() {
		return types.ListNull(types.StringType)
	}
	return list
}
