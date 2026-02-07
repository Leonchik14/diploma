package clients

import (
	"context"
	"log/slog"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbcalendar "proto/calendar"
)

type CalendarClient struct {
	client      pbcalendar.CalendarServiceClient
	conn        *grpc.ClientConn
	internalKey string
	logger      *slog.Logger
}

func NewCalendarClient(addr, internalKey string, logger *slog.Logger) *CalendarClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to calendar-service", "error", err)
		return nil
	}

	return &CalendarClient{
		client:      pbcalendar.NewCalendarServiceClient(conn),
		conn:        conn,
		internalKey: internalKey,
		logger:      logger,
	}
}

func (c *CalendarClient) DeleteUserData(ctx context.Context, userID uint) error {
	md := metadata.Pairs("x-internal-api-key", c.internalKey, "x-user-id", strconv.FormatUint(uint64(userID), 10))
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.DeleteUserData(ctx, &pbcalendar.DeleteUserDataRequest{
		UserId: uint32(userID),
	})
	return err
}
