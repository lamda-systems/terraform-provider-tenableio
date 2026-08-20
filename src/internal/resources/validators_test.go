package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNoSurroundingWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"clean", types.StringValue("Production"), false},
		{"empty", types.StringValue(""), false},
		{"inner spaces are fine", types.StringValue("Production West"), false},
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"trailing space", types.StringValue("Production "), true},
		{"leading space", types.StringValue(" Production"), true},
		{"both", types.StringValue("  Production  "), true},
		{"tab", types.StringValue("Production\t"), true},
		{"newline", types.StringValue("Production\n"), true},
		{"only whitespace", types.StringValue("   "), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			NoSurroundingWhitespace().ValidateString(
				context.Background(),
				validator.StringRequest{
					Path:        path.Root("name"),
					ConfigValue: tt.value,
				},
				resp,
			)
			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("HasError() = %v, want %v (%v)", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestNoSurroundingWhitespaceSuggestsTheFix(t *testing.T) {
	resp := &validator.StringResponse{}
	NoSurroundingWhitespace().ValidateString(
		context.Background(),
		validator.StringRequest{Path: path.Root("name"), ConfigValue: types.StringValue(" prod ")},
		resp,
	)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected a validation error, got: %v", resp.Diagnostics)
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !contains(detail, `"prod"`) {
		t.Errorf("the message should suggest the trimmed value, got: %s", detail)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
