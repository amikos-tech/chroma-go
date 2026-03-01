package cosignutil

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateCosignCertificate_RejectsNilChainVerifier(t *testing.T) {
	identity := "https://github.com/amikos-tech/pure-tokenizers/.github/workflows/rust-release.yml@refs/tags/rust-v0.1.4"
	issuer := "https://token.actions.githubusercontent.com"
	certificate := newTestCosignCertificate(t, identity, issuer)

	err := ValidateCosignCertificate(certificate, identity, issuer, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "certificate chain verifier function is nil")
}

func TestVerifySignedChecksums_RejectsEmptyIdentityOrIssuer(t *testing.T) {
	tests := []struct {
		name               string
		expectedIdentity   string
		expectedOIDCIssuer string
	}{
		{name: "empty identity", expectedIdentity: "", expectedOIDCIssuer: "https://token.actions.githubusercontent.com"},
		{name: "empty issuer", expectedIdentity: "https://example.com/workflow@refs/tags/v1.2.3", expectedOIDCIssuer: ""},
		{name: "both empty", expectedIdentity: "", expectedOIDCIssuer: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifySignedChecksums(
				"/tmp/does-not-matter-checksums",
				"/tmp/does-not-matter-signature",
				"/tmp/does-not-matter-certificate",
				tc.expectedIdentity,
				tc.expectedOIDCIssuer,
				VerifyFulcioCertificateChain,
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expectedIdentity and expectedOIDCIssuer must be non-empty")
		})
	}
}

func TestReadCosignCertificate_RejectsNonCertificatePEMType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cert-request.pem")
	wrongTypePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: []byte("not-a-certificate"),
	})
	require.NoError(t, os.WriteFile(path, wrongTypePEM, 0600))

	_, err := ReadCosignCertificate(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected PEM block type")
}

func TestVerifyCertificateChain_UsesGenericLoadErrors(t *testing.T) {
	certificate := &x509.Certificate{}

	err := VerifyCertificateChain(certificate, "definitely-not-a-pem", FulcioIntermediatePEM)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load root certificate")

	err = VerifyCertificateChain(certificate, FulcioRootPEM, "definitely-not-a-pem")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load intermediate certificate")
}

func newTestCosignCertificate(t *testing.T, identity, issuer string) *x509.Certificate {
	t.Helper()

	identityURL, err := url.Parse(identity)
	require.NoError(t, err)
	issuerValue, err := asn1.Marshal(issuer)
	require.NoError(t, err)

	return &x509.Certificate{
		NotBefore:   time.Now().Add(-1 * time.Minute),
		NotAfter:    time.Now().Add(10 * time.Minute),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		URIs:        []*url.URL{identityURL},
		Extensions: []pkix.Extension{
			{
				Id:    OIDCIssuerExtensionOID(),
				Value: issuerValue,
			},
		},
	}
}
