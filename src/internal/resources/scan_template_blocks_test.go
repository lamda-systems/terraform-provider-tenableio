package resources

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

func TestScanSettingsTableIsConsistent(t *testing.T) {
	t.Parallel()

	keys := map[string]bool{}
	attributes := map[settingBlock]map[string]bool{}
	for _, s := range scanSettings {
		if keys[s.key] {
			t.Errorf("setting %q is listed twice", s.key)
		}
		keys[s.key] = true

		if attributes[s.block] == nil {
			attributes[s.block] = map[string]bool{}
		}
		if attributes[s.block][s.attribute()] {
			t.Errorf("attribute %s is produced by two settings", s.label())
		}
		attributes[s.block][s.attribute()] = true

		if s.kind == kindChoice && len(s.choices) == 0 {
			t.Errorf("choice setting %q has no choices", s.key)
		}
		if s.kind != kindChoice && len(s.choices) > 0 {
			t.Errorf("setting %q lists choices but is not a choice", s.key)
		}
		if s.description == "" {
			t.Errorf("setting %q has no description", s.key)
		}
		if s.key == "name" || s.key == "description" {
			t.Errorf("%q is a top-level attribute, not a template setting", s.key)
		}
	}
	for _, block := range settingBlockOrder {
		if len(settingsInBlock[block]) == 0 {
			t.Errorf("block %q has no settings", block)
		}
	}
}

func TestScanSettingAttributeNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setting scanSetting
		want    string
	}{
		{scanSetting{block: blockDiscovery, key: "discovery_mode"}, "mode"},
		{scanSetting{block: blockAdvanced, key: "scan.allow_multi_target"}, "scan_allow_multi_target"},
		{scanSetting{block: blockDiscovery, key: "ping_the_remote_host"}, "ping_the_remote_host"},
	}
	for _, tt := range tests {
		if got := tt.setting.attribute(); got != tt.want {
			t.Errorf("attribute(%q) = %q, want %q", tt.setting.key, got, tt.want)
		}
	}
}

func TestScanSettingEncodeDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		kind         settingKind
		value        attr.Value
		wantWire     any
		wantReported string
	}{
		{"yes", kindYesNo, types.BoolValue(true), "yes", "yes"},
		{"no", kindYesNo, types.BoolValue(false), "no", "no"},
		{"number travels as a string", kindNumber, types.Int64Value(161), "161", "161"},
		{"integer travels as an integer", kindInteger, types.Int64Value(14), int64(14), "14"},
		{"choice", kindChoice, types.StringValue("Paranoid (more false alarms)"),
			"Paranoid (more false alarms)", "Paranoid (more false alarms)"},
		{"text", kindText, types.StringValue("built-in"), "built-in", "built-in"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := scanSetting{kind: tt.kind}
			wire, reported := s.encode(tt.value)
			if wire != tt.wantWire || reported != tt.wantReported {
				t.Errorf("encode() = %#v, %q, want %#v, %q", wire, reported, tt.wantWire, tt.wantReported)
			}
			decoded, ok := s.decode(reported)
			if !ok || !decoded.Equal(tt.value) {
				t.Errorf("decode(%q) = %v, %t, want %v", reported, decoded, ok, tt.value)
			}
		})
	}
}

func TestScanSettingDecodeRejectsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	if _, ok := (scanSetting{kind: kindYesNo}).decode("maybe"); ok {
		t.Error("decode accepted \"maybe\" for a yes/no setting")
	}
	if _, ok := (scanSetting{kind: kindNumber}).decode("5m"); ok {
		t.Error("decode accepted \"5m\" for a number")
	}
	// An empty number is a cleared one, not an unreadable one.
	if v, ok := (scanSetting{kind: kindNumber}).decode(""); !ok || !v.IsNull() {
		t.Errorf("decode(\"\") = %v, %t, want null, true", v, ok)
	}
}

// blocksWith builds a set of null blocks with the given attributes set.
func blocksWith(t *testing.T, set map[settingBlock]map[string]attr.Value) map[settingBlock]*types.Object {
	t.Helper()
	blocks := map[settingBlock]*types.Object{}
	for _, block := range settingBlockOrder {
		object := types.ObjectNull(settingBlockAttrTypes(block))
		if values, ok := set[block]; ok {
			attributes := map[string]attr.Value{}
			for _, s := range settingsInBlock[block] {
				attributes[s.attribute()] = nullOf(s.attrType())
			}
			for name, value := range values {
				attributes[name] = value
			}
			var diags diag.Diagnostics
			object, diags = types.ObjectValue(settingBlockAttrTypes(block), attributes)
			if diags.HasError() {
				t.Fatalf("building %s block: %v", block, diags)
			}
		}
		blocks[block] = &object
	}
	return blocks
}

func nullOf(t attr.Type) attr.Value {
	switch t {
	case types.BoolType:
		return types.BoolNull()
	case types.Int64Type:
		return types.Int64Null()
	default:
		return types.StringNull()
	}
}

func TestReadSettingBlocks(t *testing.T) {
	t.Parallel()

	blocks := blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {
			"ping_the_remote_host": types.BoolValue(true),
			"icmp_ping_retries":    types.Int64Value(4),
		},
		blockAssessment: {
			"webcrawl_max_depth": types.Int64Value(6),
			"scan_webapps":       types.BoolValue(true),
		},
	})
	details := &client.PolicyDetails{Settings: map[string]string{
		"ping_the_remote_host": "no",    // changed outside Terraform
		"icmp_ping_retries":    "maybe", // unreadable: keep state
		"scan_webapps":         "yes",
		"udp_ping":             "yes", // reported but not configured: never adopted
		// webcrawl_max_depth not reported: keep state
	}}

	if diags := readSettingBlocks(blocks, details); diags.HasError() {
		t.Fatalf("readSettingBlocks: %v", diags)
	}

	discovery := blocks[blockDiscovery].Attributes()
	if !discovery["ping_the_remote_host"].Equal(types.BoolValue(false)) {
		t.Errorf("ping_the_remote_host = %v, want the reported false", discovery["ping_the_remote_host"])
	}
	if !discovery["icmp_ping_retries"].Equal(types.Int64Value(4)) {
		t.Errorf("icmp_ping_retries = %v, want the state value kept", discovery["icmp_ping_retries"])
	}
	if !discovery["udp_ping"].IsNull() {
		t.Errorf("udp_ping = %v, want null: it is not configured", discovery["udp_ping"])
	}
	if got := blocks[blockAssessment].Attributes()["webcrawl_max_depth"]; !got.Equal(types.Int64Value(6)) {
		t.Errorf("webcrawl_max_depth = %v, want the state value kept", got)
	}
	if !blocks[blockReport].IsNull() {
		t.Error("an unconfigured block must stay null")
	}
}

func TestVerifySettings(t *testing.T) {
	t.Parallel()

	blocks := blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery:  {"ping_the_remote_host": types.BoolValue(false)},
		blockAdvanced:   {"max_compliance_check_timeout": types.Int64Value(300)},
		blockAssessment: {"webcrawl_max_depth": types.Int64Value(6)},
	})
	configured := configuredSettings(blocks)

	var diags diag.Diagnostics
	unreported := verifySettings(&diags, configured, &client.PolicyDetails{Settings: map[string]string{
		"ping_the_remote_host":         "no",
		"max_compliance_check_timeout": "300",
	}}, "")
	if diags.HasError() {
		t.Errorf("matching settings failed verification: %v", diags)
	}
	if !reflect.DeepEqual(unreported, []string{"assessment.webcrawl_max_depth"}) {
		t.Errorf("unreported = %v, want [assessment.webcrawl_max_depth]", unreported)
	}

	diags = nil
	verifySettings(&diags, configured, &client.PolicyDetails{Settings: map[string]string{
		"ping_the_remote_host": "yes",
	}}, "")
	if !diags.HasError() {
		t.Error("a stored value different from the plan passed verification")
	}
}

func TestRemovedSettings(t *testing.T) {
	t.Parallel()

	state := blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {"ping_the_remote_host": types.BoolValue(true), "udp_ping": types.BoolValue(true)},
		blockReport:    {"reverse_lookup": types.BoolValue(true)},
	})
	plan := blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {"ping_the_remote_host": types.BoolValue(false)},
	})

	removed := removedSettings(state, plan)
	got := make([]string, len(removed))
	for i, p := range removed {
		got[i] = p.String()
	}
	want := []string{"discovery.udp_ping", "report.reverse_lookup"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("removedSettings() = %v, want %v", got, want)
	}
}

func TestValidateModes(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	validateModes(&diags, blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {"mode": types.StringValue("Custom"), "udp_ping": types.BoolValue(true)},
	}))
	if diags.HasError() {
		t.Errorf("Custom mode with individual settings was rejected: %v", diags)
	}

	diags = nil
	validateModes(&diags, blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {"mode": types.StringValue("Port scan (all ports)"), "udp_ping": types.BoolValue(true)},
	}))
	if !diags.HasError() {
		t.Error("a preconfigured mode with individual settings was accepted")
	}
}

func TestValidateDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  map[string]attr.Value
		wantErr bool
	}{
		{"paranoia without the override", map[string]attr.Value{
			"report_paranoia": types.StringValue("Paranoid (more false alarms)"),
		}, true},
		{"paranoia with the override", map[string]attr.Value{
			"report_paranoia":   types.StringValue("Paranoid (more false alarms)"),
			"override_accuracy": types.BoolValue(true),
		}, false},
		{"Normal needs no override", map[string]attr.Value{
			"report_paranoia": types.StringValue("Normal"),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var diags diag.Diagnostics
			validateDependencies(&diags, blocksWith(t, map[settingBlock]map[string]attr.Value{blockAssessment: tt.values}))
			if diags.HasError() != tt.wantErr {
				t.Errorf("error = %v, want %t", diags, tt.wantErr)
			}
		})
	}
}

func TestFamilyMismatches(t *testing.T) {
	t.Parallel()

	configured := configuredSettings(blocksWith(t, map[settingBlock]map[string]attr.Value{
		blockDiscovery: {"ping_the_remote_host": types.BoolValue(true)},             // network only
		blockAgent:     {"staggered_start_mins": types.Int64Value(30)},              // agent only
		blockReport:    {"reverse_lookup": types.BoolValue(true)},                   // both
		blockAdvanced:  {"compliance_generate_json_results": types.BoolValue(true)}, // both
	}))

	names := func(settings []scanSetting) []string {
		var out []string
		for _, s := range settings {
			out = append(out, s.key)
		}
		return out
	}
	if got := names(familyMismatches(configured, true)); !reflect.DeepEqual(got, []string{"ping_the_remote_host"}) {
		t.Errorf("on an agent template: %v, want [ping_the_remote_host]", got)
	}
	if got := names(familyMismatches(configured, false)); !reflect.DeepEqual(got, []string{"staggered_start_mins"}) {
		t.Errorf("on a network template: %v, want [staggered_start_mins]", got)
	}
}
