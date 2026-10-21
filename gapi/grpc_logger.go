package gapi

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// GrpcLogger logs the RPC method and status without logging credentials.
func GrpcLogger(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	response, err := handler(ctx, req)
	log.Info().Str("method", info.FullMethod).
		Str("code", status.Code(err).String()).
		Dur("duration", time.Since(start)).Msg("gRPC request")
	return response, err
}
