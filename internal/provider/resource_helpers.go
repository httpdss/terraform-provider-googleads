package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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

var numericID = regexp.MustCompile(`^[0-9]+$`)

func resourceID(customerID, collection, rn string) string {
	return normalizeCustomerResourceName(customerID, collection, rn)
}

func normalizeEnum(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), "-", "_"))
}

func normalizeCustomerResourceName(customerID, collection, s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	customerID = googleads.NormalizeCustomerID(customerID)
	if strings.HasPrefix(s, "customers/") {
		parts := strings.Split(s, "/")
		if len(parts) >= 4 {
			parts[1] = googleads.NormalizeCustomerID(parts[1])
		}
		return strings.Join(parts, "/")
	}
	if strings.HasPrefix(s, collection+"/") {
		if customerID == "" {
			return s
		}
		return fmt.Sprintf("customers/%s/%s", customerID, s)
	}
	if numericID.MatchString(s) {
		if customerID == "" {
			return s
		}
		return fmt.Sprintf("customers/%s/%s/%s", customerID, collection, s)
	}
	return s
}

func normalizeConstantName(prefix, s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, prefix+"/") {
		return s
	}
	if numericID.MatchString(s) {
		return prefix + "/" + s
	}
	return s
}

func clientCustomerID(c *googleads.Client) string {
	if c == nil {
		return ""
	}
	return c.CustomerID()
}

func normalizeResourceState(customerID, collection string, v *types.String) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return
	}
	*v = types.StringValue(normalizeCustomerResourceName(customerID, collection, v.ValueString()))
}

func normalizeConstantState(prefix string, v *types.String) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return
	}
	*v = types.StringValue(normalizeConstantName(prefix, v.ValueString()))
}

func normalizeEnumState(v *types.String) {
	if v == nil || v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return
	}
	*v = types.StringValue(normalizeEnum(v.ValueString()))
}

func setCreated(id string, idp *types.String, rnp *types.String) {
	*idp = types.StringValue(id)
	*rnp = types.StringValue(id)
}
func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	var apiErr *googleads.GoogleAdsError
	if errors.As(err, &apiErr) {
		diags.AddError(summary, apiErr.DiagnosticDetail())
		return
	}
	diags.AddError(summary, err.Error())
}
func removeOp(resourceName string) []map[string]any {
	return []map[string]any{{"remove": resourceName}}
}
func updateOp(obj map[string]any, mask []string) []map[string]any {
	return []map[string]any{{"update": obj, "updateMask": strings.Join(mask, ",")}}
}
func createOp(obj map[string]any) []map[string]any { return []map[string]any{{"create": obj}} }

type normalizeStringPlanModifier struct {
	description string
	normalize   func(string) string
}

func (m normalizeStringPlanModifier) Description(ctx context.Context) string { return m.description }
func (m normalizeStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.description
}
func (m normalizeStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.StringValue(m.normalize(req.PlanValue.ValueString()))
}

func enumStringPlanModifier() planmodifier.String {
	return normalizeStringPlanModifier{description: "Normalizes enum values to Google Ads uppercase snake case.", normalize: normalizeEnum}
}

type resourceNameStringPlanModifier struct{ collection string }

func (m resourceNameStringPlanModifier) Description(ctx context.Context) string {
	return "Normalizes Google Ads resource names and suppresses diffs for equivalent numeric IDs after creation/import."
}
func (m resourceNameStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m resourceNameStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	planned := normalizeCustomerResourceName("", m.collection, req.PlanValue.ValueString())
	state := ""
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		state = normalizeCustomerResourceName("", m.collection, req.StateValue.ValueString())
	}
	shortID := strings.TrimSpace(req.PlanValue.ValueString())
	if state != "" && numericID.MatchString(shortID) && strings.HasSuffix(state, "/"+m.collection+"/"+shortID) {
		planned = state
	}
	if state != "" && strings.HasPrefix(shortID, m.collection+"/") && strings.HasSuffix(state, "/"+shortID) {
		planned = state
	}
	resp.PlanValue = types.StringValue(planned)
}

func resourceNamePlanModifier(collection string) planmodifier.String {
	return resourceNameStringPlanModifier{collection: collection}
}
func constantNamePlanModifier(prefix string) planmodifier.String {
	return normalizeStringPlanModifier{description: "Normalizes Google Ads constant numeric IDs to resource names.", normalize: func(s string) string { return normalizeConstantName(prefix, s) }}
}

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
