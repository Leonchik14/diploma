package auth

import (
	"context"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	MetadataUserID        = "x-user-id"
	MetadataInternalAPIKey = "x-internal-api-key"
)

func UserIDFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(MetadataUserID)
	if len(vals) == 0 || vals[0] == "" {
		return "", status.Errorf(codes.Unauthenticated, "missing x-user-id")
	}
	return vals[0], nil
}

func UserIDUintFromMetadata(ctx context.Context) (uint, error) {
	s, err := UserIDFromMetadata(ctx)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid x-user-id format")
	}
	return uint(n), nil
}

func InternalKeyFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(MetadataInternalAPIKey)
	if len(vals) == 0 || vals[0] == "" {
		return "", status.Errorf(codes.Unauthenticated, "missing x-internal-api-key")
	}
	return vals[0], nil
}
