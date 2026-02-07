package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"materials-service/internal/database"
	"materials-service/internal/models"

	"github.com/jackc/pgx/v5"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// GetNodeByIDWithoutUserCheck retrieves a node by ID without user check (for internal use)
func (r *Repository) GetNodeByIDWithoutUserCheck(ctx context.Context, nodeID uint) (*models.Node, error) {
	var node models.Node
	err := database.DB.QueryRow(ctx,
		`SELECT id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at
		 FROM nodes WHERE id = $1 AND deleted_at IS NULL`,
		nodeID).Scan(
		&node.ID, &node.MaterialID, &node.UserID, &node.ParentID, &node.Type, &node.Name,
		&node.CreatedAt, &node.UpdatedAt, &node.DeletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("node not found")
		}
		return nil, err
	}

	return &node, nil
}

// GetNodeByID retrieves a node by ID (must belong to user)
func (r *Repository) GetNodeByID(ctx context.Context, userID, nodeID uint) (*models.Node, error) {
	var node models.Node
	err := database.DB.QueryRow(ctx,
		`SELECT id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at 
		 FROM nodes WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		nodeID, userID).Scan(
		&node.ID, &node.MaterialID, &node.UserID, &node.ParentID, &node.Type, &node.Name,
		&node.CreatedAt, &node.UpdatedAt, &node.DeletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("node not found")
		}
		return nil, err
	}

	return &node, nil
}

// GetNodeByMaterialID retrieves a node by material_id UUID (must belong to user)
func (r *Repository) GetNodeByMaterialID(ctx context.Context, userID uint, materialID string) (*models.Node, error) {
	var node models.Node
	err := database.DB.QueryRow(ctx,
		`SELECT id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at 
		 FROM nodes WHERE material_id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		materialID, userID).Scan(
		&node.ID, &node.MaterialID, &node.UserID, &node.ParentID, &node.Type, &node.Name,
		&node.CreatedAt, &node.UpdatedAt, &node.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("node not found")
		}
		return nil, err
	}
	return &node, nil
}

// ListChildren retrieves all children of a parent node (for user)
func (r *Repository) ListChildren(ctx context.Context, userID uint, parentID *uint) ([]models.Node, error) {
	var rows pgx.Rows
	var err error

	if parentID == nil {
		// Root level
		rows, err = database.DB.Query(ctx,
			`SELECT id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at 
			 FROM nodes WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL 
			 ORDER BY type, name`,
			userID)
	} else {
		rows, err = database.DB.Query(ctx,
			`SELECT id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at 
			 FROM nodes WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL 
			 ORDER BY type, name`,
			userID, *parentID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var node models.Node
		if err := rows.Scan(
			&node.ID, &node.MaterialID, &node.UserID, &node.ParentID, &node.Type, &node.Name,
			&node.CreatedAt, &node.UpdatedAt, &node.DeletedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// CheckNameExists checks if a name already exists in parent (for user, among active nodes)
func (r *Repository) CheckNameExists(ctx context.Context, userID uint, parentID *uint, name string, excludeID *uint) (bool, error) {
	var exists bool
	var err error

	if excludeID != nil {
		if parentID == nil {
			err = database.DB.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM nodes 
				 WHERE user_id = $1 AND parent_id IS NULL AND name = $2 
				 AND deleted_at IS NULL AND id != $3)`,
				userID, name, *excludeID).Scan(&exists)
		} else {
			err = database.DB.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM nodes 
				 WHERE user_id = $1 AND parent_id = $2 AND name = $3 
				 AND deleted_at IS NULL AND id != $4)`,
				userID, *parentID, name, *excludeID).Scan(&exists)
		}
	} else {
		if parentID == nil {
			err = database.DB.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM nodes 
				 WHERE user_id = $1 AND parent_id IS NULL AND name = $2 
				 AND deleted_at IS NULL)`,
				userID, name).Scan(&exists)
		} else {
			err = database.DB.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM nodes 
				 WHERE user_id = $1 AND parent_id = $2 AND name = $3 
				 AND deleted_at IS NULL)`,
				userID, *parentID, name).Scan(&exists)
		}
	}

	return exists, err
}

// CreateNode creates a new node
func (r *Repository) CreateNode(ctx context.Context, userID uint, parentID *uint, nodeType models.NodeType, name string) (*models.Node, error) {
	var node models.Node
	err := database.DB.QueryRow(ctx,
		`INSERT INTO nodes (user_id, parent_id, type, name) 
		 VALUES ($1, $2, $3, $4) 
		 RETURNING id, material_id, user_id, parent_id, type, name, created_at, updated_at, deleted_at`,
		userID, parentID, nodeType, name).Scan(
		&node.ID, &node.MaterialID, &node.UserID, &node.ParentID, &node.Type, &node.Name,
		&node.CreatedAt, &node.UpdatedAt, &node.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &node, nil
}

// UpdateNodeName updates node name
func (r *Repository) UpdateNodeName(ctx context.Context, userID, nodeID uint, name string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE nodes SET name = $1 WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL`,
		name, nodeID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("node not found or already deleted")
	}

	return nil
}

// SoftDeleteNode performs recursive soft delete
func (r *Repository) SoftDeleteNode(ctx context.Context, userID, nodeID uint) error {
	// Check node exists and belongs to user
	_, err := r.GetNodeByID(ctx, userID, nodeID)
	if err != nil {
		return err
	}

	// Call recursive soft delete function
	_, err = database.DB.Exec(ctx,
		`SELECT recursive_soft_delete($1, $2)`,
		nodeID, userID)
	if err != nil {
		return err
	}

	return nil
}

// GetFile retrieves file metadata by node_id
func (r *Repository) GetFile(ctx context.Context, nodeID uint) (*models.File, error) {
	var file models.File
	err := database.DB.QueryRow(ctx,
		`SELECT node_id, object_key, mime_type, size, checksum, created_at 
		 FROM files WHERE node_id = $1`,
		nodeID).Scan(
		&file.NodeID, &file.ObjectKey, &file.MimeType, &file.Size, &file.Checksum, &file.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, err
	}

	return &file, nil
}

// CreateFile creates file metadata
func (r *Repository) CreateFile(ctx context.Context, nodeID uint, objectKey, mimeType string, size int64, checksum string) error {
	_, err := database.DB.Exec(ctx,
		`INSERT INTO files (node_id, object_key, mime_type, size, checksum) 
		 VALUES ($1, $2, $3, $4, $5)`,
		nodeID, objectKey, mimeType, size, checksum)
	return err
}

// UpdateFile updates file metadata
func (r *Repository) UpdateFile(ctx context.Context, nodeID uint, objectKey, mimeType string, size int64, checksum string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE files SET object_key = $1, mime_type = $2, size = $3, checksum = $4 
		 WHERE node_id = $5`,
		objectKey, mimeType, size, checksum, nodeID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// GetLink retrieves link metadata by node_id
func (r *Repository) GetLink(ctx context.Context, nodeID uint) (*models.Link, error) {
	var link models.Link
	var title, description sql.NullString

	err := database.DB.QueryRow(ctx,
		`SELECT node_id, url, title, description, created_at 
		 FROM links WHERE node_id = $1`,
		nodeID).Scan(
		&link.NodeID, &link.URL, &title, &description, &link.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("link not found")
		}
		return nil, err
	}

	if title.Valid {
		link.Title = &title.String
	}
	if description.Valid {
		link.Description = &description.String
	}

	return &link, nil
}

// CreateLink creates link metadata
func (r *Repository) CreateLink(ctx context.Context, nodeID uint, url string, title, description *string) error {
	_, err := database.DB.Exec(ctx,
		`INSERT INTO links (node_id, url, title, description) 
		 VALUES ($1, $2, $3, $4)`,
		nodeID, url, title, description)
	return err
}

// UpdateLink updates link metadata
func (r *Repository) UpdateLink(ctx context.Context, nodeID uint, url string, title, description *string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE links SET url = $1, title = $2, description = $3 WHERE node_id = $4`,
		url, title, description, nodeID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("link not found")
	}

	return nil
}

func (r *Repository) DeleteUserData(ctx context.Context, userID uint) error {
	rows, err := database.DB.Query(ctx,
		`SELECT id FROM nodes WHERE user_id = $1 AND deleted_at IS NULL`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var nodeIDs []uint
	for rows.Next() {
		var nodeID uint
		if err := rows.Scan(&nodeID); err != nil {
			return err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}

	for _, nodeID := range nodeIDs {
		if err := r.SoftDeleteNode(ctx, userID, nodeID); err != nil {
			return err
		}
	}

	return nil
}
