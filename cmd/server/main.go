package main

import (
	"flag"
	"net"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	address := flag.String("address", "0.0.0.0:8000", "server listening address")
	flag.Parse()

	logger := log.WithFields(log.Fields{"func": "main", "address": *address})

	// load certificates
	caCertPath := "certs/ca/cert.pem"
	serverCertPath := "certs/server/cert.pem"
	serverKeyPath := "certs/server/key.pem"
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
