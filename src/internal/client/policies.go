package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Policies are what the Tenable.io UI calls user-defined scan templates. Three
// objects are in play and they do not match each other:
//
//   - POST /policies and PUT /policies/{id} take {uuid, settings, plugins},
//     where settings is one flat object holding name, description and every
//     template setting side by side. POST answers {policy_id, policy_name};
//     PUT answers an empty object.
//   - GET /policies/{id} returns the same nesting back -- {uuid, settings,
//     plugins, ...} -- with every setting value a string, except two that may
//     arrive as integers. It carries no metadata: no owner, no timestamps.
//   - GET /policies lists flat summaries carrying the metadata and none of the
//     settings.
//
// So a full read is two calls: GetPolicyDetails for what the policy does and
// FindPolicy for who owns it and when it changed.

// PluginFamily is the status of one plugin family in a policy, keyed by family
// name wherever it appears.
type PluginFamily struct {
	Status string `json:"status"`
}

// PolicyRequest is the body of POST /policies and PUT /policies/{id}.
//
// Name and Description are always sent, even when empty: the schema gives
// description a default, and an elided empty value would leave a stale one in
// place, because PUT preserves every setting it is not sent.
type PolicyRequest struct {
	// TemplateUUID is the Tenable-provided template the policy is based on.
	// Sent on create only: the API ignores it on update, because a policy
	// cannot move to another template.
	TemplateUUID string
	Name         string
	Description  string
	// Visibility is not a documented policy setting. It is sent only for the
	// deprecated tenableio_policy, which has always sent it.
	Visibility string
	// Settings are template settings by API key, already in wire form.
	Settings map[string]any
	// Plugins is omitted when empty: only templates that expose plugin
	// selection accept it at all.
	Plugins map[string]PluginFamily
}

func (r PolicyRequest) MarshalJSON() ([]byte, error) {
	settings := make(map[string]any, len(r.Settings)+3)
	for key, value := range r.Settings {
		settings[key] = value
	}
	// Written last so a template setting can never shadow them.
	settings["name"] = r.Name
	settings["description"] = r.Description
	if r.Visibility != "" {
		settings["visibility"] = r.Visibility
	}

	body := map[string]any{"settings": settings}
	if r.TemplateUUID != "" {
		body["uuid"] = r.TemplateUUID
	}
	if len(r.Plugins) > 0 {
		body["plugins"] = r.Plugins
	}
	return json.Marshal(body)
}

type PolicyCreateResponse struct {
	PolicyID   int    `json:"policy_id"`
	PolicyName string `json:"policy_name"`
}

// PolicyDetails is GET /policies/{id}.
type PolicyDetails struct {
	// PresentKeys lists the top-level keys the response carried, sorted. Key
	// names only, so it is safe to quote in a diagnostic.
	PresentKeys []string
	// TemplateUUID is the response's uuid: the Tenable-provided template the
	// policy is based on, not an identifier of the policy itself.
	TemplateUUID *string
	// Settings holds every setting the response reported, name and description
	// included, each in its string form. A key that is absent was not
	// reported, which is not the same as reported empty: the response leaves
	// out settings that depend on a toggle that is off.
	Settings map[string]string
	// Plugins is nil when the response carried no plugins object, which it
	// omits for templates without plugin selection.
	Plugins map[string]PluginFamily
}

// Setting returns a reported setting, or nil when the response did not carry it.
func (d *PolicyDetails) Setting(key string) *string {
	value, ok := d.Settings[key]
	if !ok {
		return nil
	}
	return &value
}

func (d *PolicyDetails) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// The documented response nests the settings. A flat object -- settings at
	// the top level -- is accepted too, so that a responder answering in that
	// shape is read rather than mistaken for a policy with no settings at all.
	// PresentKeys still records which shape it was.
	settingsJSON, nested := raw["settings"]
	if !nested {
		settingsJSON = data
	}
	settings, err := decodeSettings(settingsJSON)
	if err != nil {
		return fmt.Errorf("decoding policy settings: %w", err)
	}

	details := PolicyDetails{PresentKeys: keys, Settings: settings}
	if nested {
		if uuid, ok := raw["uuid"]; ok {
			var s string
			if err := json.Unmarshal(uuid, &s); err == nil {
				details.TemplateUUID = &s
			}
		}
	}
	if plugins, ok := raw["plugins"]; ok && string(plugins) != "null" {
		if err := json.Unmarshal(plugins, &details.Plugins); err != nil {
			return fmt.Errorf("decoding policy plugins: %w", err)
		}
	}
	*d = details
	return nil
}

// decodeSettings reads a settings object into string form. Strings are kept as
// they are and numbers are written out in decimal, because the documented
// response mixes the two: max_compliance_check_timeout and http_reauth_delay
// may arrive as integers. Nulls, objects and arrays are not settings a scalar
// attribute can be compared with, so they are left out, as if not reported.
func decodeSettings(data []byte) (map[string]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	settings := make(map[string]string, len(raw))
	for key, value := range raw {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(value))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			return nil, fmt.Errorf("setting %q: %w", key, err)
		}
		switch v := decoded.(type) {
		case string:
			settings[key] = v
		case json.Number:
			settings[key] = v.String()
		case bool:
			settings[key] = strconv.FormatBool(v)
		}
	}
	return settings, nil
}

// PolicySummary is one entry of GET /policies, the list of user-defined scan
// templates. It is a different object from what GET /policies/{id} returns:
// the list carries the metadata (owner, sharing, timestamps) and the details
// carry the settings.
type PolicySummary struct {
	ID           int    `json:"id"`
	TemplateUUID string `json:"template_uuid"`
	Name         string `json:"name"`
	// Description is documented as "string or null"; nil keeps the two apart.
	Description          *string `json:"description"`
	OwnerID              int     `json:"owner_id"`
	Owner                string  `json:"owner"`
	Shared               int     `json:"shared"`
	UserPermissions      int     `json:"user_permissions"`
	CreationDate         int     `json:"creation_date"`
	LastModificationDate int     `json:"last_modification_date"`
	Visibility           string  `json:"visibility"`
	NoTarget             string  `json:"no_target"`
	// UUID is not documented for list entries. It is kept only because
	// tenableio_policies has always exposed it.
	UUID string `json:"uuid"`
}

type PoliciesListResponse struct {
	Policies []PolicySummary `json:"policies"`
}

func (c *Client) CreatePolicy(ctx context.Context, req PolicyRequest) (*PolicyCreateResponse, error) {
	var resp PolicyCreateResponse
	if err := c.Post(ctx, "/policies", req, &resp); err != nil {
		return nil, fmt.Errorf("creating policy: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetPolicyDetails(ctx context.Context, policyID int) (*PolicyDetails, error) {
	var resp PolicyDetails
	if err := c.Get(ctx, fmt.Sprintf("/policies/%d", policyID), &resp); err != nil {
		return nil, fmt.Errorf("getting policy: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdatePolicy(ctx context.Context, policyID int, req PolicyRequest) error {
	req.TemplateUUID = ""
	if err := c.Put(ctx, fmt.Sprintf("/policies/%d", policyID), req, nil); err != nil {
		return fmt.Errorf("updating policy: %w", err)
	}
	return nil
}

func (c *Client) DeletePolicy(ctx context.Context, policyID int) error {
	if err := c.Delete(ctx, fmt.Sprintf("/policies/%d", policyID)); err != nil {
		return fmt.Errorf("deleting policy: %w", err)
	}
	return nil
}

func (c *Client) ListPolicies(ctx context.Context) (*PoliciesListResponse, error) {
	var resp PoliciesListResponse
	if err := c.Get(ctx, "/policies", &resp); err != nil {
		return nil, fmt.Errorf("listing policies: %w", err)
	}
	return &resp, nil
}

// FindPolicy returns the list entry for one policy, or nil when the list does
// not carry it. The list has no filter, so this reads all of it.
func (c *Client) FindPolicy(ctx context.Context, policyID int) (*PolicySummary, error) {
	list, err := c.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list.Policies {
		if list.Policies[i].ID == policyID {
			return &list.Policies[i], nil
		}
	}
	return nil, nil
}
