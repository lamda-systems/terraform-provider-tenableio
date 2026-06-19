package client

import (
	"context"
	"fmt"
)

type Folder struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Custom   int    `json:"custom"`
	UnreadCount int `json:"unread_count"`
	DefaultTag  int `json:"default_tag"`
}

type FolderCreateRequest struct {
	Name string `json:"name"`
}

type FolderCreateResponse struct {
	ID int `json:"id"`
}

type FolderEditRequest struct {
	Name string `json:"name"`
}

type FoldersListResponse struct {
	Folders []Folder `json:"folders"`
}

func (c *Client) CreateFolder(ctx context.Context, name string) (*FolderCreateResponse, error) {
	var resp FolderCreateResponse
	if err := c.Post(ctx, "/folders", FolderCreateRequest{Name: name}, &resp); err != nil {
		return nil, fmt.Errorf("creating folder: %w", err)
	}
	return &resp, nil
}

func (c *Client) EditFolder(ctx context.Context, folderID int, name string) error {
	if err := c.Put(ctx, fmt.Sprintf("/folders/%d", folderID), FolderEditRequest{Name: name}, nil); err != nil {
		return fmt.Errorf("editing folder: %w", err)
	}
	return nil
}

func (c *Client) DeleteFolder(ctx context.Context, folderID int) error {
	if err := c.Delete(ctx, fmt.Sprintf("/folders/%d", folderID)); err != nil {
		return fmt.Errorf("deleting folder: %w", err)
	}
	return nil
}

func (c *Client) ListFolders(ctx context.Context) (*FoldersListResponse, error) {
	var resp FoldersListResponse
	if err := c.Get(ctx, "/folders", &resp); err != nil {
		return nil, fmt.Errorf("listing folders: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetFolder(ctx context.Context, folderID int) (*Folder, error) {
	resp, err := c.ListFolders(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range resp.Folders {
		if f.ID == folderID {
			return &f, nil
		}
	}
	return nil, &APIError{StatusCode: 404, Body: fmt.Sprintf("folder %d not found", folderID)}
}
