package client

import (
	"context"
	"fmt"
)

type Scanner struct {
	ID               int    `json:"id"`
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	ScanCount        int    `json:"scan_count"`
	EngineVersion    string `json:"engine_version"`
	Platform         string `json:"platform"`
	LoadedPluginSet  string `json:"loaded_plugin_set"`
	Owner            string `json:"owner"`
	OwnerID          int    `json:"owner_id"`
	Pool             bool   `json:"pool"`
	Shared           int    `json:"shared"`
	UserPermissions  int    `json:"user_permissions"`
	CreationDate     int    `json:"creation_date"`
	LastModifiedDate int    `json:"last_modification_date"`
	NetworkName      string `json:"network_name"`
}

type ScannersListResponse struct {
	Scanners []Scanner `json:"scanners"`
}

func (c *Client) GetScanner(ctx context.Context, scannerID int) (*Scanner, error) {
	var resp Scanner
	if err := c.Get(ctx, fmt.Sprintf("/scanners/%d", scannerID), &resp); err != nil {
		return nil, fmt.Errorf("getting scanner: %w", err)
	}
	return &resp, nil
}

func (c *Client) ListScanners(ctx context.Context) (*ScannersListResponse, error) {
	var resp ScannersListResponse
	if err := c.Get(ctx, "/scanners", &resp); err != nil {
		return nil, fmt.Errorf("listing scanners: %w", err)
	}
	return &resp, nil
}
