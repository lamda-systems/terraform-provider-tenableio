package client

import (
	"encoding/json"
	"testing"
)

func TestTemplateUnmarshal(t *testing.T) {
	t.Parallel()

	yes := true

	tests := []struct {
		name string
		json string
		want Template
	}{
		{
			name: "documented network template reports null agent and was flags",
			json: `{"unsupported":false,"cloud_only":false,"desc":"A full system scan suitable for any host.",` +
				`"order":1,"subscription_only":false,"is_was":null,"title":"Basic Network Scan",` +
				`"is_agent":null,"uuid":"731a8e52-3ea6-a291-ec0a-d2ff0619c19d7bd788d6be818b65",` +
				`"manager_only":false,"name":"basic"}`,
			want: Template{
				UUID:        "731a8e52-3ea6-a291-ec0a-d2ff0619c19d7bd788d6be818b65",
				Name:        "basic",
				Title:       "Basic Network Scan",
				Description: "A full system scan suitable for any host.",
				Order:       new(1),
			},
		},
		{
			name: "agent template with a null order",
			json: `{"unsupported":false,"cloud_only":false,"desc":"Scan for malware on systems.",` +
				`"order":null,"subscription_only":true,"is_was":null,"title":"Malware Scan",` +
				`"is_agent":true,"uuid":"u","manager_only":false,"name":"agent_malware"}`,
			want: Template{
				UUID:             "u",
				Name:             "agent_malware",
				Title:            "Malware Scan",
				Description:      "Scan for malware on systems.",
				SubscriptionOnly: true,
				IsAgent:          &yes,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got Template
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("unmarshaling template: %s", err)
			}
			if got.UUID != tt.want.UUID || got.Name != tt.want.Name || got.Title != tt.want.Title ||
				got.Description != tt.want.Description || got.SubscriptionOnly != tt.want.SubscriptionOnly {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
			if !equalIntPtr(got.Order, tt.want.Order) {
				t.Errorf("Order = %v, want %v", deref(got.Order), deref(tt.want.Order))
			}
			if !equalBoolPtr(got.IsAgent, tt.want.IsAgent) {
				t.Errorf("IsAgent = %v, want %v", got.IsAgent, tt.want.IsAgent)
			}
			if got.IsWAS != nil {
				t.Errorf("IsWAS = %v, want nil: a null must not become false", *got.IsWAS)
			}
		})
	}
}

func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
