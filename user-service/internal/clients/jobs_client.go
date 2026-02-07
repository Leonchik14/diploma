package clients

import (
	"context"
	"log/slog"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbjobs "proto/jobs"
)

type JobsClient struct {
	client      pbjobs.JobsServiceClient
	conn        *grpc.ClientConn
	internalKey string
	logger      *slog.Logger
}

func NewJobsClient(addr, internalKey string, logger *slog.Logger) *JobsClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to jobs-service", "error", err)
		return nil
	}

	return &JobsClient{
		client:      pbjobs.NewJobsServiceClient(conn),
		conn:        conn,
		internalKey: internalKey,
		logger:      logger,
	}
}

func (c *JobsClient) DeleteUserData(ctx context.Context, userID uint) error {
	md := metadata.Pairs("x-internal-api-key", c.internalKey, "x-user-id", strconv.FormatUint(uint64(userID), 10))
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.DeleteUserData(ctx, &pbjobs.DeleteUserDataRequest{
		UserId: uint32(userID),
	})
	return err
}
