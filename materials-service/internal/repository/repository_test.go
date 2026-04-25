package repository

import (
	"context"
	"materials-service/internal/database"
	"materials-service/internal/models"
	"testing"
)

const testDSN = "postgres://postgres:password@localhost:5432/materials_test?sslmode=disable"

func setupTestDB(t *testing.T) {
	if err := database.Connect(testDSN); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	// Clean up test data
	database.DB.Exec(context.Background(), "TRUNCATE TABLE file_interactions CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE nodes CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE files CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE links CASCADE")
}

func teardownTestDB(t *testing.T) {
	database.DB.Exec(context.Background(), "TRUNCATE TABLE file_interactions CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE nodes CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE files CASCADE")
	database.DB.Exec(context.Background(), "TRUNCATE TABLE links CASCADE")
}

func TestCheckNameExists_UniqueInParent(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var parentID *uint = nil // Root level

	// Create first folder
	node1, err := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, "folder1", false)
	if err != nil {
		t.Fatalf("Failed to create first node: %v", err)
	}

	// Check that same name exists
	exists, err := repo.CheckNameExists(ctx, userID, parentID, "folder1", nil)
	if err != nil {
		t.Fatalf("Failed to check name existence: %v", err)
	}
	if !exists {
		t.Error("Expected name to exist, but it doesn't")
	}

	// Check different name doesn't exist
	exists, err = repo.CheckNameExists(ctx, userID, parentID, "folder2", nil)
	if err != nil {
		t.Fatalf("Failed to check name existence: %v", err)
	}
	if exists {
		t.Error("Expected name to not exist, but it does")
	}

	// Check that after soft-delete, name becomes available again
	err = repo.SoftDeleteNode(ctx, userID, node1.ID)
	if err != nil {
		t.Fatalf("Failed to soft-delete node: %v", err)
	}

	// Name should not exist anymore (soft-deleted)
	exists, err = repo.CheckNameExists(ctx, userID, parentID, "folder1", nil)
	if err != nil {
		t.Fatalf("Failed to check name existence: %v", err)
	}
	if exists {
		t.Error("Expected name to not exist after soft-delete, but it does")
	}
}

func TestCheckNameExists_UniqueAcrossTypes(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var parentID *uint = nil

	// Create a folder with name "test"
	_, err := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, "test", false)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Try to create a file with same name - should fail uniqueness check
	exists, err := repo.CheckNameExists(ctx, userID, parentID, "test", nil)
	if err != nil {
		t.Fatalf("Failed to check name existence: %v", err)
	}
	if !exists {
		t.Error("Expected name 'test' to exist (unique across types), but it doesn't")
	}
}

func TestSoftDeleteNode_Recursive(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var rootParentID *uint = nil

	// Create folder structure:
	// root/
	//   folder1/
	//     file1
	//     folder2/
	//       link1

	// Create root folder
	rootFolder, err := repo.CreateNode(ctx, userID, rootParentID, models.NodeTypeFolder, "root", false)
	if err != nil {
		t.Fatalf("Failed to create root folder: %v", err)
	}

	// Create folder1 inside root
	folder1, err := repo.CreateNode(ctx, userID, &rootFolder.ID, models.NodeTypeFolder, "folder1", false)
	if err != nil {
		t.Fatalf("Failed to create folder1: %v", err)
	}

	// Create file1 inside folder1
	file1, err := repo.CreateNode(ctx, userID, &folder1.ID, models.NodeTypeFile, "file1", false)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	// Create folder2 inside folder1
	folder2, err := repo.CreateNode(ctx, userID, &folder1.ID, models.NodeTypeFolder, "folder2", false)
	if err != nil {
		t.Fatalf("Failed to create folder2: %v", err)
	}

	// Create link1 inside folder2
	link1, err := repo.CreateNode(ctx, userID, &folder2.ID, models.NodeTypeLink, "link1", false)
	if err != nil {
		t.Fatalf("Failed to create link1: %v", err)
	}

	// Verify all nodes exist (not deleted)
	nodes := []*models.Node{rootFolder, folder1, file1, folder2, link1}
	for _, node := range nodes {
		checkNode, err := repo.GetNodeByID(ctx, userID, node.ID)
		if err != nil {
			t.Fatalf("Node %d should exist: %v", node.ID, err)
		}
		if checkNode.DeletedAt != nil {
			t.Errorf("Node %d should not be deleted yet", node.ID)
		}
	}

	// Soft-delete folder1 (should recursively delete file1, folder2, and link1)
	err = repo.SoftDeleteNode(ctx, userID, folder1.ID)
	if err != nil {
		t.Fatalf("Failed to soft-delete folder1: %v", err)
	}

	// Verify folder1 is deleted
	folder1Check, err := repo.GetNodeByID(ctx, userID, folder1.ID)
	if err == nil && folder1Check.DeletedAt == nil {
		t.Error("folder1 should be deleted")
	}

	// Verify file1 is deleted
	file1Check, err := repo.GetNodeByID(ctx, userID, file1.ID)
	if err == nil && file1Check.DeletedAt == nil {
		t.Error("file1 should be deleted (recursively)")
	}

	// Verify folder2 is deleted
	folder2Check, err := repo.GetNodeByID(ctx, userID, folder2.ID)
	if err == nil && folder2Check.DeletedAt == nil {
		t.Error("folder2 should be deleted (recursively)")
	}

	// Verify link1 is deleted
	link1Check, err := repo.GetNodeByID(ctx, userID, link1.ID)
	if err == nil && link1Check.DeletedAt == nil {
		t.Error("link1 should be deleted (recursively)")
	}

	// Verify root folder is NOT deleted
	rootCheck, err := repo.GetNodeByID(ctx, userID, rootFolder.ID)
	if err != nil {
		t.Fatalf("root folder should still exist: %v", err)
	}
	if rootCheck.DeletedAt != nil {
		t.Error("root folder should NOT be deleted")
	}

	// Verify all deleted nodes have deleted_at timestamp
	var count int
	err = database.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM nodes WHERE id IN ($1, $2, $3, $4) AND deleted_at IS NOT NULL",
		folder1.ID, file1.ID, folder2.ID, link1.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count deleted nodes: %v", err)
	}
	if count != 4 {
		t.Errorf("Expected 4 nodes to be deleted, but got %d", count)
	}
}

func TestGetNodeByID_DeletedNodesHidden(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var parentID *uint = nil

	// Create a node
	node, err := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, "test", false)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Should be able to get it
	_, err = repo.GetNodeByID(ctx, userID, node.ID)
	if err != nil {
		t.Fatalf("Should be able to get node: %v", err)
	}

	// Soft-delete it
	err = repo.SoftDeleteNode(ctx, userID, node.ID)
	if err != nil {
		t.Fatalf("Failed to soft-delete node: %v", err)
	}

	// Should NOT be able to get it (GetNodeByID filters by deleted_at IS NULL)
	_, err = repo.GetNodeByID(ctx, userID, node.ID)
	if err == nil {
		t.Error("Should not be able to get deleted node")
	}
}

func TestListChildren_DeletedNodesHidden(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var parentID *uint = nil

	// Create nodes
	node1, _ := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, "node1", false)
	node2, _ := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFolder, "node2", false)

	// List children - should see both
	children, err := repo.ListChildren(ctx, userID, parentID)
	if err != nil {
		t.Fatalf("Failed to list children: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}

	// Delete node1
	repo.SoftDeleteNode(ctx, userID, node1.ID)

	// List children - should see only node2
	children, err = repo.ListChildren(ctx, userID, parentID)
	if err != nil {
		t.Fatalf("Failed to list children: %v", err)
	}
	if len(children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(children))
	}
	if children[0].ID != node2.ID {
		t.Errorf("Expected node2, got node %d", children[0].ID)
	}
}

func TestListRecentFiles_ReturnsLastFiveUniqueFiles(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	repo := NewRepository()
	ctx := context.Background()
	userID := uint(1)
	var parentID *uint

	var fileIDs []uint
	for i := 0; i < 6; i++ {
		name := "file" + string(rune('A'+i))
		node, err := repo.CreateNode(ctx, userID, parentID, models.NodeTypeFile, name, false)
		if err != nil {
			t.Fatalf("Failed to create file node %s: %v", name, err)
		}
		if err := repo.CreateFile(ctx, node.ID, name, "text/plain", int64(i+1), name); err != nil {
			t.Fatalf("Failed to create file metadata %s: %v", name, err)
		}
		fileIDs = append(fileIDs, node.ID)
	}

	for _, nodeID := range []uint{fileIDs[0], fileIDs[1], fileIDs[2], fileIDs[3], fileIDs[4], fileIDs[5], fileIDs[2], fileIDs[4]} {
		if err := repo.RecordFileInteraction(ctx, userID, nodeID, "download"); err != nil {
			t.Fatalf("Failed to record interaction for node %d: %v", nodeID, err)
		}
	}

	nodes, err := repo.ListRecentFiles(ctx, userID, 5)
	if err != nil {
		t.Fatalf("Failed to list recent files: %v", err)
	}

	if len(nodes) != 5 {
		t.Fatalf("Expected 5 recent files, got %d", len(nodes))
	}

	expectedOrder := []uint{fileIDs[4], fileIDs[2], fileIDs[5], fileIDs[3], fileIDs[1]}
	for i, expectedID := range expectedOrder {
		if nodes[i].ID != expectedID {
			t.Fatalf("Expected node %d at position %d, got %d", expectedID, i, nodes[i].ID)
		}
	}
}
