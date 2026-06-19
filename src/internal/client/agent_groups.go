package client

import (
	"context"
	"fmt"
)

type AgentGroup struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	OwnerID      int    `json:"owner_id"`
	Owner        string `json:"owner"`
	Shared       int    `json:"shared"`
	AgentsCount  int    `json:"agents_count"`
	CreationDate int    `json:"creation_date"`
	Timestamp    int    `json:"timestamp"`
}

type AgentGroupCreateRequest struct {
	Name string `json:"name"`
}

type AgentGroupsListResponse struct {
	Groups []AgentGroup `json:"groups"`
}

func (c *Client) CreateAgentGroup(ctx context.Context, name string) (*AgentGroup, error) {
	var resp AgentGroup
	if err := c.Post(ctx, "/scanners/null/agent-groups", AgentGroupCreateRequest{Name: name}, &resp); err != nil {
		return nil, fmt.Errorf("creating agent group: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetAgentGroup(ctx context.Context, groupID int) (*AgentGroup, error) {
	var resp AgentGroup
	if err := c.Get(ctx, fmt.Sprintf("/scanners/null/agent-groups/%d", groupID), &resp); err != nil {
		return nil, fmt.Errorf("getting agent group: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteAgentGroup(ctx context.Context, groupID int) error {
	if err := c.Delete(ctx, fmt.Sprintf("/scanners/null/agent-groups/%d", groupID)); err != nil {
		return fmt.Errorf("deleting agent group: %w", err)
	}
	return nil
}

func (c *Client) ListAgentGroups(ctx context.Context) (*AgentGroupsListResponse, error) {
	var resp AgentGroupsListResponse
	if err := c.Get(ctx, "/scanners/null/agent-groups", &resp); err != nil {
		return nil, fmt.Errorf("listing agent groups: %w", err)
	}
	return &resp, nil
}
