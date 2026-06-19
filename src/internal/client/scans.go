package client

import (
	"context"
	"fmt"
)

type ScanSettings struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	PolicyID       int      `json:"policy_id,omitempty"`
	FolderID       int      `json:"folder_id,omitempty"`
	ScannerID      int      `json:"scanner_id,omitempty"`
	TextTargets    string   `json:"text_targets,omitempty"`
	TagTargets     []string `json:"tag_targets,omitempty"`
	FileTargets    string   `json:"file_targets,omitempty"`
	Launch         string   `json:"launch,omitempty"`
	Enabled        bool     `json:"enabled"`
	Starttime      string   `json:"starttime,omitempty"`
	RRules         string   `json:"rrules,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	Emails         string   `json:"emails,omitempty"`
	ScanTimeWindow int      `json:"scan_time_window,omitempty"`
}

type ScanCreateRequest struct {
	UUID     string       `json:"uuid"`
	Settings ScanSettings `json:"settings"`
}

type ScanUpdateRequest struct {
	UUID     string       `json:"uuid,omitempty"`
	Settings ScanSettings `json:"settings"`
}

type ScanCreateResponse struct {
	Scan ScanDetail `json:"scan"`
}

type ScanDetail struct {
	ID                 int    `json:"id"`
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	PolicyID           int    `json:"policy_id"`
	FolderID           int    `json:"folder_id"`
	ScannerID          int    `json:"scanner_id"`
	TextTargets        string `json:"text_targets"`
	Starttime          string `json:"starttime"`
	RRules             string `json:"rrules"`
	Timezone           string `json:"timezone"`
	Emails             string `json:"emails"`
	Enabled            bool   `json:"enabled"`
	Launch             string `json:"launch"`
	ScanTimeWindow     int    `json:"scan_time_window"`
	Status             string `json:"status"`
	CreationDate       int    `json:"creation_date"`
	LastModificationDate int  `json:"last_modification_date"`
	Type               string `json:"type"`
}

type ScanDetailsResponse struct {
	Info ScanInfo `json:"info"`
}

type ScanInfo struct {
	ID                   int    `json:"object_id"`
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	PolicyID             int    `json:"policy_id"`
	FolderID             int    `json:"folder_id"`
	ScannerID            int    `json:"scanner_id"`
	Targets              string `json:"targets"`
	Starttime            string `json:"starttime"`
	RRules               string `json:"rrules"`
	Timezone             string `json:"timezone"`
	Emails               string `json:"notification_email_address"`
	Enabled              bool   `json:"enabled"`
	Launch               string `json:"launch"`
	ScanTimeWindow       int    `json:"scan_time_window"`
	Status               string `json:"status"`
	CreationDate         int    `json:"creation_date"`
	LastModificationDate int    `json:"last_modification_date"`
	ScanType             string `json:"scan_type"`
	TemplateUUID         string `json:"scanner_name"`
}

type ScansListResponse struct {
	Scans []ScanListItem `json:"scans"`
}

type ScanListItem struct {
	ID                   int    `json:"id"`
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	FolderID             int    `json:"folder_id"`
	Type                 string `json:"type"`
	Status               string `json:"status"`
	Enabled              bool   `json:"enabled"`
	CreationDate         int    `json:"creation_date"`
	LastModificationDate int    `json:"last_modification_date"`
}

func (c *Client) CreateScan(ctx context.Context, req ScanCreateRequest) (*ScanCreateResponse, error) {
	var resp ScanCreateResponse
	if err := c.Post(ctx, "/scans", req, &resp); err != nil {
		return nil, fmt.Errorf("creating scan: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetScan(ctx context.Context, scanID int) (*ScanDetailsResponse, error) {
	var resp ScanDetailsResponse
	if err := c.Get(ctx, fmt.Sprintf("/scans/%d", scanID), &resp); err != nil {
		return nil, fmt.Errorf("getting scan: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdateScan(ctx context.Context, scanID int, req ScanUpdateRequest) error {
	if err := c.Put(ctx, fmt.Sprintf("/scans/%d", scanID), req, nil); err != nil {
		return fmt.Errorf("updating scan: %w", err)
	}
	return nil
}

func (c *Client) DeleteScan(ctx context.Context, scanID int) error {
	if err := c.Delete(ctx, fmt.Sprintf("/scans/%d", scanID)); err != nil {
		return fmt.Errorf("deleting scan: %w", err)
	}
	return nil
}

func (c *Client) ListScans(ctx context.Context, folderID *int) (*ScansListResponse, error) {
	path := "/scans"
	if folderID != nil {
		path = fmt.Sprintf("/scans?folder_id=%d", *folderID)
	}
	var resp ScansListResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("listing scans: %w", err)
	}
	return &resp, nil
}
