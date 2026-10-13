package resources

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

// This file turns the scanSettings table into everything the resource needs:
// the nested blocks of the schema, the settings of a request, the read-back of
// a details response, the echo check, and the plan-time checks. Nothing here
// names an individual setting, so the table can grow without touching it.
//
// Every setting attribute is Optional and never Computed. A template has a
// default for every setting, and adopting those into state would make every
// unconfigured attribute a diff against a null configuration. So only what the
// configuration sets is sent, tracked and verified; everything else stays at
// whatever the template says.

var settingBlockOrder = []settingBlock{blockDiscovery, blockAssessment, blockReport, blockAdvanced, blockAgent}

var settingBlockDescriptions = map[settingBlock]string{
	blockDiscovery: "Host discovery and port scanning: the Discovery section of the template editor.",
	blockAssessment: "How vulnerabilities are assessed, including Windows enumeration, web application " +
		"tests and malware scanning: the Assessment section of the template editor.",
	blockReport: "How results are reported: the Report section of the template editor.",
	blockAdvanced: "Performance, credentials handling, debugging and compliance output: the Advanced " +
		"section of the template editor.",
	blockAgent: "Info-level reporting and start staggering for agent templates.",
}

// settingsInBlock indexes scanSettings by block, in table order.
var settingsInBlock = func() map[settingBlock][]scanSetting {
	byBlock := map[settingBlock][]scanSetting{}
	for _, s := range scanSettings {
		byBlock[s.block] = append(byBlock[s.block], s)
	}
	return byBlock
}()

// attribute is the setting's Terraform attribute name: its API key with dots
// replaced, so it reads the same as the Tenable.io documentation. A block's
// preconfigured mode (discovery_mode, ...) is just "mode" inside its block.
func (s scanSetting) attribute() string {
	if s.key == string(s.block)+"_mode" {
		return "mode"
	}
	return strings.ReplaceAll(s.key, ".", "_")
}

func (s scanSetting) path() path.Path {
	return path.Root(string(s.block)).AtName(s.attribute())
}

// label names the setting in a diagnostic the way configuration spells it.
func (s scanSetting) label() string {
	return string(s.block) + "." + s.attribute()
}

func (s scanSetting) attrType() attr.Type {
	switch s.kind {
	case kindYesNo:
		return types.BoolType
	case kindNumber, kindInteger:
		return types.Int64Type
	default:
		return types.StringType
	}
}

func (s scanSetting) schemaAttribute() schema.Attribute {
	description := s.description
	switch s.family {
	case familyNetwork:
		description += " Network templates only."
	case familyAgent:
		description += " Agent templates only."
	}

	switch s.kind {
	case kindYesNo:
		return schema.BoolAttribute{Description: description, Optional: true}
	case kindChoice:
		return schema.StringAttribute{
			Description: description,
			Optional:    true,
			Validators:  []validator.String{OneOf(s.choices...)},
		}
	case kindNumber, kindInteger:
		attribute := schema.Int64Attribute{Description: description, Optional: true}
		if s.min != 0 || s.max != 0 {
			attribute.Validators = []validator.Int64{Int64Between(s.min, s.max)}
		}
		return attribute
	default:
		return schema.StringAttribute{
			Description: description,
			Optional:    true,
			Validators:  []validator.String{NoSurroundingWhitespace()},
		}
	}
}

func settingBlockAttrTypes(block settingBlock) map[string]attr.Type {
	attrTypes := map[string]attr.Type{}
	for _, s := range settingsInBlock[block] {
		attrTypes[s.attribute()] = s.attrType()
	}
	return attrTypes
}

func settingBlocksSchema() map[string]schema.Block {
	blocks := map[string]schema.Block{}
	for _, block := range settingBlockOrder {
		attributes := map[string]schema.Attribute{}
		for _, s := range settingsInBlock[block] {
			attributes[s.attribute()] = s.schemaAttribute()
		}
		blocks[string(block)] = schema.SingleNestedBlock{
			Description: settingBlockDescriptions[block],
			Attributes:  attributes,
		}
	}
	return blocks
}

// encode returns a configured value as the request carries it, and as the
// string a details response would report it as.
func (s scanSetting) encode(value attr.Value) (wire any, reported string) {
	switch s.kind {
	case kindYesNo:
		if value.(types.Bool).ValueBool() {
			return "yes", "yes"
		}
		return "no", "no"
	case kindNumber:
		text := strconv.FormatInt(value.(types.Int64).ValueInt64(), 10)
		return text, text
	case kindInteger:
		n := value.(types.Int64).ValueInt64()
		return n, strconv.FormatInt(n, 10)
	default:
		text := value.(types.String).ValueString()
		return text, text
	}
}

// decode converts a reported setting back into an attribute value. ok is false
// when the report cannot be read as this kind of setting -- "maybe" for a
// yes/no setting, say -- in which case the caller keeps what it had.
func (s scanSetting) decode(reported string) (value attr.Value, ok bool) {
	switch s.kind {
	case kindYesNo:
		switch reported {
		case "yes":
			return types.BoolValue(true), true
		case "no":
			return types.BoolValue(false), true
		}
		return nil, false
	case kindNumber, kindInteger:
		// An empty number is a cleared one, as for scan.host_timeout_minutes,
		// where empty means no timeout.
		if reported == "" {
			return types.Int64Null(), true
		}
		n, err := strconv.ParseInt(reported, 10, 64)
		if err != nil {
			return nil, false
		}
		return types.Int64Value(n), true
	default:
		return types.StringValue(reported), true
	}
}

// configuredSetting is a setting the configuration gives a known value.
type configuredSetting struct {
	setting scanSetting
	value   attr.Value
}

func blockAttribute(block types.Object, name string) attr.Value {
	if block.IsNull() || block.IsUnknown() {
		return nil
	}
	return block.Attributes()[name]
}

func isSet(value attr.Value) bool {
	return value != nil && !value.IsNull() && !value.IsUnknown()
}

// configuredSettings lists every setting with a known value, in table order.
func configuredSettings(blocks map[settingBlock]*types.Object) []configuredSetting {
	var configured []configuredSetting
	for _, s := range scanSettings {
		if value := blockAttribute(*blocks[s.block], s.attribute()); isSet(value) {
			configured = append(configured, configuredSetting{setting: s, value: value})
		}
	}
	return configured
}

// settingsRequest returns the configured settings by API key, in wire form.
func settingsRequest(configured []configuredSetting) map[string]any {
	settings := make(map[string]any, len(configured))
	for _, c := range configured {
		settings[c.setting.key], _ = c.setting.encode(c.value)
	}
	return settings
}

// readSettingBlocks refreshes the settings state already tracks from a details
// response. A setting the response does not report keeps its state value,
// because the response leaves out settings whose toggle is off; a setting state
// does not track is never adopted, because it is not configured.
func readSettingBlocks(blocks map[settingBlock]*types.Object, details *client.PolicyDetails) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, block := range settingBlockOrder {
		current := *blocks[block]
		if current.IsNull() || current.IsUnknown() {
			continue
		}
		attributes := make(map[string]attr.Value, len(current.Attributes()))
		for name, value := range current.Attributes() {
			attributes[name] = value
		}
		for _, s := range settingsInBlock[block] {
			if !isSet(attributes[s.attribute()]) {
				continue
			}
			reported := details.Setting(s.key)
			if reported == nil {
				continue
			}
			if value, ok := s.decode(*reported); ok {
				attributes[s.attribute()] = value
			}
		}
		refreshed, d := types.ObjectValue(settingBlockAttrTypes(block), attributes)
		diags.Append(d...)
		*blocks[block] = refreshed
	}
	return diags
}

// verifySettings compares every configured setting with what the details
// response reports, failing the apply on a difference. It returns the settings
// the response did not report at all, which cannot be verified.
func verifySettings(diags *diag.Diagnostics, configured []configuredSetting, details *client.PolicyDetails, source string) []string {
	var unreported []string
	for _, c := range configured {
		reported := details.Setting(c.setting.key)
		if reported == nil {
			unreported = append(unreported, c.setting.label())
			continue
		}
		_, planned := c.setting.encode(c.value)
		requireEcho(diags, scanTemplateCustomType, c.setting.label(), planned, *reported, source)
	}
	return unreported
}

// removedSettings lists the settings state tracks that the plan no longer
// sets. Tenable.io keeps a setting an update leaves out, so dropping one from
// the configuration cannot be applied in place.
func removedSettings(state, plan map[settingBlock]*types.Object) path.Paths {
	var removed path.Paths
	for _, s := range scanSettings {
		before := blockAttribute(*state[s.block], s.attribute())
		after := blockAttribute(*plan[s.block], s.attribute())
		if isSet(before) && (after == nil || after.IsNull()) {
			removed = append(removed, s.path())
		}
	}
	return removed
}

// validateModes rejects a block that pairs a preconfigured mode with
// individual settings. Tenable.io applies its own settings for any mode but
// Custom, so the individual ones would be overridden.
func validateModes(diags *diag.Diagnostics, blocks map[settingBlock]*types.Object) {
	for _, block := range settingBlockOrder {
		mode, ok := blockAttribute(*blocks[block], "mode").(types.String)
		if !ok || mode.IsNull() || mode.IsUnknown() || mode.ValueString() == "Custom" {
			continue
		}
		for _, s := range settingsInBlock[block] {
			if s.attribute() == "mode" || !isSet(blockAttribute(*blocks[block], s.attribute())) {
				continue
			}
			diags.AddAttributeError(s.path(), "Setting Overridden by Mode",
				fmt.Sprintf("%s.mode is %q, which applies Tenable.io's preconfigured %s settings and "+
					"overrides %s. Set %s.mode = \"Custom\" to configure %s settings individually.",
					block, mode.ValueString(), block, s.label(), block, block))
		}
	}
}

// validateDependencies rejects settings whose documented effect depends on
// another setting the configuration leaves off.
func validateDependencies(diags *diag.Diagnostics, blocks map[settingBlock]*types.Object) {
	assessment := *blocks[blockAssessment]
	paranoia, ok := blockAttribute(assessment, "report_paranoia").(types.String)
	if !ok || paranoia.IsNull() || paranoia.IsUnknown() || paranoia.ValueString() == "Normal" {
		return
	}
	override, ok := blockAttribute(assessment, "override_accuracy").(types.Bool)
	if ok && (override.IsUnknown() || override.ValueBool()) {
		return
	}
	diags.AddAttributeError(path.Root("assessment").AtName("report_paranoia"), "Setting Needs Its Toggle",
		fmt.Sprintf("assessment.report_paranoia is %q, but Tenable.io uses \"Normal\" unless "+
			"assessment.override_accuracy is true. Set override_accuracy = true as well.", paranoia.ValueString()))
}

// familyMismatches lists the configured settings documented only for the
// other kind of template.
func familyMismatches(configured []configuredSetting, isAgent bool) []scanSetting {
	var mismatched []scanSetting
	for _, c := range configured {
		if c.setting.family == familyNetwork && isAgent || c.setting.family == familyAgent && !isAgent {
			mismatched = append(mismatched, c.setting)
		}
	}
	return mismatched
}

func sortedList(items []string) string {
	sort.Strings(items)
	return strings.Join(items, ", ")
}
