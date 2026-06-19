package client

import (
	"context"
	"fmt"
)

type ExclusionSchedule struct {
	Enabled   bool   `json:"enabled"`
	Starttime string `json:"starttime,omitempty"`
	Endtime   string `json:"endtime,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	RRules    string `json:"rrules,omitempty"`
}

type Exclusion struct {
	ID                   int                `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Members              string             `json:"members"`
	NetworkID            string             `json:"network_id"`
	Schedule             ExclusionSchedule  `json:"schedule"`
	CreationDate         int                `json:"creation_date"`
	LastModificationDate int                `json:"last_modification_date"`
}

type ExclusionCreateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Members     string             `json:"members"`
	NetworkID   string             `json:"network_id,omitempty"`
	Schedule    *ExclusionSchedule `json:"schedule,omitempty"`
}

type ExclusionUpdateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Members     string             `json:"members"`
	NetworkID   string             `json:"network_id,omitempty"`
	Schedule    *ExclusionSchedule `json:"schedule,omitempty"`
}

type ExclusionsListResponse struct {
	Exclusions []Exclusion `json:"exclusions"`
}

func (c *Client) CreateExclusion(ctx context.Context, req ExclusionCreateRequest) (*Exclusion, error) {
	var resp Exclusion
	if err := c.Post(ctx, "/exclusions", req, &resp); err != nil {
		return nil, fmt.Errorf("creating exclusion: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetExclusion(ctx context.Context, exclusionID int) (*Exclusion, error) {
	var resp Exclusion
	if err := c.Get(ctx, fmt.Sprintf("/exclusions/%d", exclusionID), &resp); err != nil {
		return nil, fmt.Errorf("getting exclusion: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdateExclusion(ctx context.Context, exclusionID int, req ExclusionUpdateRequest) (*Exclusion, error) {
	var resp Exclusion
	if err := c.Put(ctx, fmt.Sprintf("/exclusions/%d", exclusionID), req, &resp); err != nil {
		return nil, fmt.Errorf("updating exclusion: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteExclusion(ctx context.Context, exclusionID int) error {
	if err := c.Delete(ctx, fmt.Sprintf("/exclusions/%d", exclusionID)); err != nil {
		return fmt.Errorf("deleting exclusion: %w", err)
	}
	return nil
}

func (c *Client) ListExclusions(ctx context.Context) (*ExclusionsListResponse, error) {
	var resp ExclusionsListResponse
	if err := c.Get(ctx, "/exclusions", &resp); err != nil {
		return nil, fmt.Errorf("listing exclusions: %w", err)
	}
	return &resp, nil
}
