package clients

import (
	"context"
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
