package datasources

import (
	"testing"

	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

func TestCustomTemplateQueryMatch(t *testing.T) {
	t.Parallel()

	// Tenable.io does not require names to be unique.
	policies := []client.PolicySummary{
		{ID: 45, Name: "Internal - Credentialed"},
		{ID: 46, Name: "Duplicate"},
		{ID: 47, Name: "Duplicate"},
	}

	tests := []struct {
		name  string
		query customTemplateQuery
		want  []int
	}{
		{name: "by name", query: customTemplateQuery{name: "Internal - Credentialed"}, want: []int{45}},
		{name: "name is case-sensitive", query: customTemplateQuery{name: "internal - credentialed"}, want: nil},
		{name: "shared name is ambiguous", query: customTemplateQuery{name: "Duplicate"}, want: []int{46, 47}},
		{name: "id disambiguates", query: customTemplateQuery{id: new(int64(47))}, want: []int{47}},
		{name: "unknown id", query: customTemplateQuery{id: new(int64(99))}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.query.match(policies)
			if len(got) != len(tt.want) {
				t.Fatalf("match() returned %d templates, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, id := range tt.want {
				if got[i].ID != id {
					t.Errorf("match()[%d].ID = %d, want %d", i, got[i].ID, id)
				}
			}
		})
	}
}

func TestScanTemplateCustomModelKeepsNullDescription(t *testing.T) {
	t.Parallel()

	got := scanTemplateCustomModel(client.PolicySummary{ID: 1, Name: "x", Shared: 1})
	if !got.Description.IsNull() {
		t.Errorf("Description = %v, want null for a null description", got.Description)
	}
	if !got.Shared.ValueBool() {
		t.Error("Shared = false, want true for shared = 1")
	}
}
