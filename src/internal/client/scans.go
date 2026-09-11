package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// Fields that the resource schema gives a Default must NOT be tagged
// omitempty. The schema promises the attribute always has a value, so the plan
// always carries one; dropping it from the wire when it happens to be the zero
// value means the API never learns the user cleared it, echoes the stale value
// back, and the apply fails on a mismatch the provider itself created.
// omitempty stays only where absence is genuinely meaningful.
type ScanSettings struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
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

// ScanUpdateResponse carries whatever PUT /scans/{id} answered with.
//
// Three shapes are in play and the provider has to survive all of them: the
// documented response is the scan object *bare* -- not wrapped in "scan" the
// way POST /scans wraps it -- while tenants have been seen to answer with no
// body at all. Scan is nil for an empty body, which means "the update was not
// echoed", not "the update echoed empty values".
type ScanUpdateResponse struct {
	Scan *ScanDetail
}

func (r *ScanUpdateResponse) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		r.Scan = nil
		return nil
	}

	var wrapped struct {
		Scan *ScanDetail `json:"scan"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Scan != nil {
		r.Scan = wrapped.Scan
		return nil
	}

	var bare ScanDetail
	if err := json.Unmarshal(data, &bare); err != nil {
		return err
	}
	r.Scan = &bare
	return nil
}

// UnmarshalJSON decodes a scan object and records which keys it carried.
func (d *ScanDetail) UnmarshalJSON(data []byte) error {
	type plain ScanDetail
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	keys, err := presentKeys(data)
	if err != nil {
		return err
	}
	*d = ScanDetail(decoded)
	d.PresentKeys = keys
	return nil
}

// UnmarshalJSON decodes an info object and records which keys it carried.
func (i *ScanInfo) UnmarshalJSON(data []byte) error {
	type plain ScanInfo
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	keys, err := presentKeys(data)
	if err != nil {
		return err
	}
	*i = ScanInfo(decoded)
	i.PresentKeys = keys
	return nil
}

// presentKeys returns the top-level keys of a JSON object, sorted.
func presentKeys(data []byte) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

type ScanDetail struct {
	// PresentKeys lists the keys the response actually carried, sorted. Key
	// names only, never values, so it is safe to quote in a diagnostic: when a
	// field comes back empty it answers "did the API omit it, or is the whole
	// object thinner than it should be?" -- which is the difference between a
	// normalising tenant and something other than Tenable.io answering.
	PresentKeys []string `json:"-"`

	ID   int    `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
	// Description is a pointer so that "the response did not carry the field"
	// stays distinguishable from "the response carried an empty string". Only
	// the second can be compared with configuration; treating the first as ""
	// is what made a tenant that omits the key look like one that wiped the
	// text. See ScanInfo.Description.
	Description          *string `json:"description"`
	PolicyID             int     `json:"policy_id"`
	FolderID             int     `json:"folder_id"`
	ScannerID            int     `json:"scanner_id"`
	TextTargets          string  `json:"text_targets"`
	Starttime            string  `json:"starttime"`
	RRules               string  `json:"rrules"`
	Timezone             string  `json:"timezone"`
	Emails               string  `json:"emails"`
	Enabled              bool    `json:"enabled"`
	Launch               string  `json:"launch"`
	ScanTimeWindow       int     `json:"scan_time_window"`
	Status               string  `json:"status"`
	CreationDate         int     `json:"creation_date"`
	LastModificationDate int     `json:"last_modification_date"`
	Type                 string  `json:"type"`
}

type ScanDetailsResponse struct {
	Info ScanInfo `json:"info"`
}

type ScanInfo struct {
	// PresentKeys is ScanDetail.PresentKeys for the details response.
	PresentKeys []string `json:"-"`

	ID   int    `json:"object_id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
	// Description is absent from the documented info schema and from live
	// responses: GET /scans/{id} reports a scan *result*, not the settings that
	// were submitted. nil therefore means "unknown", and the provider must not
	// compare it with configuration or write it over state -- doing so reported
	// every described scan as having had its description wiped.
	Description          *string `json:"description"`
	PolicyID             int     `json:"policy_id"`
	FolderID             int     `json:"folder_id"`
	ScannerID            int     `json:"scanner_id"`
	Targets              string  `json:"targets"`
	Starttime            string  `json:"starttime"`
	RRules               string  `json:"rrules"`
	Timezone             string  `json:"timezone"`
	Emails               string  `json:"notification_email_address"`
	Enabled              bool    `json:"enabled"`
	Launch               string  `json:"launch"`
	ScanTimeWindow       int     `json:"scan_time_window"`
	Status               string  `json:"status"`
	CreationDate         int     `json:"creation_date"`
	LastModificationDate int     `json:"last_modification_date"`
	ScanType             string  `json:"scan_type"`
	TemplateUUID         string  `json:"scanner_name"`
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

// UpdateScan returns the PUT echo when there is one. The response is the only
// place an update's description comes back from, since the scan details
// endpoint does not report it; resp.Scan is nil when the tenant answers with an
// empty body.
func (c *Client) UpdateScan(ctx context.Context, scanID int, req ScanUpdateRequest) (*ScanUpdateResponse, error) {
	var resp ScanUpdateResponse
	if err := c.Put(ctx, fmt.Sprintf("/scans/%d", scanID), req, &resp); err != nil {
		return nil, fmt.Errorf("updating scan: %w", err)
	}
	return &resp, nil
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
