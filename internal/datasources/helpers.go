package datasources

import "github.com/hashicorp/terraform-plugin-framework/types"

func getOptionalString(data map[string]interface{}, key string) types.String {
	if v, ok := data[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}
