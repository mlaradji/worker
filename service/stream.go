package service

import (
	"context"

	"google.golang.org/grpc"
)

// ServerStreamWithContext implements grpc.ServerStream, and allows one to replace the context.
type ServerStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the attached context.
func (sswc *ServerStreamWithContext) Context() context.Context {
	return sswc.ctx
}

// NewServerStreamWithContext returns a new ServerStreamWithContext, which attaches the passed context to the stream.
func NewServerStreamWithContext(ctx context.Context, ss grpc.ServerStream) *ServerStreamWithContext {
	return &ServerStreamWithContext{ServerStream: ss, ctx: ctx}
}
