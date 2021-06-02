// Inspired by https://stackoverflow.com/a/52080545/9954163.
package service_test

import (
	"context"
	"net"
	"testing"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufferSize = 1024

var (
	listener *bufconn.Listener
)

func init() {
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

func TestJobStart(t *testing.T) {
	t.Parallel()

	caCertPath := "../certs/ca/cert.pem"
	clientCertPath := "../certs/client1/cert.pem"
	clientKeyPath := "../certs/client1/key.pem"

	// load client certificate
	cert, certPool, err := service.LoadTLSCertificate(caCertPath, clientCertPath, clientKeyPath)
	require.Nil(t, err)
	tlsCredentials := service.MakeClientTLSCredentials(cert, certPool)

	// connect to server
	ctx := context.Background()
	dialOption := grpc.WithContextDialer(dialListener)
	target := "bufnet"
	conn, err := grpc.DialContext(ctx, target, dialOption, grpc.WithTransportCredentials(tlsCredentials))
	require.Nil(t, err)
	defer conn.Close()

	client := pb.NewJobServiceClient(conn)

	_, err = client.JobStart(ctx, &pb.JobStartRequest{Command: "echo", Args: []string{"hi"}})
	require.Nil(t, err, err)

}
