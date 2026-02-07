package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	publicKey     interface{}
	internalKey   string
	logger        *slog.Logger
	skipAuthPaths map[string]bool
}

func NewAuthInterceptor(publicKey interface{}, internalKey string, logger *slog.Logger) *AuthInterceptor {
	skipPaths := map[string]bool{
		"/gateway.BackendGateway/Register":             true,
		"/gateway.BackendGateway/Login":               true,
		"/gateway.BackendGateway/Refresh":              true,
		"/gateway.BackendGateway/CheckPasswordResetEmail": true,
		"/gateway.BackendGateway/SendPasswordResetCode": true,
		"/gateway.BackendGateway/VerifyPasswordReset":  true,
	}

	return &AuthInterceptor{
		publicKey:    publicKey,
		internalKey:  internalKey,
		logger:       logger,
		skipAuthPaths: skipPaths,
	}
}

func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for public methods and reflection
		if a.skipAuthPaths[info.FullMethod] || strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.publicKey, nil
		})

		if err != nil || !token.Valid {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
		}

		userID, ok := claims["sub"].(float64)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "invalid user id in token")
		}

		newMD := metadata.New(map[string]string{
			"x-user-id":         fmt.Sprintf("%.0f", userID),
			"x-internal-api-key": a.internalKey,
		})

		newCtx := metadata.NewOutgoingContext(ctx, newMD)
		newCtx = metadata.NewIncomingContext(newCtx, newMD)

		return handler(newCtx, req)
	}
}

func (a *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Skip auth for public methods and reflection
		if a.skipAuthPaths[info.FullMethod] || strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.publicKey, nil
		})

		if err != nil || !token.Valid {
			return status.Errorf(codes.Unauthenticated, "invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "invalid token claims")
		}

		userID, ok := claims["sub"].(float64)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "invalid user id in token")
		}

		newMD := metadata.New(map[string]string{
			"x-user-id":         fmt.Sprintf("%.0f", userID),
			"x-internal-api-key": a.internalKey,
		})

		newCtx := metadata.NewOutgoingContext(ctx, newMD)
		newCtx = metadata.NewIncomingContext(newCtx, newMD)

		return handler(srv, &streamWrapper{ServerStream: ss, ctx: newCtx})
	}
}

type streamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *streamWrapper) Context() context.Context {
	return w.ctx
}
