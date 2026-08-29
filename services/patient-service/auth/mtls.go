// Package auth — mTLS certificate validation for inter-service calls.
// All internal Kinara OS services communicate over mTLS; this file
// builds the server-side TLS config that enforces client certificate
// verification.
package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// MTLSConfig holds paths to TLS artefacts.
type MTLSConfig struct {
	CACertPath  string // path to the Kinara OS internal CA cert
	CertPath    string // this service's TLS certificate
	KeyPath     string // this service's TLS private key
}

// BuildServerTLSConfig returns a *tls.Config that:
//   - requires and verifies client certificates signed by the internal CA
//   - uses TLS 1.3 minimum
//   - disables session tickets (forward-secrecy enforcement)
func BuildServerTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to read CA cert %q: %w", cfg.CACertPath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("mtls: failed to parse CA cert — is it a valid PEM?")
	}

	serverCert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to load server cert/key: %w", err)
	}

	return &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caPool,
		Certificates:             []tls.Certificate{serverCert},
		MinVersion:               tls.VersionTLS13,
		SessionTicketsDisabled:   true,
		PreferServerCipherSuites: true,
	}, nil
}

// BuildClientTLSConfig returns a *tls.Config for outbound inter-service calls.
func BuildClientTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to read CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("mtls: failed to parse CA cert")
	}

	clientCert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to load client cert/key: %w", err)
	}

	return &tls.Config{
		RootCAs:                caPool,
		Certificates:           []tls.Certificate{clientCert},
		MinVersion:             tls.VersionTLS13,
		SessionTicketsDisabled: true,
	}, nil
}
