package resources

import "github.com/hashicorp/terraform-plugin-framework/types"

// setOptional adds a key to the body map if the value is not null.
func setOptional(body map[string]interface{}, key string, val types.String) {
	if !val.IsNull() && !val.IsUnknown() {
		body[key] = val.ValueString()
	}
}

// getString safely extracts a string from a map.
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// getOptionalString extracts a string as types.String, null if missing.
func getOptionalString(data map[string]interface{}, key string) types.String {
	if v, ok := data[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}
