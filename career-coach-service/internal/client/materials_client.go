package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbmaterials "proto/materials"
)

type MaterialsClient struct {
	conn         *grpc.ClientConn
	client       pbmaterials.MaterialsServiceClient
	internalKey  string
}

func NewMaterialsClient(grpcAddr, internalKey string, timeout time.Duration) *MaterialsClient {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to materials-service: %v", err))
	}

	return &MaterialsClient{
		conn:        conn,
		client:      pbmaterials.NewMaterialsServiceClient(conn),
		internalKey: internalKey,
	}
}

func (c *MaterialsClient) UploadFile(ctx context.Context, fileContent []byte, filename string, userID uint, hidden bool) (string, error) {
	md := metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
		"x-user-id":          fmt.Sprintf("%d", userID),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := c.client.UploadFile(ctx, &pbmaterials.UploadFileRequest{
		FileContent: fileContent,
		Filename:    filename,
		Hidden:      hidden,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return resp.MaterialId, nil
}

func (c *MaterialsClient) DownloadFile(ctx context.Context, materialID string, userID uint) (io.ReadCloser, string, error) {
	// Add internal API key and user ID to metadata
	md := metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
		"x-user-id":          fmt.Sprintf("%d", userID),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Call gRPC DownloadFile
	resp, err := c.client.DownloadFile(ctx, &pbmaterials.DownloadFileRequest{
		MaterialId: materialID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}

	// Convert bytes to ReadCloser
	reader := io.NopCloser(bytes.NewReader(resp.Content))

	return reader, resp.MimeType, nil
}

func (c *MaterialsClient) DeleteByMaterialID(ctx context.Context, materialID string, userID uint) error {
	md := metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
		"x-user-id":          fmt.Sprintf("%d", userID),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	if _, err := c.client.DeleteByMaterialID(ctx, &pbmaterials.DeleteByMaterialIDRequest{
		MaterialId: materialID,
	}); err != nil {
		return fmt.Errorf("failed to delete file by material id: %w", err)
	}

	return nil
}

func (c *MaterialsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
