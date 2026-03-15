package models

import (
	"time"
)

// NodeType represents the type of node
type NodeType string

const (
	NodeTypeFolder NodeType = "folder"
	NodeTypeFile   NodeType = "file"
	NodeTypeLink   NodeType = "link"
)

// Node represents a node in the virtual filesystem
type Node struct {
	ID         uint       `json:"id" db:"id"`
	MaterialID string     `json:"material_id" db:"material_id"` // UUID, public identifier for download/API
	UserID     uint       `json:"user_id" db:"user_id"`
	ParentID   *uint      `json:"parent_id" db:"parent_id"` // NULL = root
	Type       NodeType   `json:"type" db:"type"`
	Name       string     `json:"name" db:"name"`
	Hidden     bool       `json:"hidden" db:"hidden"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// File represents file metadata
type File struct {
	NodeID    uint      `json:"node_id" db:"node_id"`
	ObjectKey string    `json:"object_key" db:"object_key"`
	MimeType  string    `json:"mime_type" db:"mime_type"`
	Size      int64     `json:"size" db:"size"`
	Checksum  string    `json:"checksum" db:"checksum"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Link represents link metadata
type Link struct {
	NodeID      uint      `json:"node_id" db:"node_id"`
	URL         string    `json:"url" db:"url"`
	Title       *string   `json:"title,omitempty" db:"title"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// NodeWithDetails represents a node with its type-specific details
type NodeWithDetails struct {
	Node
	File *File `json:"file,omitempty"`
	Link *Link `json:"link,omitempty"`
}

// CreateFolderRequest represents request to create a folder
type CreateFolderRequest struct {
	ParentID *uint  `json:"parent_id" binding:"omitempty"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
}

// CreateFileRequest represents request to create a file
type CreateFileRequest struct {
	ParentID *uint `json:"parent_id" binding:"omitempty"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
	// File content is uploaded as multipart/form-data
}

// CreateLinkRequest represents request to create a link
type CreateLinkRequest struct {
	ParentID    *uint  `json:"parent_id" binding:"omitempty"`
	Name        string `json:"name" binding:"required,min=1,max=255"`
	URL         string `json:"url" binding:"required,url"`
	Title       string `json:"title" binding:"omitempty,max=255"`
	Description string `json:"description" binding:"omitempty,max=1000"`
}

// UpdateNodeRequest represents request to update a node (rename)
type UpdateNodeRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

// UpdateFileRequest represents request to update a file
type UpdateFileRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
	// New file content is uploaded as multipart/form-data
}

// UpdateLinkRequest represents request to update a link
type UpdateLinkRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	URL         string `json:"url" binding:"required,url"`
	Title       string `json:"title" binding:"omitempty,max=255"`
	Description string `json:"description" binding:"omitempty,max=1000"`
}

// ListNodesResponse represents response for listing nodes
type ListNodesResponse struct {
	Nodes []NodeWithDetails `json:"nodes"`
}

// GetNodeResponse represents response for getting a single node
type GetNodeResponse struct {
	Node NodeWithDetails `json:"node"`
}

// GetFileDownloadResponse represents response for file download URL
type GetFileDownloadResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// GetLinkResponse represents response for getting a link
type GetLinkResponse struct {
	URL         string  `json:"url"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// FileUploadResponse represents response for file upload
type FileUploadResponse struct {
	MaterialID  string `json:"material_id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type"`
	DownloadURL string `json:"download_url,omitempty"`
}

// FileMetaResponse represents response for file metadata
type FileMetaResponse struct {
	MaterialID string `json:"material_id"` // UUID
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mime_type"`
	UserID     uint   `json:"user_id"`
}
