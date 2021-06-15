package main

import (
	"net"

	"github.com/docopt/docopt-go"
	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Usage is the help docs, which docopt can directly parse.
const Usage = `Usage:
	worker-server [options]

Options:
	-h --help             Show this screen.
	--debug               Set log level to DEBUG.
	--address=<addr>      Server address and port [default: 0.0.0.0:8000]
	--cert=<cert>         Path to the server certificate for mTLS. [default: certs/server/cert.pem]
	--key=<key>           Path to the server key for mTLS. [default: certs/server/key.pem]
	--ca=<ca>             Path to the CA certificate for mTLS. [default: certs/ca1/cert.pem]`

// Configuration contains all variables that were passed (implicity or explicitly) to the command.
type Configuration struct {
	// options

	Debug   bool   `docopt:"--debug"`
	Address string `docopt:"--address"`
	Cert    string `docopt:"--cert"`
	Key     string `docopt:"--key"`
	CA      string `docopt:"--ca"`
}

var (
	Config         = &Configuration{}
	TLSCredentials credentials.TransportCredentials
)

func init() {
	logger := log.WithField("func", "init")

	opts, err := docopt.ParseDoc(Usage)
	if err != nil {
		logger.WithError(err).Fatal("unable to parse usage doc")
	}

	// extract config fields
	err = opts.Bind(Config)
	if err != nil {
		logger.WithError(err).Fatal("unable to parse configuration")
	}

	// enable debug logs if --debug was passed
	if Config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	logger.WithField("Config", Config).Debug("successfully parsed configuration")

	// load certificates
	cert, certPool, err := service.LoadTLSCertificate(Config.CA, Config.Cert, Config.Key)
	if err != nil {
		logger.WithError(err).Fatal("unable to load TLS certificate")
	}

	TLSCredentials = service.MakeServerTLSCredentials(cert, certPool)

	logger.Debug("successfully loaded certificates")
}

func main() {
	logger := log.WithFields(log.Fields{"func": "main", "address": Config.Address})

	// initialize job service
	jobStore := worker.NewJobStore()
	jobServer := service.NewJobServer(jobStore)

	// initialize gRPC server with authentication and authorization interceptors
	grpcServer := grpc.NewServer(grpc.Creds(TLSCredentials), grpc.UnaryInterceptor(service.UnaryAuth), grpc.StreamInterceptor(service.StreamAuth))
	pb.RegisterJobServiceServer(grpcServer, jobServer)

	// start listening
	listener, err := net.Listen("tcp", Config.Address)
	if err != nil {
		logger.WithError(err).WithField("address", Config.Address).Fatal("unable to listen on address")
	}
	logger.Info("started listening")

	// start server
	err = grpcServer.Serve(listener)
	if err != nil {
		logger.WithError(err).Fatal("unable to serve job server on listener")
	}
}
