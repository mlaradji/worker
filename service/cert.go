package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"

	"google.golang.org/grpc/credentials"
)

// LoadTLSCertificate loads client certificate and CA.
func LoadTLSCertificate(caCertPath string, clientCertPath string, clientKeyPath string) (tls.Certificate, *x509.CertPool, error) {
	// load CA certificate
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	// add CA to list of accepted certificates
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return tls.Certificate{}, nil, errors.New("failed to append CA to certificate pool")
	}

	// load server's key pair
	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	return cert, certPool, nil
}

// MakeClientTLSCredentials generates client-side TLS configuration.
func MakeClientTLSCredentials(cert tls.Certificate, certPool *x509.CertPool) credentials.TransportCredentials {
	config := &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: certPool}
	return credentials.NewTLS(config)
}

// MakeServerTLSCredentials generates server-side TLS configuration.
func MakeServerTLSCredentials(cert tls.Certificate, certPool *x509.CertPool) credentials.TransportCredentials {
	config := &tls.Config{Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: certPool, MinVersion: tls.VersionTLS13}
	return credentials.NewTLS(config)
}
