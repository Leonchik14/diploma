package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"materials-service/internal/models"
	"materials-service/internal/repository"
	"materials-service/internal/storage"

	"github.com/google/uuid"
)

type Service struct {
	repo    *repository.Repository
	storage *storage.Storage
}

const recentFilesLimit = 5

func NewService(repo *repository.Repository, storage *storage.Storage) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
	}
}

// CreateFolder creates a folder
func (s *Service) CreateFolder(ctx context.Context, userID uint, parentID *uint, name string) (*models.Node, error) {
	// Check if parent exists and belongs to user (if parentID is not nil)
	if parentID != nil {
		parent, err := s.repo.GetNodeByID(ctx, userID, *parentID)
		if err != nil {
			return nil, fmt.Errorf("parent folder not found: %w", err)
		}
		if parent.Type != models.NodeTypeFolder {
			return nil, fmt.Errorf("parent is not a folder")
		}
	}

	// Check name uniqueness
	exists, err := s.repo.CheckNameExists(ctx, userID, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("name already exists in this folder")
	}

	return s.repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, name, false)
}

// CreateFile creates a file node and uploads content to MinIO
func (s *Service) CreateFile(ctx context.Context, userID uint, parentID *uint, name string, content io.Reader, size int64, mimeType string, hidden bool) (*models.Node, error) {
	// Check if parent exists and belongs to user (if parentID is not nil)
	if parentID != nil {
		parent, err := s.repo.GetNodeByID(ctx, userID, *parentID)
		if err != nil {
			return nil, fmt.Errorf("parent folder not found: %w", err)
		}
		if parent.Type != models.NodeTypeFolder {
			return nil, fmt.Errorf("parent is not a folder")
		}
	}

	// Check name uniqueness
	exists, err := s.repo.CheckNameExists(ctx, userID, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("name already exists in this folder")
	}

	node, err := s.repo.CreateNode(ctx, userID, parentID, models.NodeTypeFile, name, hidden)
	if err != nil {
		return nil, err
	}

	objectKey := s.generateObjectKey(userID, node.ID)

	// Calculate checksum while reading
	hash := md5.New()
	multiReader := io.TeeReader(content, hash)

	// Upload to MinIO
	if err := s.storage.PutObject(ctx, objectKey, multiReader, size, mimeType); err != nil {
		// If upload fails, we should rollback node creation
		// For simplicity, we'll leave the node but mark it as deleted or handle cleanup separately
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// Create file metadata
	if err := s.repo.CreateFile(ctx, node.ID, objectKey, mimeType, size, checksum); err != nil {
		return nil, fmt.Errorf("failed to create file metadata: %w", err)
	}

	if err := s.repo.RecordFileInteraction(ctx, userID, node.ID, "upload"); err != nil {
		return nil, fmt.Errorf("failed to record file interaction: %w", err)
	}

	return node, nil
}

// CreateLink creates a link node
func (s *Service) CreateLink(ctx context.Context, userID uint, parentID *uint, name, url string, title, description *string) (*models.Node, error) {
	// Check if parent exists and belongs to user (if parentID is not nil)
	if parentID != nil {
		parent, err := s.repo.GetNodeByID(ctx, userID, *parentID)
		if err != nil {
			return nil, fmt.Errorf("parent folder not found: %w", err)
		}
		if parent.Type != models.NodeTypeFolder {
			return nil, fmt.Errorf("parent is not a folder")
		}
	}

	// Check name uniqueness
	exists, err := s.repo.CheckNameExists(ctx, userID, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("name already exists in this folder")
	}

	node, err := s.repo.CreateNode(ctx, userID, parentID, models.NodeTypeLink, name, false)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateLink(ctx, node.ID, url, title, description); err != nil {
		return nil, fmt.Errorf("failed to create link metadata: %w", err)
	}

	return node, nil
}

// ListChildren lists children of a parent node
func (s *Service) ListChildren(ctx context.Context, userID uint, parentID *uint) ([]models.NodeWithDetails, error) {
	nodes, err := s.repo.ListChildren(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}

	result := make([]models.NodeWithDetails, 0, len(nodes))
	for _, node := range nodes {
		nodeWithDetails := models.NodeWithDetails{Node: node}

		switch node.Type {
		case models.NodeTypeFile:
			file, err := s.repo.GetFile(ctx, node.ID)
			if err == nil {
				nodeWithDetails.File = file
			}
		case models.NodeTypeLink:
			link, err := s.repo.GetLink(ctx, node.ID)
			if err == nil {
				nodeWithDetails.Link = link
			}
		}

		result = append(result, nodeWithDetails)
	}

	return result, nil
}

// ListRecentFiles returns up to 5 recently interacted files ordered by latest interaction.
func (s *Service) ListRecentFiles(ctx context.Context, userID uint) ([]models.NodeWithDetails, error) {
	nodes, err := s.repo.ListRecentFiles(ctx, userID, recentFilesLimit)
	if err != nil {
		return nil, err
	}

	result := make([]models.NodeWithDetails, 0, len(nodes))
	for _, node := range nodes {
		nodeWithDetails := models.NodeWithDetails{Node: node}
		file, err := s.repo.GetFile(ctx, node.ID)
		if err == nil {
			nodeWithDetails.File = file
		}
		result = append(result, nodeWithDetails)
	}

	return result, nil
}

// GetNode retrieves a node with its details
func (s *Service) GetNode(ctx context.Context, userID, nodeID uint) (*models.NodeWithDetails, error) {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return nil, err
	}

	nodeWithDetails := models.NodeWithDetails{Node: *node}

	switch node.Type {
	case models.NodeTypeFile:
		file, err := s.repo.GetFile(ctx, node.ID)
		if err == nil {
			nodeWithDetails.File = file
		}
	case models.NodeTypeLink:
		link, err := s.repo.GetLink(ctx, node.ID)
		if err == nil {
			nodeWithDetails.Link = link
		}
	}

	return &nodeWithDetails, nil
}

// GetNodeByMaterialID retrieves a node by material_id UUID (must belong to user)
func (s *Service) GetNodeByMaterialID(ctx context.Context, userID uint, materialID string) (*models.Node, error) {
	return s.repo.GetNodeByMaterialID(ctx, userID, materialID)
}

// RecordFileInteractionByNode records an interaction if the node is an active file.
func (s *Service) RecordFileInteractionByNode(ctx context.Context, userID, nodeID uint, interactionType string) error {
	nodeType, err := s.repo.GetNodeTypeForUser(ctx, userID, nodeID)
	if err != nil {
		return err
	}
	if nodeType != models.NodeTypeFile {
		return nil
	}

	return s.repo.RecordFileInteraction(ctx, userID, nodeID, interactionType)
}

// RecordFileInteractionByMaterialID records an interaction for a file resolved by material_id.
func (s *Service) RecordFileInteractionByMaterialID(ctx context.Context, userID uint, materialID, interactionType string) error {
	node, err := s.repo.GetNodeByMaterialID(ctx, userID, materialID)
	if err != nil {
		return err
	}
	if node.Type != models.NodeTypeFile {
		return nil
	}

	return s.repo.RecordFileInteraction(ctx, userID, node.ID, interactionType)
}

// UpdateNodeName updates node name
func (s *Service) UpdateNodeName(ctx context.Context, userID, nodeID uint, name string) error {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return err
	}

	// Check name uniqueness (excluding current node)
	excludeID := &nodeID
	exists, err := s.repo.CheckNameExists(ctx, userID, node.ParentID, name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("name already exists in this folder")
	}

	if err := s.repo.UpdateNodeName(ctx, userID, nodeID, name); err != nil {
		return err
	}

	if node.Type == models.NodeTypeFile {
		if err := s.repo.RecordFileInteraction(ctx, userID, nodeID, "rename"); err != nil {
			return fmt.Errorf("failed to record file interaction: %w", err)
		}
	}

	return nil
}

// UpdateFile updates file content and metadata
func (s *Service) UpdateFile(ctx context.Context, userID, nodeID uint, name string, content io.Reader, size int64, mimeType string) error {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return err
	}
	if node.Type != models.NodeTypeFile {
		return fmt.Errorf("node is not a file")
	}

	// Check name uniqueness if name changed
	if name != node.Name {
		excludeID := &nodeID
		exists, err := s.repo.CheckNameExists(ctx, userID, node.ParentID, name, excludeID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("name already exists in this folder")
		}
	}

	// Generate new object key
	newObjectKey := s.generateObjectKey(userID, nodeID)

	// Calculate checksum while reading
	hash := md5.New()
	multiReader := io.TeeReader(content, hash)

	// Upload new content to MinIO
	if err := s.storage.PutObject(ctx, newObjectKey, multiReader, size, mimeType); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// Update file metadata
	if err := s.repo.UpdateFile(ctx, nodeID, newObjectKey, mimeType, size, checksum); err != nil {
		return err
	}

	// Update node name if changed
	if name != node.Name {
		if err := s.repo.UpdateNodeName(ctx, userID, nodeID, name); err != nil {
			return err
		}
	}

	// TODO: Delete old file from MinIO in background (oldObjectKey)

	return nil
}

// UpdateLink updates link metadata
func (s *Service) UpdateLink(ctx context.Context, userID, nodeID uint, name, url string, title, description *string) error {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return err
	}
	if node.Type != models.NodeTypeLink {
		return fmt.Errorf("node is not a link")
	}

	// Check name uniqueness if name changed
	if name != node.Name {
		excludeID := &nodeID
		exists, err := s.repo.CheckNameExists(ctx, userID, node.ParentID, name, excludeID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("name already exists in this folder")
		}
	}

	// Update link metadata
	if err := s.repo.UpdateLink(ctx, nodeID, url, title, description); err != nil {
		return err
	}

	// Update node name if changed
	if name != node.Name {
		if err := s.repo.UpdateNodeName(ctx, userID, nodeID, name); err != nil {
			return err
		}
	}

	return nil
}

// DeleteNode performs recursive soft delete
func (s *Service) DeleteNode(ctx context.Context, userID, nodeID uint) error {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return err
	}

	if node.Type == models.NodeTypeFile {
		if err := s.repo.RecordFileInteraction(ctx, userID, nodeID, "delete"); err != nil {
			return fmt.Errorf("failed to record file interaction: %w", err)
		}
	}

	return s.repo.SoftDeleteNode(ctx, userID, nodeID)
}

// DeleteByMaterialID deletes a file or node resolved by material_id for the user.
func (s *Service) DeleteByMaterialID(ctx context.Context, userID uint, materialID string) error {
	node, err := s.repo.GetNodeByMaterialID(ctx, userID, materialID)
	if err != nil {
		return err
	}

	return s.DeleteNode(ctx, userID, node.ID)
}

// GetFileDownloadURL generates a presigned URL for file download
func (s *Service) GetFileDownloadURL(ctx context.Context, userID, nodeID uint) (string, error) {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return "", err
	}
	if node.Type != models.NodeTypeFile {
		return "", fmt.Errorf("node is not a file")
	}

	file, err := s.repo.GetFile(ctx, nodeID)
	if err != nil {
		return "", err
	}

	url, err := s.storage.PresignedGetObject(ctx, file.ObjectKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

// GetLinkDetails retrieves link details
func (s *Service) GetLinkDetails(ctx context.Context, userID, nodeID uint) (*models.Link, error) {
	node, err := s.repo.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Type != models.NodeTypeLink {
		return nil, fmt.Errorf("node is not a link")
	}

	return s.repo.GetLink(ctx, nodeID)
}

// GetFileStream retrieves file stream from MinIO (for internal use)
func (s *Service) GetFileStream(ctx context.Context, nodeID uint) (io.ReadCloser, *models.File, error) {
	file, err := s.repo.GetFile(ctx, nodeID)
	if err != nil {
		return nil, nil, err
	}

	obj, err := s.storage.GetObject(ctx, file.ObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file from storage: %w", err)
	}

	return obj, file, nil
}

// GetFileOwner checks if file exists and returns owner user_id
func (s *Service) GetFileOwner(ctx context.Context, nodeID uint) (uint, error) {
	node, err := s.repo.GetNodeByIDWithoutUserCheck(ctx, nodeID)
	if err != nil {
		return 0, err
	}
	if node.Type != models.NodeTypeFile {
		return 0, fmt.Errorf("node is not a file")
	}
	return node.UserID, nil
}

// generateObjectKey generates a safe object key for MinIO
func (s *Service) generateObjectKey(userID, nodeID uint) string {
	randomUUID := uuid.New().String()
	return fmt.Sprintf("user/%d/%d/%s", userID, nodeID, randomUUID)
}

// DeleteUserData deletes all user data (nodes, files from MinIO)
func (s *Service) DeleteUserData(ctx context.Context, userID uint) error {
	return s.repo.DeleteUserData(ctx, userID)
}
