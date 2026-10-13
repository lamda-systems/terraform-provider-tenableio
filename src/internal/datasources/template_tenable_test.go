package datasources

import (
	"testing"

	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

func TestTemplateQueryMatch(t *testing.T) {
	t.Parallel()

	yes, no := true, false
	// Titles are not unique: Tenable.io lists a network and an agent template
	// under the same title, which is what is_agent exists to disambiguate.
	templates := []client.Template{
		{Name: "advanced", Title: "Advanced Network Scan"},
		{Name: "malware", Title: "Malware Scan"},
		{Name: "agent_malware", Title: "Malware Scan", IsAgent: &yes},
		{Name: "compliance", Title: "Policy Compliance Auditing", IsAgent: &no},
	}

	tests := []struct {
		name  string
		query templateQuery
		want  []string
	}{
		{name: "by name", query: templateQuery{name: "advanced"}, want: []string{"advanced"}},
		{name: "name is case-insensitive", query: templateQuery{name: "ADVANCED"}, want: []string{"advanced"}},
		{name: "by title", query: templateQuery{title: "advanced network scan"}, want: []string{"advanced"}},
		{name: "shared title is ambiguous", query: templateQuery{title: "Malware Scan"}, want: []string{"malware", "agent_malware"}},
		{name: "is_agent picks the agent template", query: templateQuery{title: "Malware Scan", isAgent: &yes}, want: []string{"agent_malware"}},
		{name: "is_agent false matches a null flag", query: templateQuery{title: "Malware Scan", isAgent: &no}, want: []string{"malware"}},
		{name: "is_agent false matches an explicit false", query: templateQuery{name: "compliance", isAgent: &no}, want: []string{"compliance"}},
		{name: "no partial matches", query: templateQuery{title: "Malware"}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.query.match(templates)
			if len(got) != len(tt.want) {
				t.Fatalf("match() returned %d templates, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].Name != name {
					t.Errorf("match()[%d] = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}

func TestTemplateModelNormalisesNullFlags(t *testing.T) {
	t.Parallel()

	got := templateModel(client.Template{Name: "basic"})
	if !got.Order.IsNull() {
		t.Errorf("Order = %v, want null for an unreported order", got.Order)
	}
	if got.IsAgent.IsNull() || got.IsAgent.ValueBool() {
		t.Errorf("IsAgent = %v, want false for a null flag", got.IsAgent)
	}
	if got.IsWAS.IsNull() || got.IsWAS.ValueBool() {
		t.Errorf("IsWAS = %v, want false for a null flag", got.IsWAS)
	}
}
