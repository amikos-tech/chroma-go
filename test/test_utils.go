//go:build basic

package test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/types"
)

func Compare(t *testing.T, actual, expected map[string]interface{}) bool {
	builtExprJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	require.Equal(t, string(expectedJSON), string(builtExprJSON))
	return true
}

func GetTestDocumentTest() ([]string, []string, []map[string]interface{}, []*types.Embedding) {
	var documents = []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	var ids = []string{
		"ID1",
		"ID2",
	}

	var metadatas = []map[string]interface{}{
		{"key1": "value1"},
		{"key2": "value2"},
	}
	var embeddings = [][]float32{
		[]float32{0.1, 0.2, 0.3},
		[]float32{0.4, 0.5, 0.6},
	}
	return documents, ids, metadatas, types.NewEmbeddingsFromFloat32(embeddings)
}

func CreateSelfSignedCert(certPath, keyPath string) {
	// Generate a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// Prepare certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Chroma, Inc."},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	// Write the certificate to file
	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %v", err)
	}
	log.Printf("Written %s", certPath)

	// Write the private key to file
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open key.pem for writing: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing key.pem: %v", err)
	}
	log.Printf("Written %s", keyPath)
}

func getMockServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "pre-flight-checks") {
			w.WriteHeader(http.StatusOK)

			_, err := w.Write([]byte(`{"max_batch_size": 41666}`))
			require.NoError(t, err)
			return
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "version") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`0.5.17`))
			require.NoError(t, err)
			return
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "tenants/default_tenant") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"name": "default_tenant"}`))
			require.NoError(t, err)
			return
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "databases/default_database") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{
									  "id": "00000000-0000-0000-0000-000000000000",
									  "name": "default_database",
									  "tenant": "default_tenant"
										}`))
			require.NoError(t, err)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "collections/test_collection") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{
						  "id": "5f9345dc-3c21-4952-bcb9-6fbbad7a5707",
						  "name": "test_collection",
						  "configuration_json": {
							"hnsw_configuration": {
							  "space": "l2",
							  "ef_construction": 100,
							  "ef_search": 10,
							  "num_threads": 14,
							  "M": 16,
							  "resize_factor": 1.2,
							  "batch_size": 100,
							  "sync_threshold": 1000,
							  "_type": "HNSWConfigurationInternal"
							},
							"_type": "CollectionConfigurationInternal"
						  },
						  "metadata": null,
						  "dimension": null,
						  "tenant": "default_tenant",
						  "database": "default_database",
						  "version": 0,
						  "log_position": 0
								}`))
			require.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"error":"InternalServerError","message": "something went wrong"}`))
		require.NoError(t, err)
	}))
	return server
}
