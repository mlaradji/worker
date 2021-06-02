package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"

	"google.golang.org/grpc/credentials"
)

var cipherSuites = []uint16{
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_AES_128_GCM_SHA256,
}

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
	config := &tls.Config{Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: certPool, CipherSuites: cipherSuites}
	return credentials.NewTLS(config)
}
