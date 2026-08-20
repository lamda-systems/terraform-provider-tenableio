package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// readOptionalString maps an optional string from an API response into state.
// A field already present in state, or one the API reports a value for, takes
// the API value; otherwise it stays null so an unset attribute does not drift.
//
// Adopting the API's value when state is null is what makes import work, but it
// is only safe for attributes declared Optional *and* Computed. For an
// Optional-only attribute that Tenable.io fills in by itself, it writes a value
// into state that configuration never asked for, and every subsequent plan
// proposes removing it. If a field turns out to be server-populated, add
// Computed to its schema rather than special-casing it here.
func readOptionalString(current types.String, apiValue string) types.String {
	if !current.IsNull() || apiValue != "" {
		return types.StringValue(apiValue)
	}
	return types.StringNull()
}

// readOptionalInt64 is readOptionalString for int64, with the same caveat.
func readOptionalInt64(current types.Int64, apiValue int) types.Int64 {
	if !current.IsNull() || apiValue != 0 {
		return types.Int64Value(int64(apiValue))
	}
	return types.Int64Null()
}

// requireEcho fails the apply when Tenable.io stored a value different from the
// one the configuration asked for.
//
// The provider keeps the planned value in state rather than the API's echo, so
// that an apply never trips Terraform's "Provider produced inconsistent result"
// check. On its own that would trade a crash for something arguably worse: a
// plan that proposes the same change on every run and never settles, with
// nothing on screen explaining why. Stopping here instead reports the
// divergence once, names the attribute, and shows both values.
//
// Call it after the state has been set, so a created object is still recorded
// and does not leak. State holds the planned value, which keeps it consistent
// with the plan; it is reconciled against reality by the next Read.
func requireEcho(diags *diag.Diagnostics, resourceType, attribute, planned, actual string) {
	if planned == actual {
		return
	}
	diags.AddError(
		"Tenable.io Stored a Different Value",
		fmt.Sprintf(
			"Tenable.io stored %s as %q, but the configuration for %s asked for %q.\n\n"+
				"Terraform records the configured value, so leaving this alone would make "+
				"every later plan propose the same change and never settle. The apply has "+
				"been stopped instead.\n\n"+
				"Set %s = %q to match what Tenable.io stores, or delete the object in "+
				"Tenable.io and apply again. A difference like this usually means the API "+
				"normalises the field -- case folding, trimming, or truncation.",
			attribute, actual, resourceType, planned, attribute, actual,
		),
	)
}
