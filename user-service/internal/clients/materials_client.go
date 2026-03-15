package clients

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbmaterials "proto/materials"
)

type MaterialsClient struct {
	client      pbmaterials.MaterialsServiceClient
	conn        *grpc.ClientConn
	internalKey string
	logger      *slog.Logger
}

func NewMaterialsClient(addr, internalKey string, logger *slog.Logger) *MaterialsClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to materials-service", "error", err)
		return nil
	}

	return &MaterialsClient{
		client:      pbmaterials.NewMaterialsServiceClient(conn),
		conn:        conn,
		internalKey: internalKey,
		logger:      logger,
	}
}

func (c *MaterialsClient) DeleteUserData(ctx context.Context, userID uint) error {
	md := metadata.Pairs("x-internal-api-key", c.internalKey)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.DeleteUserData(ctx, &pbmaterials.DeleteUserDataRequest{
		UserId: uint32(userID),
	})
	return err
}

func (c *MaterialsClient) UploadUserProfilePhoto(ctx context.Context, userID uint, content []byte, filename, mimeType string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("materials client is not initialized")
	}

	md := metadata.Pairs(
		"x-internal-api-key", c.internalKey,
		"x-user-id", fmt.Sprintf("%d", userID),
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	if filename == "" {
		filename = "profile_photo"
	}

	req := &pbmaterials.UploadFileRequest{
		FileContent: content,
		Filename:    filename,
		Name:        fmt.Sprintf("user_%d_profile_photo", userID),
	}

	resp, err := c.client.UploadFile(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.MaterialId, nil
}

func (c *MaterialsClient) DownloadFile(ctx context.Context, materialID string, userID uint) (*pbmaterials.DownloadFileResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("materials client is not initialized")
	}

	md := metadata.Pairs(
		"x-internal-api-key", c.internalKey,
		"x-user-id", fmt.Sprintf("%d", userID),
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	return c.client.DownloadFile(ctx, &pbmaterials.DownloadFileRequest{
		MaterialId: materialID,
	})
}
