package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTagRuleMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rule TagRule
		want string
	}{
		{
			name: "single value collapses to a string",
			rule: TagRule{Property: "asset_class", Operator: "equals", Values: []string{"server"}},
			want: `{"property":"asset_class","operator":"equals","value":"server"}`,
		},
		{
			name: "multiple values stay an array",
			rule: TagRule{Property: "ipv4", Operator: "equals", Values: []string{"192.0.2.57", "192.0.2.58"}},
			want: `{"property":"ipv4","operator":"equals","value":["192.0.2.57","192.0.2.58"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.rule)
			if err != nil {
				t.Fatalf("marshaling rule: %s", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTagRuleUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
		want TagRule
	}{
		{
			name: "request shape with property and string value",
			json: `{"property":"asset_class","operator":"equals","value":"server"}`,
			want: TagRule{Property: "asset_class", Operator: "equals", Values: []string{"server"}},
		},
		{
			name: "response shape with field and array value",
			json: `{"field":"ipv4","operator":"eq","value":["192.0.2.57","192.0.2.58"]}`,
			want: TagRule{Property: "ipv4", Operator: "eq", Values: []string{"192.0.2.57", "192.0.2.58"}},
		},
		{
			name: "property wins when both keys are present",
			json: `{"property":"fqdn","field":"ignored","operator":"match","value":"corp.example.com"}`,
			want: TagRule{Property: "fqdn", Operator: "match", Values: []string{"corp.example.com"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got TagRule
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("unmarshaling rule: %s", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// The API returns filters.asset in tag value details as a JSON-formatted
// string, not an object.
func TestTagValueResponseFiltersParseAssetRules(t *testing.T) {
	t.Parallel()

	f := &TagValueResponseFilters{
		Asset: `{"and":[{"field":"operating_system","operator":"match","value":"FreeBSD"}]}`,
	}
	rules, err := f.ParseAssetRules()
	if err != nil {
		t.Fatalf("parsing asset rules: %s", err)
	}
	want := &TagAssetRules{
		And: []TagRule{{Property: "operating_system", Operator: "match", Values: []string{"FreeBSD"}}},
	}
	if !reflect.DeepEqual(rules, want) {
		t.Errorf("ParseAssetRules() = %+v, want %+v", rules, want)
	}

	if rules, err := (*TagValueResponseFilters)(nil).ParseAssetRules(); err != nil || rules != nil {
		t.Errorf("nil receiver: got %+v, %v; want nil, nil", rules, err)
	}
}

func TestAssetTagFilterListEntryUnmarshal(t *testing.T) {
	t.Parallel()

	var control AssetTagFilterControl
	payload := `{
		"type": "dropdown",
		"list": ["running", {"name": "US East (N. Virginia)", "value": "us-east-1"}]
	}`
	if err := json.Unmarshal([]byte(payload), &control); err != nil {
		t.Fatalf("unmarshaling control: %s", err)
	}

	want := []AssetTagFilterListEntry{
		{Name: "running", Value: "running"},
		{Name: "US East (N. Virginia)", Value: "us-east-1"},
	}
	if !reflect.DeepEqual(control.List, want) {
		t.Errorf("control.List = %+v, want %+v", control.List, want)
	}
}
