package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPolicySummaryUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		json        string
		wantDesc    *string
		wantShared  int
		wantOwnerID int
	}{
		{
			name: "documented list entry",
			json: `{"no_target":"false","template_uuid":"ab4bacd2-05f6-425c-9d79-3ba7941a78d5d24bd260ef5f9e66",` +
				`"description":"An example policy.","name":"Example Network Scan Policy",` +
				`"owner":"example.user@example.com","visibility":"private","shared":0,` +
				`"user_permissions":128,"last_modification_date":1759190400,` +
				`"creation_date":1759104000,"owner_id":1,"id":45}`,
			wantDesc:    new("An example policy."),
			wantOwnerID: 1,
		},
		{
			name:       "null description stays nil",
			json:       `{"id":46,"name":"Shared","description":null,"shared":1}`,
			wantShared: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got PolicySummary
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("unmarshaling policy summary: %s", err)
			}
			if (got.Description == nil) != (tt.wantDesc == nil) ||
				(got.Description != nil && *got.Description != *tt.wantDesc) {
				t.Errorf("Description = %v, want %v", got.Description, tt.wantDesc)
			}
			if got.Shared != tt.wantShared || got.OwnerID != tt.wantOwnerID {
				t.Errorf("Shared, OwnerID = %d, %d, want %d, %d", got.Shared, got.OwnerID, tt.wantShared, tt.wantOwnerID)
			}
		})
	}
}

func TestPolicyRequestMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  PolicyRequest
		want string
	}{
		{
			name: "create flattens template settings beside name and description",
			req: PolicyRequest{
				TemplateUUID: "t", Name: "p", Description: "d",
				Settings: map[string]any{"ping_the_remote_host": "no", "refresh_reporting_frequency_days": int64(14)},
			},
			want: `{"settings":{"description":"d","name":"p","ping_the_remote_host":"no",` +
				`"refresh_reporting_frequency_days":14},"uuid":"t"}`,
		},
		{
			name: "a template setting cannot shadow the name",
			req:  PolicyRequest{Name: "p", Settings: map[string]any{"name": "other"}},
			want: `{"settings":{"description":"","name":"p"}}`,
		},
		{
			name: "plugins only when there are some",
			req:  PolicyRequest{Name: "p", Plugins: map[string]PluginFamily{"Backdoors": {Status: "disabled"}}},
			want: `{"plugins":{"Backdoors":{"status":"disabled"}},"settings":{"description":"","name":"p"}}`,
		},
		{
			name: "legacy visibility is sent when set",
			req:  PolicyRequest{Name: "p", Visibility: "shared"},
			want: `{"settings":{"description":"","name":"p","visibility":"shared"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshaling request: %s", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s\nwant          %s", got, tt.want)
			}
		})
	}
}

func TestPolicyDetailsUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		json         string
		wantTemplate *string
		wantSettings map[string]string
		wantPlugins  map[string]PluginFamily
		wantKeys     []string
	}{
		{
			name: "documented nested shape with integer settings",
			json: `{"uuid":"ab4bacd2","settings":{"name":"Example","description":"An example policy.",` +
				`"ping_the_remote_host":"yes","max_compliance_check_timeout":300,"http_reauth_delay":0},` +
				`"plugins":{"Backdoors":{"status":"enabled"}}}`,
			wantTemplate: new("ab4bacd2"),
			wantSettings: map[string]string{
				"name": "Example", "description": "An example policy.", "ping_the_remote_host": "yes",
				"max_compliance_check_timeout": "300", "http_reauth_delay": "0",
			},
			wantPlugins: map[string]PluginFamily{"Backdoors": {Status: "enabled"}},
			wantKeys:    []string{"plugins", "settings", "uuid"},
		},
		{
			name:         "non-scalar and null settings are not reported",
			json:         `{"uuid":"u","settings":{"name":"p","acls":[{"type":"default"}],"smtp_to":null}}`,
			wantTemplate: new("u"),
			wantSettings: map[string]string{"name": "p"},
			wantKeys:     []string{"settings", "uuid"},
		},
		{
			name:         "flat shape is read as settings, and its uuid is not taken for the template",
			json:         `{"uuid":"policy-own-uuid","name":"p","description":"","id":45}`,
			wantSettings: map[string]string{"uuid": "policy-own-uuid", "name": "p", "description": "", "id": "45"},
			wantKeys:     []string{"description", "id", "name", "uuid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got PolicyDetails
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("unmarshaling details: %s", err)
			}
			if (got.TemplateUUID == nil) != (tt.wantTemplate == nil) ||
				(got.TemplateUUID != nil && *got.TemplateUUID != *tt.wantTemplate) {
				t.Errorf("TemplateUUID = %v, want %v", got.TemplateUUID, tt.wantTemplate)
			}
			if !reflect.DeepEqual(got.Settings, tt.wantSettings) {
				t.Errorf("Settings = %v, want %v", got.Settings, tt.wantSettings)
			}
			if !reflect.DeepEqual(got.Plugins, tt.wantPlugins) {
				t.Errorf("Plugins = %v, want %v", got.Plugins, tt.wantPlugins)
			}
			if !reflect.DeepEqual(got.PresentKeys, tt.wantKeys) {
				t.Errorf("PresentKeys = %v, want %v", got.PresentKeys, tt.wantKeys)
			}
		})
	}
}

func TestPolicyDetailsSettingDistinguishesAbsentFromEmpty(t *testing.T) {
	t.Parallel()

	d := PolicyDetails{Settings: map[string]string{"description": ""}}
	if got := d.Setting("description"); got == nil || *got != "" {
		t.Errorf("Setting(description) = %v, want a pointer to \"\"", got)
	}
	if got := d.Setting("webcrawl_max_depth"); got != nil {
		t.Errorf("Setting(webcrawl_max_depth) = %q, want nil for an unreported setting", *got)
	}
}
