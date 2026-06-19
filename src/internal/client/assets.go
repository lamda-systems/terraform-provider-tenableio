package client

import (
	"context"
	"fmt"
	"strings"
)

type AssetListResponse struct {
	Assets []Asset `json:"assets"`
}

type Asset struct {
	ID              string   `json:"id"`
	HasAgent        bool     `json:"has_agent"`
	HasPluginResult bool     `json:"has_plugin_results"`
	FQDN            []string `json:"fqdn"`
	IPv4            []string `json:"ipv4"`
	IPv6            []string `json:"ipv6"`
	MacAddress      []string `json:"mac_address"`
	NetbiosName     []string `json:"netbios_name"`
	OperatingSystem []string `json:"operating_system"`
	AgentName       []string `json:"agent_name"`
	LastSeen        string   `json:"last_seen"`
	FirstSeen       string   `json:"first_seen"`
}

type AssetInfoResponse struct {
	Info AssetInfo `json:"info"`
}

type AssetInfo struct {
	ID                string   `json:"id"`
	HasAgent          bool     `json:"has_agent"`
	HasPluginResults  bool     `json:"has_plugin_results"`
	FQDN              []string `json:"fqdn"`
	IPv4              []string `json:"ipv4"`
	IPv6              []string `json:"ipv6"`
	MacAddress        []string `json:"mac_address"`
	NetbiosName       []string `json:"netbios_name"`
	OperatingSystem   []string `json:"operating_system"`
	AgentName         []string `json:"agent_name"`
	LastSeen          string   `json:"last_seen"`
	FirstSeen         string   `json:"first_seen"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	SystemType        string   `json:"system_type"`
	HostName          []string `json:"hostname"`
	AWSInstanceID     []string `json:"aws_ec2_instance_id"`
	AWSVPCID          []string `json:"aws_vpc_id"`
	AzureResourceID   []string `json:"azure_resource_id"`
	AzureVMID         []string `json:"azure_vm_id"`
	GCPProjectID      []string `json:"gcp_project_id"`
	GCPInstanceID     []string `json:"gcp_instance_id"`
	SeverityCounts    map[string]int `json:"counts"`
}

type AssetFilter struct {
	Filter    string `json:"filter"`
	Quality   string `json:"quality"`
	Value     string `json:"value"`
}

func (c *Client) ListAssets(ctx context.Context, dateRange int) (*AssetListResponse, error) {
	path := "/workbenches/assets"
	if dateRange > 0 {
		path = fmt.Sprintf("%s?date_range=%d", path, dateRange)
	}
	var resp AssetListResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetAsset(ctx context.Context, assetID string) (*AssetInfoResponse, error) {
	var resp AssetInfoResponse
	if err := c.Get(ctx, fmt.Sprintf("/workbenches/assets/%s/info", assetID), &resp); err != nil {
		return nil, fmt.Errorf("getting asset: %w", err)
	}
	return &resp, nil
}

func (c *Client) ListAssetsFiltered(ctx context.Context, dateRange int, filters []AssetFilter, searchType string) (*AssetListResponse, error) {
	path := "/workbenches/assets"
	params := []string{}
	if dateRange > 0 {
		params = append(params, fmt.Sprintf("date_range=%d", dateRange))
	}
	for i, f := range filters {
		params = append(params,
			fmt.Sprintf("filter.%d.filter=%s", i, f.Filter),
			fmt.Sprintf("filter.%d.quality=%s", i, f.Quality),
			fmt.Sprintf("filter.%d.value=%s", i, f.Value),
		)
	}
	if searchType != "" {
		params = append(params, fmt.Sprintf("filter.search_type=%s", searchType))
	}
	if len(params) > 0 {
		path = path + "?" + strings.Join(params, "&")
	}
	var resp AssetListResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("listing filtered assets: %w", err)
	}
	return &resp, nil
}
