package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/worker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	address := flag.String("address", "0.0.0.0:8000", "server listening address")
	flag.Parse()

	logger := log.WithFields(log.Fields{"func": "main", "address": *address})

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	jobStore := worker.NewJobStore()
	grpcServer := grpc.NewServer(grpc.Creds(tlsCredentials), grpc.UnaryInterceptor(unaryInterceptor), grpc.StreamInterceptor(streamInterceptor))
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	_, err := net.Listen("tcp", *address)
	if err != nil {
		logger.WithError(err).Fatal("unable to listen on address")
	}
	logger.Info("server started")
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load CA certificate
	caCert, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, fmt.Errorf("failed to append CA to certificate pool")
	}

	// load server's key pair
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the transport credentials and return it
	config := &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: certPool, CipherSuites: []uint16{
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
	}}

	return credentials.NewTLS(config), nil
}
