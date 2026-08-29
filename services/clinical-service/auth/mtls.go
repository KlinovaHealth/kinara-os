package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

// MTLSConfig holds paths to TLS material for a Kinara OS microservice.
type MTLSConfig struct {
	CertPath   string
	KeyPath    string
	CACertPath string
}

// BuildServerTLSConfig returns a TLS 1.3 config that requires and verifies client certificates.
func BuildServerTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}
	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                caPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		MinVersion:               tls.VersionTLS13,
		SessionTicketsDisabled:   true,
	}, nil
}

// BuildClientTLSConfig returns a TLS 1.3 config for outbound inter-service calls.
func BuildClientTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
