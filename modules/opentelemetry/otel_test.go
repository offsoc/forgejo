// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package opentelemetry

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNoopDefault(t *testing.T) {
	inMem := tracetest.NewInMemoryExporter()
	called := false
	exp := func(ctx context.Context) (sdktrace.SpanExporter, error) {
		called = true
		return inMem, nil
	}
	exporter["inmemory"] = exp
	t.Cleanup(func() {
		delete(exporter, "inmemory")
	})
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces, "inmemory")

	ctx := context.Background()
	require.NoError(t, Init(ctx))
	tracer := otel.Tracer("test_noop")

	_, span := tracer.Start(ctx, "test span")
	defer span.End()

	assert.False(t, span.SpanContext().HasTraceID())
	assert.False(t, span.SpanContext().HasSpanID())
	assert.False(t, called)
}

func generateTestTLS(t *testing.T, path, host string) *tls.Config {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "Failed to generate private key: %v", err)

	keyUsage := x509.KeyUsageDigitalSignature

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoError(t, err, "Failed to generate serial number: %v", err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Forgejo Testing"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	require.NoError(t, err, "Failed to create certificate: %v", err)

	certOut, err := os.Create(path + "/cert.pem")
	require.NoError(t, err, "Failed to open cert.pem for writing: %v", err)

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	if err := certOut.Close(); err != nil {
		t.Fatalf("Error closing cert.pem: %v", err)
	}
	keyOut, err := os.OpenFile(path+"/key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	require.NoError(t, err, "Failed to open key.pem for writing: %v", err)

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err, "Unable to marshal private key: %v", err)

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		t.Fatalf("Error closing key.pem: %v", err)
	}
	serverCert, err := tls.LoadX509KeyPair(path+"/cert.pem", path+"/key.pem")
	require.NoError(t, err, "failed to load the key pair")
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAnyClientCert,
	}
}
