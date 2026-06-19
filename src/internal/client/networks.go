package client

import (
	"context"
	"fmt"
)

type Network struct {
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	IsDefault          bool   `json:"is_default"`
	CreatedBy          string `json:"created_by"`
	CreatedInSeconds   int    `json:"created_in_seconds"`
	ModifiedInSeconds  int    `json:"modified_in_seconds"`
	ScannerCount       int    `json:"scanner_count"`
	AssetsTTLDays      int    `json:"assets_ttl_days"`
}

type NetworkCreateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	AssetsTTLDays int    `json:"assets_ttl_days,omitempty"`
}

type NetworkUpdateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	AssetsTTLDays int    `json:"assets_ttl_days,omitempty"`
}

type NetworksListResponse struct {
	Networks []Network `json:"networks"`
}

func (c *Client) CreateNetwork(ctx context.Context, req NetworkCreateRequest) (*Network, error) {
	var resp Network
	if err := c.Post(ctx, "/networks", req, &resp); err != nil {
		return nil, fmt.Errorf("creating network: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetNetwork(ctx context.Context, networkID string) (*Network, error) {
	var resp Network
	if err := c.Get(ctx, fmt.Sprintf("/networks/%s", networkID), &resp); err != nil {
		return nil, fmt.Errorf("getting network: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdateNetwork(ctx context.Context, networkID string, req NetworkUpdateRequest) (*Network, error) {
	var resp Network
	if err := c.Put(ctx, fmt.Sprintf("/networks/%s", networkID), req, &resp); err != nil {
		return nil, fmt.Errorf("updating network: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteNetwork(ctx context.Context, networkID string) error {
	if err := c.Delete(ctx, fmt.Sprintf("/networks/%s", networkID)); err != nil {
		return fmt.Errorf("deleting network: %w", err)
	}
	return nil
}

func (c *Client) ListNetworks(ctx context.Context) (*NetworksListResponse, error) {
	var resp NetworksListResponse
	if err := c.Get(ctx, "/networks", &resp); err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	return &resp, nil
}
