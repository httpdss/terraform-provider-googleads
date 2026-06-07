package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/httpdss/terraform-provider-googleads/internal/googleads"
)

type baseResource struct{ client *googleads.Client }

func (r *baseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*googleads.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *googleads.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func idAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}, "resource_name": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}}
}
func mergeAttrs(maps ...map[string]schema.Attribute) map[string]schema.Attribute {
	out := map[string]schema.Attribute{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}
func int64Value(v types.Int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueInt64()
}
func floatValue(v types.Float64) float64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueFloat64()
}
func boolValue(v types.Bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return false
	}
	return v.ValueBool()
}
func strPtr(v types.String) any {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	return v.ValueString()
}
func resourceID(customerID, collection, rn string) string {
	if strings.HasPrefix(rn, "customers/") {
		return rn
	}
	return fmt.Sprintf("customers/%s/%s/%s", customerID, collection, rn)
}
func setCreated(id string, idp *types.String, rnp *types.String) {
	*idp = types.StringValue(id)
	*rnp = types.StringValue(id)
}
func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(summary, err.Error())
}
func removeOp(resourceName string) []map[string]any {
	return []map[string]any{{"remove": resourceName}}
}
func updateOp(obj map[string]any, mask []string) []map[string]any {
	return []map[string]any{{"update": obj, "updateMask": strings.Join(mask, ",")}}
}
func createOp(obj map[string]any) []map[string]any { return []map[string]any{{"create": obj}} }
func stringList(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	var xs []string
	if list.IsNull() || list.IsUnknown() {
		return xs
	}
	diags.Append(list.ElementsAs(ctx, &xs, false)...)
	return xs
}
func stringObjectList(ctx context.Context, list types.List, diags *diag.Diagnostics) []map[string]string {
	var xs []map[string]string
	if list.IsNull() || list.IsUnknown() {
		return xs
	}
	diags.Append(list.ElementsAs(ctx, &xs, false)...)
	return xs
}
func assetList(items []map[string]string) []map[string]any {
	out := []map[string]any{}
	for _, it := range items {
		text := it["text"]
		if text == "" {
			continue
		}
		asset := map[string]any{"text": text}
		if pin := it["pinned_field"]; pin != "" {
			asset["pinnedField"] = pin
		}
		out = append(out, map[string]any{"asset": asset})
	}
	return out
}
