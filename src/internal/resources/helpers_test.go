package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The readReported* family exists because Tenable.io's read endpoints are
// narrower than its write endpoints. Two rules carry the whole contract, and
// each one has a production failure behind it.
func TestReadReportedString(t *testing.T) {
	tests := []struct {
		name    string
		current types.String
		api     *string
		want    types.String
	}{
		{
			// The drift loop: an absent field read as "" wipes state, the next
			// plan proposes restoring it, and nothing ever settles.
			name:    "absent keeps the configured value",
			current: types.StringValue("daily scan"),
			api:     nil,
			want:    types.StringValue("daily scan"),
		},
		{
			name:    "absent keeps null",
			current: types.StringNull(),
			api:     nil,
			want:    types.StringNull(),
		},
		{
			name:    "reported value is adopted",
			current: types.StringValue("old"),
			api:     strptr("new"),
			want:    types.StringValue("new"),
		},
		{
			// A reported empty against a set attribute is a real clearing, and
			// requireEcho is what decides whether to accept it.
			name:    "reported empty clears a set value",
			current: types.StringValue("old"),
			api:     strptr(""),
			want:    types.StringValue(""),
		},
		{
			// An unset Optional attribute must not acquire "", or it diffs
			// against a null configuration forever. This is what makes import
			// work.
			name:    "reported empty leaves an unset attribute unset",
			current: types.StringNull(),
			api:     strptr(""),
			want:    types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readReportedString(tt.current, tt.api); !got.Equal(tt.want) {
				t.Errorf("readReportedString() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestReadReportedInt64(t *testing.T) {
	tests := []struct {
		name    string
		current types.Int64
		api     *int
		want    types.Int64
	}{
		{
			// The reported case: scan_time_window 180 in configuration, absent
			// from the details response, read back as 0.
			name:    "absent keeps the configured value",
			current: types.Int64Value(180),
			api:     nil,
			want:    types.Int64Value(180),
		},
		{name: "absent keeps null", current: types.Int64Null(), api: nil, want: types.Int64Null()},
		{name: "reported value is adopted", current: types.Int64Value(1), api: intptr(180), want: types.Int64Value(180)},
		{name: "reported zero clears a set value", current: types.Int64Value(180), api: intptr(0), want: types.Int64Value(0)},
		{name: "reported zero leaves an unset attribute unset", current: types.Int64Null(), api: intptr(0), want: types.Int64Null()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readReportedInt64(tt.current, tt.api); !got.Equal(tt.want) {
				t.Errorf("readReportedInt64() = %s, want %s", got, tt.want)
			}
		})
	}
}

// Bool has no "empty means unset" case: a reported false is a real false.
func TestReadReportedBool(t *testing.T) {
	if got := readReportedBool(types.BoolValue(true), nil); !got.Equal(types.BoolValue(true)) {
		t.Errorf("absent = %s, want true preserved", got)
	}
	if got := readReportedBool(types.BoolValue(true), boolptr(false)); !got.Equal(types.BoolValue(false)) {
		t.Errorf("reported false = %s, want false adopted", got)
	}
	if got := readReportedBool(types.BoolNull(), boolptr(false)); !got.Equal(types.BoolValue(false)) {
		t.Errorf("reported false against null = %s, want false adopted", got)
	}
}

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }
func boolptr(b bool) *bool    { return &b }
