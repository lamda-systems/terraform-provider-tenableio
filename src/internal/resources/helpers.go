package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// readOptionalString maps an optional string from an API response into state,
// for a response that always carries the field. When the field may be absent,
// use readReportedString instead: it takes a pointer so that "not reported" and
// "reported empty" stay apart.
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

// readReportedString maps a string the API may not report at all.
//
// Two rules, and each one exists because of a bug it prevents:
//
//  1. nil means the response did not carry the field -- not that the stored
//     value is empty -- so state keeps what it already holds. Writing "" there
//     invents a change nobody made, and the resource never settles.
//  2. An empty report onto an attribute that is currently null leaves it null,
//     so an unset Optional attribute does not acquire "" and diff forever. This
//     is what makes import work.
//
// Use it wherever the response schema is narrower than the request schema, i.e.
// where the API accepts a field on write but may not return it on read.
// readOptionalString is the same thing for a response that always carries the
// field, where empty genuinely means empty.
func readReportedString(current types.String, apiValue *string) types.String {
	if apiValue == nil {
		return current
	}
	if current.IsNull() && *apiValue == "" {
		return current
	}
	return types.StringValue(*apiValue)
}

// readReportedInt64 is readReportedString for int64, with the same two rules.
func readReportedInt64(current types.Int64, apiValue *int) types.Int64 {
	if apiValue == nil {
		return current
	}
	if current.IsNull() && *apiValue == 0 {
		return current
	}
	return types.Int64Value(int64(*apiValue))
}

// readReportedBool is readReportedString for bool, minus the second rule: false
// is a meaningful value rather than an absent one, so a reported false is
// adopted even against a null attribute.
func readReportedBool(current types.Bool, apiValue *bool) types.Bool {
	if apiValue == nil {
		return current
	}
	return types.BoolValue(*apiValue)
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
func requireEcho(diags *diag.Diagnostics, resourceType, attribute, planned, actual string, notes ...string) {
	if planned == actual {
		return
	}
	detail := fmt.Sprintf(
		"Tenable.io stored %s as %q, but the configuration for %s asked for %q.\n\n"+
			"Terraform records the configured value, so leaving this alone would make "+
			"every later plan propose the same change and never settle. The apply has "+
			"been stopped instead.\n\n"+
			"Set %s = %q to match what Tenable.io stores, or delete the object in "+
			"Tenable.io and apply again. A difference like this usually means the API "+
			"normalises the field -- case folding, trimming, or truncation.",
		attribute, actual, resourceType, planned, attribute, actual,
	)
	// Notes describe the response the value was read from. They are the only
	// diagnostic channel that survives a production run where logs cannot be
	// collected, so put what identifies the responder in here: the URL called
	// and the keys that came back.
	for _, note := range notes {
		if note != "" {
			detail += "\n\n" + note
		}
	}
	diags.AddError("Tenable.io Stored a Different Value", detail)
}

// echoSource describes the response an echoed value was read from, for a
// requireEcho note. It names the exact URL, so a base_url pointing somewhere
// other than Tenable.io is visible in the error itself, and lists the keys the
// object carried, which separates "the API omitted this field" from "the object
// that came back is not the one this endpoint documents".
func echoSource(method, url string, presentKeys []string, proxy string) string {
	note := fmt.Sprintf(
		"Read from the response to %s %s, whose object carried these keys: %s.\n"+
			"If that URL is not the Tenable.io tenant you expect, or those keys are "+
			"thinner than the endpoint documents, then what answered the provider is "+
			"not what answered your own API client.",
		method, url, keyList(presentKeys),
	)
	// Whether a proxy sits in the path is the other half of that question, and
	// it is invisible from the URL alone: Terraform's requests honour
	// HTTP(S)_PROXY, while a hand-made request from an API client usually does
	// not. Report both outcomes -- "no proxy" rules the explanation out, which
	// is worth as much as naming one.
	if proxy != "" {
		return note + fmt.Sprintf("\nThe environment routes that URL through the proxy %s, "+
			"so the provider and a direct API client are not talking to the same thing.", proxy)
	}
	return note + "\nNo proxy from the environment applies to that URL."
}
