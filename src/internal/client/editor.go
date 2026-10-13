package client

import (
	"context"
	"fmt"
)

// TemplateType selects which of Tenable.io's lists of Tenable-provided
// templates to read.
//
// The same template -- "advanced", say -- is listed under more than one type,
// and nothing in the docs promises its uuid is the same in each. The docs do
// say which list each consumer draws from: a scan is created from a uuid in the
// scan list, a policy (a user-defined scan template) from one in the policy
// list. Keep the two apart rather than assume they coincide.
type TemplateType string

const (
	TemplateTypeScan   TemplateType = "scan"
	TemplateTypePolicy TemplateType = "policy"
)

// Template is one entry of GET /editor/{type}/templates.
type Template struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Title string `json:"title"`
	// The response calls it desc; it is the template's description.
	Description      string `json:"desc"`
	Order            *int   `json:"order"`
	Unsupported      bool   `json:"unsupported"`
	CloudOnly        bool   `json:"cloud_only"`
	SubscriptionOnly bool   `json:"subscription_only"`
	ManagerOnly      bool   `json:"manager_only"`
	// IsAgent and IsWAS are documented as "boolean or null", and the documented
	// example reports null for a template that is not an agent template. They
	// stay pointers so that null is not mistaken for an answer the API gave.
	IsAgent *bool `json:"is_agent"`
	IsWAS   *bool `json:"is_was"`
}

type TemplatesListResponse struct {
	Templates []Template `json:"templates"`
}

func (c *Client) ListTemplates(ctx context.Context, templateType TemplateType) (*TemplatesListResponse, error) {
	var resp TemplatesListResponse
	if err := c.Get(ctx, fmt.Sprintf("/editor/%s/templates", templateType), &resp); err != nil {
		return nil, fmt.Errorf("listing %s templates: %w", templateType, err)
	}
	return &resp, nil
}
