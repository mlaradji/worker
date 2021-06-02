package service

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/worker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// JobStop is a unary RPC to stop an existing job.
func (server *JobServer) JobStop(ctx context.Context, req *pb.JobStopRequest) (*pb.JobStopResponse, error) {
	// get command name and args from request
	jobId := req.GetJobId()

	logger := log.WithFields(log.Fields{"func": "JobStop", "jobId": jobId})

	// get userId attached to context
	userId, err := GetUserIdFromContext(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to get userId from context")
		return nil, status.Error(codes.Internal, "unable to get userId") // internal server error since the interceptor should have set the user id in context
	}

	logger = logger.WithField("userId", userId)

	logger.Debug("received a job stop request")

	job, err := server.Store.LoadJob(worker.JobKey{UserId: userId, JobId: jobId})
	if err != nil {
		if errors.Is(err, worker.ErrJobDoesNotExist) {
			logger.Debug("job was not found")
			return nil, status.Error(codes.NotFound, "job was not found")
		}

		logger.WithError(err).Error("job is invalid")
		return nil, status.Error(codes.Internal, "job is invalid")
	}

	job.Stop()
	logger.Debug("sent job stop request")

	return &pb.JobStopResponse{}, nil
}

// JobStatus is a unary RPC to query for job status.
func (server *JobServer) JobStatus(ctx context.Context, req *pb.JobStatusRequest) (*pb.JobStatusResponse, error) {
	// get command name and args from request
	jobId := req.GetJobId()

	logger := log.WithFields(log.Fields{"func": "JobStatus", "jobId": jobId})

	// get userId attached to context
	userId, err := GetUserIdFromContext(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to get userId from context")
		return nil, status.Error(codes.Internal, "unable to get userId") // internal server error since the interceptor should have set the user id in context
	}

	logger = logger.WithField("userId", userId)

	logger.Debug("received a job status query request")

	job, err := server.Store.LoadJob(worker.JobKey{UserId: userId, JobId: jobId})
	if err != nil {
		if errors.Is(err, worker.ErrJobDoesNotExist) {
			logger.Debug("job was not found")
			return nil, status.Error(codes.NotFound, "job was not found")
		}

		logger.WithError(err).Error("job is invalid")
		return nil, status.Error(codes.Internal, "job is invalid")
	}

	jobStatus := &pb.JobStatusResponse{
		JobInfo: &pb.JobInfo{
			Id:         job.Key.JobId,
			UserId:     job.Key.UserId,
			Command:    job.Command,
			Args:       job.Args,
			JobStatus:  job.JobStatus,
			ExitCode:   job.ExitCode,
			CreatedAt:  timestamppb.New(job.CreatedAt),
			FinishedAt: timestamppb.New(job.FinishedAt),
		},
	}

	return jobStatus, nil
}

// JobLog is a server-side streaming RPC to follow a job's log until the job is done.
func (server *JobServer) JobLogsStream(req *pb.JobLogsRequest, stream pb.JobService_JobLogsStreamServer) error {
	// get command name and args from request
	jobId := req.GetJobId()

	logger := log.WithFields(log.Fields{"func": "JobLogsStream", "jobId": jobId})

	// get userId attached to context
	userId, err := GetUserIdFromContext(stream.Context())
	if err != nil {
		logger.WithError(err).Error("unable to get userId from context")
		return status.Error(codes.Internal, "unable to get userId") // internal server error since the interceptor should have set the user id in context
	}

	logger = logger.WithField("userId", userId)

	logger.Debug("received a job log follow request")

	job, err := server.Store.LoadJob(worker.JobKey{UserId: userId, JobId: jobId})
	if err != nil {
		if errors.Is(err, worker.ErrJobDoesNotExist) {
			logger.Debug("job was not found")
			return status.Error(codes.NotFound, "job was not found")
		}

		logger.WithError(err).Error("job is invalid")
		return status.Error(codes.Internal, "job is invalid")
	}

	logChannel, err := job.Log()
	if err != nil {
		logger.WithError(err).Error("unable to follow logs")
		return status.Error(codes.Internal, "server unable to follow job logs")
	}

	for logChunk := range logChannel {
		res := &pb.JobLogsResponse{Log: logChunk}
		err := stream.Send(res)
		if err != nil {
			logger.WithError(err).Error("unable to send log chunk")
			return status.Errorf(codes.Internal, "unable to send log chunk")
		}
	}

	return nil
}
