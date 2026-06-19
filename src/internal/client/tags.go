package client

import (
	"context"
	"fmt"
)

type TagCategory struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by"`
	UpdatedAt   string `json:"updated_at"`
	UpdatedBy   string `json:"updated_by"`
}

type TagCategoryCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type TagCategoryUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type TagCategoriesListResponse struct {
	Categories []TagCategory `json:"categories"`
}

type TagValue struct {
	UUID                string `json:"uuid"`
	Value               string `json:"value"`
	Description         string `json:"description"`
	CategoryUUID        string `json:"category_uuid"`
	CategoryName        string `json:"category_name"`
	CategoryDescription string `json:"category_description"`
	Type                string `json:"type"`
	CreatedAt           string `json:"created_at"`
	CreatedBy           string `json:"created_by"`
	UpdatedAt           string `json:"updated_at"`
	UpdatedBy           string `json:"updated_by"`
}

type TagValueCreateRequest struct {
	CategoryUUID        string `json:"category_uuid,omitempty"`
	CategoryName        string `json:"category_name,omitempty"`
	CategoryDescription string `json:"category_description,omitempty"`
	Value               string `json:"value"`
	Description         string `json:"description,omitempty"`
}

type TagValueUpdateRequest struct {
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
}

type TagValuesListResponse struct {
	Values []TagValue `json:"values"`
}

func (c *Client) CreateTagCategory(ctx context.Context, req TagCategoryCreateRequest) (*TagCategory, error) {
	var resp TagCategory
	if err := c.Post(ctx, "/tags/categories", req, &resp); err != nil {
		return nil, fmt.Errorf("creating tag category: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetTagCategory(ctx context.Context, categoryUUID string) (*TagCategory, error) {
	var resp TagCategory
	if err := c.Get(ctx, fmt.Sprintf("/tags/categories/%s", categoryUUID), &resp); err != nil {
		return nil, fmt.Errorf("getting tag category: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdateTagCategory(ctx context.Context, categoryUUID string, req TagCategoryUpdateRequest) (*TagCategory, error) {
	var resp TagCategory
	if err := c.Put(ctx, fmt.Sprintf("/tags/categories/%s", categoryUUID), req, &resp); err != nil {
		return nil, fmt.Errorf("updating tag category: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteTagCategory(ctx context.Context, categoryUUID string) error {
	if err := c.Delete(ctx, fmt.Sprintf("/tags/categories/%s", categoryUUID)); err != nil {
		return fmt.Errorf("deleting tag category: %w", err)
	}
	return nil
}

func (c *Client) ListTagCategories(ctx context.Context) (*TagCategoriesListResponse, error) {
	var resp TagCategoriesListResponse
	if err := c.Get(ctx, "/tags/categories", &resp); err != nil {
		return nil, fmt.Errorf("listing tag categories: %w", err)
	}
	return &resp, nil
}

func (c *Client) CreateTagValue(ctx context.Context, req TagValueCreateRequest) (*TagValue, error) {
	var resp TagValue
	if err := c.Post(ctx, "/tags/values", req, &resp); err != nil {
		return nil, fmt.Errorf("creating tag value: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetTagValue(ctx context.Context, valueUUID string) (*TagValue, error) {
	var resp TagValue
	if err := c.Get(ctx, fmt.Sprintf("/tags/values/%s", valueUUID), &resp); err != nil {
		return nil, fmt.Errorf("getting tag value: %w", err)
	}
	return &resp, nil
}

func (c *Client) UpdateTagValue(ctx context.Context, valueUUID string, req TagValueUpdateRequest) (*TagValue, error) {
	var resp TagValue
	if err := c.Put(ctx, fmt.Sprintf("/tags/values/%s", valueUUID), req, &resp); err != nil {
		return nil, fmt.Errorf("updating tag value: %w", err)
	}
	return &resp, nil
}

func (c *Client) DeleteTagValue(ctx context.Context, valueUUID string) error {
	if err := c.Delete(ctx, fmt.Sprintf("/tags/values/%s", valueUUID)); err != nil {
		return fmt.Errorf("deleting tag value: %w", err)
	}
	return nil
}

func (c *Client) ListTagValues(ctx context.Context) (*TagValuesListResponse, error) {
	var resp TagValuesListResponse
	if err := c.Get(ctx, "/tags/values", &resp); err != nil {
		return nil, fmt.Errorf("listing tag values: %w", err)
	}
	return &resp, nil
}
