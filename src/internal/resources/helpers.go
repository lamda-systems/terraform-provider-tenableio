package resources

import "github.com/hashicorp/terraform-plugin-framework/types"

// readOptionalString preserves null for optional fields:
// if the field was already configured (non-null in state) OR the API returns a
// non-zero value, write the API value. Otherwise keep null to avoid drift.
func readOptionalString(current types.String, apiValue string) types.String {
	if !current.IsNull() || apiValue != "" {
		return types.StringValue(apiValue)
	}
	return types.StringNull()
}

// readOptionalInt64 preserves null for optional int64 fields.
func readOptionalInt64(current types.Int64, apiValue int) types.Int64 {
	if !current.IsNull() || apiValue != 0 {
		return types.Int64Value(int64(apiValue))
	}
	return types.Int64Null()
}
