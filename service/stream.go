package service

import (
	"context"

	"google.golang.org/grpc"
)

// ServerStreamWithContext implements grpc.ServerStream, and allows one to replace the context.
type ServerStreamWithContext struct {
	grpc.ServerStream
	NewContext context.Context // NewContext
}

// Context returns NewContext, the new attached context.
func (sswc *ServerStreamWithContext) Context() context.Context {
	return sswc.NewContext
}

// NewServerStreamWithContext returns a new ServerStreamWithContext, which attaches the new context to the stream.
func NewServerStreamWithContext(ctx context.Context, ss grpc.ServerStream) *ServerStreamWithContext {
	return &ServerStreamWithContext{ServerStream: ss, NewContext: ctx}
}
