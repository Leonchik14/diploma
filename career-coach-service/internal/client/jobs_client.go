package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbjobs "proto/jobs"
)

type JobsClient struct {
	conn        *grpc.ClientConn
	client      pbjobs.JobsServiceClient
	internalKey string
}

func NewJobsClient(grpcAddr, internalKey string, timeout time.Duration) *JobsClient {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to job-service: %v", err))
	}

	return &JobsClient{
		conn:        conn,
		client:      pbjobs.NewJobsServiceClient(conn),
		internalKey: internalKey,
	}
}

func (c *JobsClient) GetVacancy(ctx context.Context, vacancyID string) (*pbjobs.Vacancy, error) {
	md := metadata.New(map[string]string{
		"x-internal-api-key": c.internalKey,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := c.client.GetVacancy(ctx, &pbjobs.GetVacancyRequest{
		VacancyId: vacancyID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Vacancy, nil
}

func (c *JobsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
