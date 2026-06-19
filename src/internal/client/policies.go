package client

import (
	"context"
	"fmt"
)

type PolicySettings struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

type PolicyCreateRequest struct {
	UUID     string         `json:"uuid"`
	Settings PolicySettings `json:"settings"`
}

type PolicyUpdateRequest struct {
	UUID     string         `json:"uuid,omitempty"`
	Settings PolicySettings `json:"settings"`
}

type PolicyCreateResponse struct {
	PolicyID   int    `json:"policy_id"`
	PolicyName string `json:"policy_name"`
}

type PolicyDetail struct {
	ID                   int    `json:"id"`
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Owner                string `json:"owner"`
	OwnerID              int    `json:"owner_id"`
	Visibility           string `json:"visibility"`
	CreationDate         int    `json:"creation_date"`
	LastModificationDate int    `json:"last_modification_date"`
	NoTarget             string `json:"no_target"`
	TemplateUUID         string `json:"template_uuid"`
}

type PolicyDetailsResponse struct {
	UUID     string         `json:"uuid"`
	Settings PolicySettings `json:"settings"`
}

type PoliciesListResponse struct {
	Policies []PolicyDetail `json:"policies"`
}

func (c *Client) CreatePolicy(ctx context.Context, req PolicyCreateRequest) (*PolicyCreateResponse, error) {
	var resp PolicyCreateResponse
	if err := c.Post(ctx, "/policies", req, &resp); err != nil {
		return nil, fmt.Errorf("creating policy: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetPolicy(ctx context.Context, policyID int) (*PolicyDetail, error) {
	var resp PolicyDetail
	if err := c.Get(ctx, fmt.Sprintf("/policies/%d", policyID), &resp); err != nil {
		return nil, fmt.Errorf("getting policy: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdatePolicy(ctx context.Context, policyID int, req PolicyUpdateRequest) error {
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
