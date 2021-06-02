package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"
	"net"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var (
	cipherSuites = []uint16{
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
	}

	clientUserMap = map[service.ClientId]string{
		{Issuer: "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=CA/CN=CA/emailAddress=mlaradji@pm.me", Subject: "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 1/emailAddress=mlaradji@pm.me"}: "client1",
		{Issuer: "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=CA/CN=CA/emailAddress=mlaradji@pm.me", Subject: "/C=CA/ST=British Columbia/L=Vancouver/O=Mohamed, Inc./OU=Client/CN=Client 2/emailAddress=mlaradji@pm.me"}: "client2",
	} // clientUserMap maps certificate Issuer+Subject to userId. Clients not in this map will not be allowed access.
)

func main() {
	address := flag.String("address", "0.0.0.0:8000", "server listening address")
	flag.Parse()

	logger := log.WithFields(log.Fields{"func": "main", "address": *address})

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		logger.WithError(err).Fatal("cannot load TLS credentials")
	}

	// initialize job service
	jobStore := worker.NewJobStore()
	jobServer := service.NewJobServer(jobStore)

	// initialize gRPC server with authentication and authorization interceptors
	grpcServer := grpc.NewServer(grpc.Creds(tlsCredentials), grpc.UnaryInterceptor(unaryAuth), grpc.StreamInterceptor(streamAuth))
	pb.RegisterJobServiceServer(grpcServer, jobServer)

	// start listening
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		logger.WithError(err).Fatal("unable to listen on address")
	}
	logger.Info("started listener")

	// start server
	err = grpcServer.Serve(listener)
	if err != nil {
		logger.WithError(err).Fatal("server failed to start")
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// load CA certificate
	caCert, err := ioutil.ReadFile("certs/ca/cert.pem")
	if err != nil {
		return nil, err
	}

	// add CA to list of accepted certificates
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, errors.New("failed to append CA to certificate pool")
	}

	// load server's key pair
	cert, err := tls.LoadX509KeyPair("certs/server/cert.pem", "certs/server/key.pem")
	if err != nil {
		return nil, err
	}

	// construct and return transport credentials
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: certPool, CipherSuites: cipherSuites}
	return credentials.NewTLS(tlsConfig), nil
}

func unaryAuth(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	userId, err := authenticateAndAuthorize(ctx)
	if err != nil {
		return nil, err
	}

	// attach userId to context
	ctxUser := service.SetUserIdInContext(ctx, userId)

	return handler(ctxUser, req)
}

func streamAuth(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	userId, err := authenticateAndAuthorize(ss.Context())
	if err != nil {
		return err
	}

	// attach userId to context
	ctxUser := service.SetUserIdInContext(ss.Context(), userId)
	sswc := service.NewServerStreamWithContext(ctxUser, ss)

	return handler(srv, sswc)
}

// authenticateAndAuthorize gets the client ID from the context, checks it against the global user map, and returns the userId.
func authenticateAndAuthorize(ctx context.Context) (string, error) {
	logger := log.WithFields(log.Fields{"func": "authenticateAndAuthorize"})

	// client authentication
	clientId, err := service.ClientIdFromContext(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to get client id from context")
		return "", status.Error(codes.Unauthenticated, "unable to determine client id")
	}

	// client authorization
	userId, ok := clientUserMap[clientId]
	if !ok {
		logger.WithField("clientId", clientId).Debug("client is not authorized to access resource")
		return "", status.Error(codes.PermissionDenied, "client is not authorized to access resource")
	}

	return userId, nil
}
