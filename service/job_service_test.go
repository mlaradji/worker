// Inspired by https://stackoverflow.com/a/52080545/9954163.
package service_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufferSize = 1024
	echoLoop   = `#!/bin/sh

for i in {1..10}
do
  echo "Command no. $i"
  sleep 0.2
done`
)

var (
	listener *bufconn.Listener
)

func init() {
	log.SetLevel(log.DebugLevel)

	logger := log.WithFields(log.Fields{"func": "init"})

	caCertPath := "../certs/ca/cert.pem"
	serverCertPath := "../certs/server/cert.pem"
	serverKeyPath := "../certs/server/key.pem"

	cert, certPool, err := service.LoadTLSCertificate(caCertPath, serverCertPath, serverKeyPath)
	if err != nil {
		logger.WithError(err).Fatal("cannot load TLS certificate")
	}

	tlsCredentials := service.MakeServerTLSCredentials(cert, certPool)

	// initialize job service
	jobStore := worker.NewJobStore()
	jobServer := service.NewJobServer(jobStore)

	// initialize gRPC server with authentication and authorization interceptors
	grpcServer := grpc.NewServer(grpc.Creds(tlsCredentials), grpc.UnaryInterceptor(service.UnaryAuth), grpc.StreamInterceptor(service.StreamAuth))
	pb.RegisterJobServiceServer(grpcServer, jobServer)

	// start listening
	listener = bufconn.Listen(bufferSize)

	// start server
	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			logger.WithError(err).Fatal("server failed to start")
		}
	}()
}

func dialListener(context.Context, string) (net.Conn, error) {
	return listener.Dial()
}

func createConnection(ctx context.Context, caCertPath string, clientCertPath string, clientKeyPath string) (*grpc.ClientConn, error) {
	// load client certificate
	cert, certPool, err := service.LoadTLSCertificate(caCertPath, clientCertPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	tlsCredentials := service.MakeClientTLSCredentials(cert, certPool)

	// connect to server

	dialOption := grpc.WithContextDialer(dialListener)
	target := "bufnet"
	return grpc.DialContext(ctx, target, dialOption, grpc.WithTransportCredentials(tlsCredentials))
}

// TestJobFlow starts a long running process, queries its status, listens to its logs, and stops it. A second client will attempt to query information about it, which should not be allowed.
func TestJobFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := createConnection(ctx, "../certs/ca/cert.pem", "../certs/client1/cert.pem", "../certs/client1/key.pem")
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewJobServiceClient(conn)

	// start a long running job for which we know the output
	command := "sh"
	args := []string{"-c", echoLoop}
	startRes, err := client.JobStart(ctx, &pb.JobStartRequest{Command: command, Args: args})
	require.NoError(t, err)

	// check that job was successfully started
	jobId := startRes.GetJobId()
	statusRes, err := client.JobStatus(ctx, &pb.JobStatusRequest{JobId: jobId})
	require.NoError(t, err)

	jobInfo := statusRes.GetJobInfo()
	require.Equal(t, command, jobInfo.Command)
	require.Equal(t, args, jobInfo.Args)
	require.Contains(t, []int32{-1, 0}, jobInfo.ExitCode)
	require.Equal(t, jobId, jobInfo.Id)
	require.Equal(t, "client1", jobInfo.UserId)                                                          // TODO: remove hard-coded userId
	require.Contains(t, []pb.JobStatus{pb.JobStatus_RUNNING, pb.JobStatus_SUCCEEDED}, jobInfo.JobStatus) // process might have finished

	// check that log output is correct. We should get 10 messages.
	expectedOutput := []byte{}
	for i := 1; i < 11; i++ {
		expectedOutput = append(expectedOutput, []byte(fmt.Sprintf("Command no. %d\n", i))...) // echo will emit an extra newline char
	}
	actualOutput := []byte{}
	logStream, err := client.JobLogsStream(ctx, &pb.JobLogsRequest{JobId: jobId})
	require.NoError(t, err)

	for {
		logRes, err := logStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		actualOutput = append(actualOutput, logRes.GetLog()...)
	}

	require.Equal(t, expectedOutput, actualOutput)

	// check that the process is done
	successRes, err := client.JobStatus(ctx, &pb.JobStatusRequest{JobId: jobId})
	require.NoError(t, err)

	successJobInfo := successRes.GetJobInfo()
	require.Equal(t, int32(0), successJobInfo.ExitCode)
	require.Equal(t, pb.JobStatus_SUCCEEDED, successJobInfo.JobStatus)

	// try to modify or query the job from another client
	ctx = context.Background()
	conn2, err := createConnection(ctx, "../certs/ca/cert.pem", "../certs/client2/cert.pem", "../certs/client2/key.pem")
	require.NoError(t, err)
	defer conn.Close()

	client2 := pb.NewJobServiceClient(conn2)

	_, err = client2.JobStatus(ctx, &pb.JobStatusRequest{JobId: jobId})
	errStatus, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, errStatus.Code())

	_, err = client2.JobStop(ctx, &pb.JobStopRequest{JobId: jobId})
	errStatus, ok = status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, errStatus.Code())

	_, err = client2.JobLogsStream(ctx, &pb.JobLogsRequest{JobId: jobId})
	require.Error(t, err, "err", err)
	log.Debug(err)
	errStatus, ok = status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, errStatus.Code(), "details", errStatus.Details())
}
