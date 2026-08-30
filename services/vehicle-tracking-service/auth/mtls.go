package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

type MTLSConfig struct {
	CertPath   string
	KeyPath    string
	CACertPath string
}

func NewTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to parse CA cert")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
