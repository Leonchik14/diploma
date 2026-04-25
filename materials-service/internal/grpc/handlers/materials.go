package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"

	"materials-service/internal/models"
	"materials-service/internal/requestctx"
	"materials-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbmaterials "proto/materials"
)

type MaterialsHandler struct {
	pbmaterials.UnimplementedMaterialsServiceServer
	service *service.Service
	logger  *slog.Logger
}

func NewMaterialsHandler(svc *service.Service, logger *slog.Logger) *MaterialsHandler {
	return &MaterialsHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *MaterialsHandler) getUserID(ctx context.Context) (uint, error) {
	userID, ok := requestctx.UserID(ctx)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func (h *MaterialsHandler) UploadFile(ctx context.Context, req *pbmaterials.UploadFileRequest) (*pbmaterials.UploadFileResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.FileContent) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "file_content is required")
	}

	if req.Filename == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filename is required")
	}

	var parentID *uint
	if req.ParentId != nil {
		pid := uint(*req.ParentId)
		parentID = &pid
	}

	name := req.Filename
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}

	mimeType := detectMimeType(req.Filename)

	fileReader := bytes.NewReader(req.FileContent)
	node, err := h.service.CreateFile(ctx, userID, parentID, name, fileReader, int64(len(req.FileContent)), mimeType, req.Hidden)
	if err != nil {
		h.logger.Error("failed to create file", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
	}

	// Get file details
	nodeWithDetails, err := h.service.GetNode(ctx, userID, node.ID)
	if err != nil {
		h.logger.Error("failed to get file details", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get file details")
	}

	if nodeWithDetails.File == nil {
		return nil, status.Errorf(codes.Internal, "file metadata not found")
	}

	return &pbmaterials.UploadFileResponse{
		MaterialId: node.MaterialID,
		Name:       node.Name,
		Size:       nodeWithDetails.File.Size,
		MimeType:   nodeWithDetails.File.MimeType,
	}, nil
}

func detectMimeType(filename string) string {
	ext := getFileExtension(filename)
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".rtf":  "application/rtf",
	}
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}
	return "application/octet-stream"
}

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

func (h *MaterialsHandler) DownloadFile(ctx context.Context, req *pbmaterials.DownloadFileRequest) (*pbmaterials.DownloadFileResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.MaterialId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "material_id is required")
	}

	// Resolve material_id (UUID) to node
	node, err := h.service.GetNodeByMaterialID(ctx, userID, req.MaterialId)
	if err != nil {
		h.logger.Error("failed to get node by material_id", "error", err, "material_id", req.MaterialId)
		return nil, status.Errorf(codes.NotFound, "file not found")
	}
	if node.Type != models.NodeTypeFile {
		return nil, status.Errorf(codes.InvalidArgument, "material is not a file")
	}

	// Get file stream
	stream, file, err := h.service.GetFileStream(ctx, node.ID)
	if err != nil {
		h.logger.Error("failed to get file stream", "error", err, "node_id", node.ID)
		return nil, status.Errorf(codes.NotFound, "file not found")
	}
	defer stream.Close()

	// Verify ownership
	ownerID, err := h.service.GetFileOwner(ctx, node.ID)
	if err != nil {
		h.logger.Error("failed to get file owner", "error", err)
		return nil, status.Errorf(codes.NotFound, "file not found")
	}

	if ownerID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	// Read file content
	content, err := io.ReadAll(stream)
	if err != nil {
		h.logger.Error("failed to read file content", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to read file")
	}

	filename := node.Name
	if filename == "" {
		filename = "file"
	}

	// Добавляем base64 версию для удобства передачи через gRPC
	contentBase64 := base64.StdEncoding.EncodeToString(content)

	if err := h.service.RecordFileInteractionByMaterialID(ctx, userID, req.MaterialId, "download"); err != nil {
		h.logger.Error("failed to record file interaction", "error", err, "material_id", req.MaterialId)
		return nil, status.Errorf(codes.Internal, "failed to record recent file interaction")
	}

	return &pbmaterials.DownloadFileResponse{
		Content:       content,
		Filename:      filename,
		MimeType:      file.MimeType,
		Size:          file.Size,
		ContentBase64: &contentBase64,
	}, nil
}

func (h *MaterialsHandler) ListFolder(ctx context.Context, req *pbmaterials.ListFolderRequest) (*pbmaterials.ListFolderResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	var parentID *uint
	if req.ParentId != nil {
		pid := uint(*req.ParentId)
		parentID = &pid
	}

	nodes, err := h.service.ListChildren(ctx, userID, parentID)
	if err != nil {
		h.logger.Error("failed to list folder", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list folder: %v", err)
	}

	pbNodes := make([]*pbmaterials.Node, len(nodes))
	for i := range nodes {
		pbNodes[i] = convertNodeToProto(&nodes[i])
	}

	return &pbmaterials.ListFolderResponse{Nodes: pbNodes}, nil
}

func (h *MaterialsHandler) RecentFiles(ctx context.Context, req *pbmaterials.RecentFilesRequest) (*pbmaterials.RecentFilesResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	nodes, err := h.service.ListRecentFiles(ctx, userID)
	if err != nil {
		h.logger.Error("failed to list recent files", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list recent files: %v", err)
	}

	pbNodes := make([]*pbmaterials.Node, len(nodes))
	for i := range nodes {
		pbNodes[i] = convertNodeToProto(&nodes[i])
	}

	return &pbmaterials.RecentFilesResponse{Nodes: pbNodes}, nil
}

func (h *MaterialsHandler) CreateFolder(ctx context.Context, req *pbmaterials.CreateFolderRequest) (*pbmaterials.CreateFolderResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}

	var parentID *uint
	if req.ParentId != nil {
		pid := uint(*req.ParentId)
		parentID = &pid
	}

	node, err := h.service.CreateFolder(ctx, userID, parentID, req.Name)
	if err != nil {
		h.logger.Error("failed to create folder", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create folder: %v", err)
	}

	nodeWithDetails, err := h.service.GetNode(ctx, userID, node.ID)
	if err != nil {
		h.logger.Error("failed to get folder details", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get folder details")
	}

	return &pbmaterials.CreateFolderResponse{Node: convertNodeToProto(nodeWithDetails)}, nil
}

func (h *MaterialsHandler) CreateLink(ctx context.Context, req *pbmaterials.CreateLinkRequest) (*pbmaterials.CreateLinkResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	if req.Url == "" {
		return nil, status.Errorf(codes.InvalidArgument, "url is required")
	}

	var parentID *uint
	if req.ParentId != nil {
		pid := uint(*req.ParentId)
		parentID = &pid
	}

	var title, description *string
	if req.Title != nil && *req.Title != "" {
		title = req.Title
	}
	if req.Description != nil && *req.Description != "" {
		description = req.Description
	}

	node, err := h.service.CreateLink(ctx, userID, parentID, req.Name, req.Url, title, description)
	if err != nil {
		h.logger.Error("failed to create link", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create link: %v", err)
	}

	nodeWithDetails, err := h.service.GetNode(ctx, userID, node.ID)
	if err != nil {
		h.logger.Error("failed to get link details", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get link details")
	}

	return &pbmaterials.CreateLinkResponse{Node: convertNodeToProto(nodeWithDetails)}, nil
}

func (h *MaterialsHandler) RenameNode(ctx context.Context, req *pbmaterials.RenameNodeRequest) (*pbmaterials.RenameNodeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.NodeId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "node_id is required")
	}
	if req.NewName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "new_name is required")
	}

	if err := h.service.UpdateNodeName(ctx, userID, uint(req.NodeId), req.NewName); err != nil {
		h.logger.Error("failed to rename node", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to rename node: %v", err)
	}

	return &pbmaterials.RenameNodeResponse{Success: true}, nil
}

func (h *MaterialsHandler) DeleteNode(ctx context.Context, req *pbmaterials.DeleteNodeRequest) (*pbmaterials.DeleteNodeResponse, error) {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.NodeId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "node_id is required")
	}

	if err := h.service.DeleteNode(ctx, userID, uint(req.NodeId)); err != nil {
		h.logger.Error("failed to delete node", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete node: %v", err)
	}

	return &pbmaterials.DeleteNodeResponse{Success: true}, nil
}

func (h *MaterialsHandler) DeleteUserData(ctx context.Context, req *pbmaterials.DeleteUserDataRequest) (*pbmaterials.DeleteUserDataResponse, error) {
	userID := uint(req.UserId)

	if err := h.service.DeleteUserData(ctx, userID); err != nil {
		h.logger.Error("failed to delete user data", "user_id", userID, "error", err)
		return &pbmaterials.DeleteUserDataResponse{Ok: true}, nil
	}

	return &pbmaterials.DeleteUserDataResponse{Ok: true}, nil
}

func convertNodeToProto(node *models.NodeWithDetails) *pbmaterials.Node {
	pbNode := &pbmaterials.Node{
		Id:         uint32(node.ID),
		UserId:     uint32(node.UserID),
		Type:       string(node.Type),
		Name:       node.Name,
		CreatedAt:  node.CreatedAt.Unix(),
		UpdatedAt:  node.UpdatedAt.Unix(),
		MaterialId: &node.MaterialID,
	}

	if node.ParentID != nil {
		pid := uint32(*node.ParentID)
		pbNode.ParentId = &pid
	}

	if node.File != nil {
		pbNode.File = &pbmaterials.FileInfo{
			ObjectKey: node.File.ObjectKey,
			MimeType:  node.File.MimeType,
			Size:      node.File.Size,
		}
	}

	if node.Link != nil {
		pbLink := &pbmaterials.LinkInfo{
			Url: node.Link.URL,
		}
		if node.Link.Title != nil {
			pbLink.Title = node.Link.Title
		}
		if node.Link.Description != nil {
			pbLink.Description = node.Link.Description
		}
		pbNode.Link = pbLink
	}

	return pbNode
}
