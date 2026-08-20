package client

import (
	"context"
	"encoding/json"
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

// Fields that the resource schema gives a Default must NOT be tagged
// omitempty. The schema promises the attribute always has a value, so the plan
// always carries one; dropping it from the wire when it happens to be the zero
// value means the API never learns the user cleared it, echoes the stale value
// back, and the apply fails on a mismatch the provider itself created.
// omitempty stays only where absence is genuinely meaningful.
type TagCategoryCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TagCategoryUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TagCategoriesListResponse struct {
	Categories []TagCategory `json:"categories"`
}

// TagRule is a single asset-matching rule of a dynamic tag. The API accepts
// the matched value as either a string or an array of strings; Values always
// holds the array form and MarshalJSON collapses a single element back to a
// string, mirroring the documented request examples.
type TagRule struct {
	Property string
	Operator string
	Values   []string
}

func (r TagRule) MarshalJSON() ([]byte, error) {
	out := struct {
		Property string `json:"property"`
		Operator string `json:"operator"`
		Value    any    `json:"value"`
	}{Property: r.Property, Operator: r.Operator}
	if len(r.Values) == 1 {
		out.Value = r.Values[0]
	} else {
		out.Value = r.Values
	}
	return json.Marshal(out)
}

// UnmarshalJSON accepts both the request shape ("property", readable
// operators) and the shape the API echoes back in tag value details, where the
// attribute key is "field" and the value may be a bare string.
func (r *TagRule) UnmarshalJSON(data []byte) error {
	var raw struct {
		Property string          `json:"property"`
		Field    string          `json:"field"`
		Operator string          `json:"operator"`
		Value    json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Property = raw.Property
	if r.Property == "" {
		r.Property = raw.Field
	}
	r.Operator = raw.Operator
	r.Values = nil
	if len(raw.Value) == 0 {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw.Value, &single); err == nil {
		r.Values = []string{single}
		return nil
	}
	if err := json.Unmarshal(raw.Value, &r.Values); err != nil {
		return fmt.Errorf("tag rule value is neither a string nor a string array: %w", err)
	}
	return nil
}

// TagAssetRules groups the rules of a dynamic tag: "and" rules must all
// match, "or" rules match any.
type TagAssetRules struct {
	And []TagRule `json:"and,omitempty"`
	Or  []TagRule `json:"or,omitempty"`
}

// TagValueFilters is the request-side filters object. Its presence on create
// or update makes the tag dynamic.
type TagValueFilters struct {
	Asset TagAssetRules `json:"asset"`
}

// TagValueResponseFilters is the response-side filters object. Unlike the
// request, the API returns the asset rules as a JSON-formatted string; use
// ParseAssetRules to decode it.
type TagValueResponseFilters struct {
	Asset string `json:"asset"`
}

func (f *TagValueResponseFilters) ParseAssetRules() (*TagAssetRules, error) {
	if f == nil || f.Asset == "" {
		return nil, nil
	}
	var rules TagAssetRules
	if err := json.Unmarshal([]byte(f.Asset), &rules); err != nil {
		return nil, fmt.Errorf("parsing tag asset rules %q: %w", f.Asset, err)
	}
	return &rules, nil
}

type TagValue struct {
	UUID                string                   `json:"uuid"`
	Value               string                   `json:"value"`
	Description         string                   `json:"description"`
	CategoryUUID        string                   `json:"category_uuid"`
	CategoryName        string                   `json:"category_name"`
	CategoryDescription string                   `json:"category_description"`
	Type                string                   `json:"type"`
	Filters             *TagValueResponseFilters `json:"filters,omitempty"`
	CreatedAt           string                   `json:"created_at"`
	CreatedBy           string                   `json:"created_by"`
	UpdatedAt           string                   `json:"updated_at"`
	UpdatedBy           string                   `json:"updated_by"`
}

type TagValueCreateRequest struct {
	CategoryUUID        string `json:"category_uuid,omitempty"`
	CategoryName        string `json:"category_name,omitempty"`
	CategoryDescription string `json:"category_description,omitempty"`
	Value               string `json:"value"`
	Description         string `json:"description"`
	// filters keeps omitempty: its presence is what makes a tag dynamic.
	Filters *TagValueFilters `json:"filters,omitempty"`
}

type TagValueUpdateRequest struct {
	Value       string `json:"value"`
	Description string `json:"description"`
	// filters keeps omitempty: its presence is what makes a tag dynamic.
	Filters *TagValueFilters `json:"filters,omitempty"`
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

// AssetTagFilterListEntry is one selectable option of a dropdown control. The
// API returns entries either as bare strings or as {name, value} objects;
// bare strings are normalized so Name and Value both hold the string.
type AssetTagFilterListEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (e *AssetTagFilterListEntry) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		e.Name = s
		e.Value = s
		return nil
	}
	var obj struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("asset tag filter list entry is neither a string nor an object: %w", err)
	}
	e.Name = obj.Name
	e.Value = obj.Value
	return nil
}

// AssetTagFilterControl describes how the UI collects a value for the filter:
// free-form entry controls carry a validation regex, dropdown controls carry
// the list of options.
type AssetTagFilterControl struct {
	Type          string                    `json:"type"`
	Regex         string                    `json:"regex,omitempty"`
	ReadableRegex string                    `json:"readable_regex,omitempty"`
	List          []AssetTagFilterListEntry `json:"list,omitempty"`
}

// AssetTagFilter describes one asset attribute usable as a dynamic tag rule
// property, with the operators it supports.
type AssetTagFilter struct {
	Name         string                `json:"name"`
	ReadableName string                `json:"readable_name"`
	Operators    []string              `json:"operators"`
	Control      AssetTagFilterControl `json:"control"`
}

type AssetTagFiltersListResponse struct {
	Filters []AssetTagFilter `json:"filters"`
}

func (c *Client) ListAssetTagFilters(ctx context.Context) (*AssetTagFiltersListResponse, error) {
	var resp AssetTagFiltersListResponse
	if err := c.Get(ctx, "/tags/assets/filters", &resp); err != nil {
		return nil, fmt.Errorf("listing asset tag filters: %w", err)
	}
	return &resp, nil
}
