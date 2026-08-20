package client

import (
	"encoding/json"
	"strings"
	"testing"
)

// A request field backing a schema attribute that has a Default must always
// reach the wire, even when empty. Dropping it means the API never learns the
// user cleared the field, so it echoes the previous value back and the apply
// fails on a mismatch the provider created itself.
func TestRequestsSendClearedFieldsExplicitly(t *testing.T) {
	tests := []struct {
		name       string
		req        any
		mustHave   []string
		mustNotHav []string
	}{
		{
			name:     "tag category create",
			req:      TagCategoryCreateRequest{Name: "Location"},
			mustHave: []string{`"description":""`},
		},
		{
			name:     "tag category update",
			req:      TagCategoryUpdateRequest{Name: "Location"},
			mustHave: []string{`"name":"Location"`, `"description":""`},
		},
		{
			name:     "tag value create",
			req:      TagValueCreateRequest{CategoryUUID: "c", Value: "London"},
			mustHave: []string{`"description":""`},
			// filters is genuinely optional: its presence makes a tag dynamic,
			// so an absent one must stay absent.
			mustNotHav: []string{`"filters"`, `"category_name"`},
		},
		{
			name:       "tag value update",
			req:        TagValueUpdateRequest{Value: "London"},
			mustHave:   []string{`"value":"London"`, `"description":""`},
			mustNotHav: []string{`"filters"`},
		},
		{
			name:     "network create",
			req:      NetworkCreateRequest{Name: "lab"},
			mustHave: []string{`"description":""`},
		},
		{
			name:     "network update",
			req:      NetworkUpdateRequest{Name: "lab"},
			mustHave: []string{`"description":""`},
		},
		{
			name:     "exclusion create",
			req:      ExclusionCreateRequest{Name: "x", Members: "10.0.0.1"},
			mustHave: []string{`"description":""`},
			// network_id is optional with no default: absence is meaningful.
			mustNotHav: []string{`"network_id"`, `"schedule"`},
		},
		{
			name:     "policy settings carry both defaulted fields",
			req:      PolicySettings{Name: "p"},
			mustHave: []string{`"description":""`, `"visibility":""`},
		},
		{
			name:     "scan settings",
			req:      ScanSettings{Name: "s"},
			mustHave: []string{`"description":""`},
			// These have no schema default; absence is meaningful.
			mustNotHav: []string{`"policy_id"`, `"text_targets"`, `"launch"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshaling: %v", err)
			}
			got := string(encoded)
			for _, want := range tt.mustHave {
				if !strings.Contains(got, want) {
					t.Errorf("payload is missing %s\ngot: %s", want, got)
				}
			}
			for _, unwanted := range tt.mustNotHav {
				if strings.Contains(got, unwanted) {
					t.Errorf("payload should not contain %s\ngot: %s", unwanted, got)
				}
			}
		})
	}
}
