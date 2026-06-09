package googleads

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var secretLikeValue = regexp.MustCompile(`(?i)(developer[_-]?token|client[_-]?secret|refresh[_-]?token|access[_-]?token|authorization|password)([\"'\s:=]+)([^\"'\s,}]+)`)

// GoogleAdsAPIErrorDetail is a parsed entry from a GoogleAdsFailure error payload.
type GoogleAdsAPIErrorDetail struct {
	ErrorCode      string
	Message        string
	Trigger        string
	FieldPath      string
	OperationIndex *int
}

// DiagnosticDetail returns an actionable, redacted Terraform diagnostic detail.
func (e *GoogleAdsError) DiagnosticDetail() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	if e.Message != "" {
		fmt.Fprintf(&b, "Google Ads API returned HTTP %d: %s", e.Status, redactSecrets(e.Message))
	} else {
		fmt.Fprintf(&b, "Google Ads API returned HTTP %d.", e.Status)
	}
	for i, d := range e.Details {
		fmt.Fprintf(&b, "\n\nError %d:", i+1)
		if d.ErrorCode != "" {
			fmt.Fprintf(&b, "\n- code: %s", d.ErrorCode)
		}
		if d.Message != "" {
			fmt.Fprintf(&b, "\n- message: %s", redactSecrets(d.Message))
		}
		if d.Trigger != "" {
			fmt.Fprintf(&b, "\n- trigger: %s", redactSecrets(d.Trigger))
		}
		if d.FieldPath != "" {
			fmt.Fprintf(&b, "\n- field_path: %s", d.FieldPath)
		}
		if d.OperationIndex != nil {
			fmt.Fprintf(&b, "\n- operation_index: %d", *d.OperationIndex)
		}
	}
	if len(e.Details) == 0 && e.Body != "" && e.Message == "" {
		fmt.Fprintf(&b, "\n%s", redactSecrets(e.Body))
	}
	return b.String()
}

func parseGoogleAdsError(b []byte) (string, []GoogleAdsAPIErrorDetail) {
	var raw map[string]any
	if json.Unmarshal(b, &raw) != nil {
		return strings.TrimSpace(redactSecrets(string(b))), nil
	}
	errObj, _ := raw["error"].(map[string]any)
	message, _ := errObj["message"].(string)
	return redactSecrets(message), parseGoogleAdsFailureDetails(errObj)
}

func parseGoogleAdsFailureDetails(errObj map[string]any) []GoogleAdsAPIErrorDetail {
	if errObj == nil {
		return nil
	}
	details, _ := errObj["details"].([]any)
	var parsed []GoogleAdsAPIErrorDetail
	for _, item := range details {
		detail, _ := item.(map[string]any)
		if detail == nil || !strings.Contains(fmt.Sprint(detail["@type"]), "GoogleAdsFailure") {
			continue
		}
		errs, _ := detail["errors"].([]any)
		for _, rawErr := range errs {
			errEntry, _ := rawErr.(map[string]any)
			if errEntry == nil {
				continue
			}
			parsed = append(parsed, GoogleAdsAPIErrorDetail{
				ErrorCode:      parseErrorCode(errEntry["errorCode"]),
				Message:        stringFromAny(errEntry["message"]),
				Trigger:        parseTrigger(errEntry["trigger"]),
				FieldPath:      parseFieldPath(errEntry["location"]),
				OperationIndex: parseOperationIndex(errEntry["location"]),
			})
		}
	}
	return parsed
}

func parseErrorCode(v any) string {
	m, _ := v.(map[string]any)
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s := stringFromAny(m[k]); s != "" {
			return fmt.Sprintf("%s.%s", k, s)
		}
	}
	return ""
}

func parseTrigger(v any) string {
	switch x := v.(type) {
	case map[string]any:
		for _, key := range []string{"stringValue", "int64Value", "floatValue", "booleanValue"} {
			if s := stringFromAny(x[key]); s != "" {
				return s
			}
		}
	case string:
		return x
	}
	return ""
}

func parseFieldPath(location any) string {
	elements := fieldPathElements(location)
	if len(elements) == 0 {
		return ""
	}
	parts := make([]string, 0, len(elements))
	for _, elem := range elements {
		field := stringFromAny(elem["fieldName"])
		if field == "" {
			continue
		}
		if idx, ok := numberAsInt(elem["index"]); ok {
			field = fmt.Sprintf("%s[%d]", field, idx)
		}
		parts = append(parts, field)
	}
	return strings.Join(parts, ".")
}

func parseOperationIndex(location any) *int {
	for _, elem := range fieldPathElements(location) {
		field := stringFromAny(elem["fieldName"])
		if field == "operations" {
			if idx, ok := numberAsInt(elem["index"]); ok {
				return &idx
			}
		}
	}
	return nil
}

func fieldPathElements(location any) []map[string]any {
	loc, _ := location.(map[string]any)
	if loc == nil {
		return nil
	}
	rawElements, _ := loc["fieldPathElements"].([]any)
	out := make([]map[string]any, 0, len(rawElements))
	for _, raw := range rawElements {
		elem, _ := raw.(map[string]any)
		if elem != nil {
			out = append(out, elem)
		}
	}
	return out
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return fmt.Sprintf("%g", x)
	case bool:
		return fmt.Sprintf("%t", x)
	}
	return ""
}

func numberAsInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		return int(i), err == nil
	case int:
		return x, true
	}
	return 0, false
}

func redactSecrets(s string) string {
	return secretLikeValue.ReplaceAllString(s, "$1$2[REDACTED]")
}
