package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// noSurroundingWhitespace rejects a string carrying leading or trailing
// whitespace.
//
// Tenable.io trims these fields itself. Terraform requires the value applied to
// equal the value planned, so a configuration with a stray space plans "prod "
// and applies "prod", and the apply dies on a mismatch whose message names
// neither the attribute nor the cause. Catching it during validation stops that
// before a single API call is made, and says exactly what to fix.
//
// Trimming silently instead is not available here: a plan modifier may only
// change an attribute that is Computed, and most of these are Required, so a
// modifier that rewrote them would trip Terraform's "planned value does not
// match config value" check. Rejecting is the more predictable contract in any
// case -- what the configuration says is what gets stored.
type noSurroundingWhitespace struct{}

// NoSurroundingWhitespace returns the validator above. Apply it to every string
// attribute whose value is sent to Tenable.io and echoed back.
func NoSurroundingWhitespace() validator.String { return noSurroundingWhitespace{} }

func (noSurroundingWhitespace) Description(_ context.Context) string {
	return "must not begin or end with whitespace"
}

func (v noSurroundingWhitespace) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (noSurroundingWhitespace) ValidateString(
	_ context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	trimmed := strings.TrimSpace(value)
	if trimmed == value {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Leading or Trailing Whitespace",
		fmt.Sprintf(
			"%s is %q, which begins or ends with whitespace.\n\n"+
				"Tenable.io strips the surrounding whitespace when it stores the value, so "+
				"Terraform would plan one string and apply a different one, and the apply "+
				"would fail on the mismatch.\n\n"+
				"Use %q instead.",
			req.Path, value, trimmed,
		),
	)
}

// oneOf rejects a string that is not one of a fixed set of values.
//
// Tenable.io discards or rejects a value outside a setting's documented
// choices, and either way the configuration would never settle; listing the
// choices at validate time says what to write instead.
type oneOf struct{ values []string }

// OneOf returns the validator above.
func OneOf(values ...string) validator.String { return oneOf{values: values} }

func (v oneOf) Description(_ context.Context) string {
	return "must be one of: " + quoteAll(v.values)
}

func (v oneOf) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v oneOf) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	for _, allowed := range v.values {
		if value == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid Value",
		fmt.Sprintf("%s is %q, which is not one of the values Tenable.io accepts: %s.",
			req.Path, value, quoteAll(v.values)))
}

// int64Between rejects an integer outside an inclusive range.
type int64Between struct{ min, max int64 }

// Int64Between returns the validator above.
func Int64Between(minimum, maximum int64) validator.Int64 { return int64Between{min: minimum, max: maximum} }

func (v int64Between) Description(_ context.Context) string {
	return fmt.Sprintf("must be between %d and %d", v.min, v.max)
}

func (v int64Between) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v int64Between) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if value := req.ConfigValue.ValueInt64(); value < v.min || value > v.max {
		resp.Diagnostics.AddAttributeError(req.Path, "Value Out of Range",
			fmt.Sprintf("%s is %d; Tenable.io accepts %d to %d.", req.Path, value, v.min, v.max))
	}
}

func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}
