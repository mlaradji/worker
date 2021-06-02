package service

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/worker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// JobServer is a server wrapper around a job store.
type JobServer struct {
	pb.UnimplementedJobServiceServer
	Store *worker.JobStore
}

// NewJobServer returns a new JobServer.
func NewJobServer(store *worker.JobStore) *JobServer {
	return &JobServer{Store: store}
}

// JobStart is a unary RPC to start a new job.
func (server *JobServer) JobStart(ctx context.Context, req *pb.JobStartRequest) (*pb.JobStartResponse, error) {
	// get command name and args from request
	command, args := req.GetCommand(), req.GetArgs()

	logger := log.WithFields(log.Fields{"func": "JobStart", "command": command, "args": args})

	// get userId attached to context
	userId, err := GetUserIdFromContext(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to get userId from context")
		return nil, status.Error(codes.Internal, "unable to get userId") // internal server error since the interceptor should have set the user id in context
	}

	logger = logger.WithField("userId", userId)

	logger.Debug("received a job start request")

	job, err := server.Store.AddJob(userId, command, args)
	if err != nil {
		logger.WithError(err).Error("failed to start job")
		return nil, status.Error(codes.Internal, "failed to start job")
	}

	logger.Debug("successfully started a job")

	res := &pb.JobStartResponse{JobId: job.Key.JobId}
	return res, nil
}
