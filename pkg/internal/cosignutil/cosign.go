package cosignutil

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	stderrors "errors"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
)

var oidcIssuerExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

// OIDCIssuerExtensionOID returns a copy of the Sigstore Fulcio OIDC issuer certificate extension OID.
func OIDCIssuerExtensionOID() asn1.ObjectIdentifier {
	return append(asn1.ObjectIdentifier(nil), oidcIssuerExtensionOID...)
}

const (
	// FulcioIntermediatePEM is the pinned Sigstore Fulcio intermediate certificate.
	// Source: https://fulcio.sigstore.dev/api/v1/rootCert
	FulcioIntermediatePEM = `-----BEGIN CERTIFICATE-----
MIICGjCCAaGgAwIBAgIUALnViVfnU0brJasmRkHrn/UnfaQwCgYIKoZIzj0EAwMw
KjEVMBMGA1UEChMMc2lnc3RvcmUuZGV2MREwDwYDVQQDEwhzaWdzdG9yZTAeFw0y
MjA0MTMyMDA2MTVaFw0zMTEwMDUxMzU2NThaMDcxFTATBgNVBAoTDHNpZ3N0b3Jl
LmRldjEeMBwGA1UEAxMVc2lnc3RvcmUtaW50ZXJtZWRpYXRlMHYwEAYHKoZIzj0C
AQYFK4EEACIDYgAE8RVS/ysH+NOvuDZyPIZtilgUF9NlarYpAd9HP1vBBH1U5CV7
7LSS7s0ZiH4nE7Hv7ptS6LvvR/STk798LVgMzLlJ4HeIfF3tHSaexLcYpSASr1kS
0N/RgBJz/9jWCiXno3sweTAOBgNVHQ8BAf8EBAMCAQYwEwYDVR0lBAwwCgYIKwYB
BQUHAwMwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQU39Ppz1YkEZb5qNjp
KFWixi4YZD8wHwYDVR0jBBgwFoAUWMAeX5FFpWapesyQoZMi0CrFxfowCgYIKoZI
zj0EAwMDZwAwZAIwPCsQK4DYiZYDPIaDi5HFKnfxXx6ASSVmERfsynYBiX2X6SJR
nZU84/9DZdnFvvxmAjBOt6QpBlc4J/0DxvkTCqpclvziL6BCCPnjdlIB3Pu3BxsP
mygUY7Ii2zbdCdliiow=
-----END CERTIFICATE-----`

	// FulcioRootPEM is the pinned Sigstore Fulcio root certificate.
	// Source: https://fulcio.sigstore.dev/api/v1/rootCert
	FulcioRootPEM = `-----BEGIN CERTIFICATE-----
MIIB9zCCAXygAwIBAgIUALZNAPFdxHPwjeDloDwyYChAO/4wCgYIKoZIzj0EAwMw
KjEVMBMGA1UEChMMc2lnc3RvcmUuZGV2MREwDwYDVQQDEwhzaWdzdG9yZTAeFw0y
MTEwMDcxMzU2NTlaFw0zMTEwMDUxMzU2NThaMCoxFTATBgNVBAoTDHNpZ3N0b3Jl
LmRldjERMA8GA1UEAxMIc2lnc3RvcmUwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAT7
XeFT4rb3PQGwS4IajtLk3/OlnpgangaBclYpsYBr5i+4ynB07ceb3LP0OIOZdxex
X69c5iVuyJRQ+Hz05yi+UF3uBWAlHpiS5sh0+H2GHE7SXrk1EC5m1Tr19L9gg92j
YzBhMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBRY
wB5fkUWlZql6zJChkyLQKsXF+jAfBgNVHSMEGDAWgBRYwB5fkUWlZql6zJChkyLQ
KsXF+jAKBggqhkjOPQQDAwNpADBmAjEAj1nHeXZp+13NWBNa+EDsDP8G1WWg1tCM
WP/WHPqpaVo0jhsweNFZgSs0eE7wYI4qAjEA2WB9ot98sIkoF3vZYdd3/VtWB5b9
TNMea7Ix/stJ5TfcLLeABLE4BNJOsQ4vnBHJ
-----END CERTIFICATE-----`
)

// DecodeBase64Bytes decodes base64 data, trying standard then raw encoding.
func DecodeBase64Bytes(raw []byte) ([]byte, error) {
	encoded := string(bytes.Join(bytes.Fields(raw), nil))
	if encoded == "" {
		return nil, errors.New("base64 payload is empty")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}
	decoded, rawErr := base64.RawStdEncoding.DecodeString(encoded)
	if rawErr == nil {
		return decoded, nil
	}
	return nil, errors.Wrap(stderrors.Join(err, rawErr), "invalid base64 payload")
}

// ReadBase64EncodedFile reads a file and decodes its base64 content.
func ReadBase64EncodedFile(filePath string) ([]byte, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return DecodeBase64Bytes(raw)
}

// ReadCosignCertificate reads and parses a PEM or base64-encoded X.509 certificate.
func ReadCosignCertificate(filePath string) (*x509.Certificate, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read certificate file %s", filePath)
	}
	certPEM := bytes.TrimSpace(raw)
	if len(certPEM) == 0 {
		return nil, errors.New("certificate payload is empty")
	}
	if !bytes.Contains(certPEM, []byte("BEGIN CERTIFICATE")) {
		decoded, decodeErr := DecodeBase64Bytes(certPEM)
		if decodeErr != nil {
			return nil, decodeErr
		}
		certPEM = decoded
	}
	block, remainder := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}
	if !strings.EqualFold(strings.TrimSpace(block.Type), "CERTIFICATE") {
		return nil, errors.Errorf("unexpected PEM block type %q: expected CERTIFICATE", block.Type)
	}
	if len(bytes.TrimSpace(remainder)) > 0 {
		return nil, errors.New("certificate PEM contains unexpected trailing data")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse certificate from %s", filePath)
	}
	return certificate, nil
}

// CertificateExtensionValue extracts a string value from a certificate extension by OID.
func CertificateExtensionValue(certificate *x509.Certificate, oid asn1.ObjectIdentifier) (string, bool, error) {
	if certificate == nil {
		return "", false, errors.New("certificate is nil")
	}

	for _, extension := range certificate.Extensions {
		if !extension.Id.Equal(oid) {
			continue
		}
		var value string
		var unmarshalErr error
		if remainder, err := asn1.Unmarshal(extension.Value, &value); err == nil {
			if len(remainder) == 0 {
				return value, true, nil
			}
			unmarshalErr = errors.Errorf("ASN.1 payload contains %d trailing bytes", len(remainder))
		} else {
			unmarshalErr = err
		}
		if utf8.Valid(extension.Value) {
			value = strings.TrimSpace(string(extension.Value))
			if value != "" {
				return value, true, nil
			}
		}
		if unmarshalErr != nil {
			return "", false, errors.Wrapf(unmarshalErr, "failed to decode certificate extension %s", oid.String())
		}
		return "", false, errors.Errorf("failed to decode certificate extension %s", oid.String())
	}
	return "", false, nil
}

// VerifyBlobSignature verifies a signature over payload using the certificate's public key.
func VerifyBlobSignature(certificate *x509.Certificate, payload, signature []byte) error {
	if certificate == nil {
		return errors.New("certificate is nil")
	}
	if len(signature) == 0 {
		return errors.New("signature is empty")
	}

	payloadDigest := sha256.Sum256(payload)

	switch pubKey := certificate.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pubKey, payloadDigest[:], signature) {
			return errors.New("ECDSA signature verification failed")
		}
		return nil
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, payloadDigest[:], signature); err != nil {
			return errors.Wrap(err, "RSA signature verification failed")
		}
		return nil
	case ed25519.PublicKey:
		if !ed25519.Verify(pubKey, payload, signature) {
			return errors.New("Ed25519 signature verification failed")
		}
		return nil
	default:
		return errors.Errorf("unsupported certificate public key type %T", certificate.PublicKey)
	}
}

// VerifyCertificateChain verifies a leaf certificate against the given root and intermediate PEMs.
func VerifyCertificateChain(certificate *x509.Certificate, rootPEM, intermediatePEM string) error {
	if certificate == nil {
		return errors.New("certificate is nil")
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM([]byte(rootPEM)); !ok {
		return errors.New("failed to load root certificate")
	}
	intermediates := x509.NewCertPool()
	if ok := intermediates.AppendCertsFromPEM([]byte(intermediatePEM)); !ok {
		return errors.New("failed to load intermediate certificate")
	}

	// Fulcio leaf certs are short-lived; verify chain validity at issuance time.
	verifyAt := certificate.NotBefore
	if certificate.NotAfter.After(certificate.NotBefore) {
		verifyAt = certificate.NotBefore.Add(certificate.NotAfter.Sub(certificate.NotBefore) / 2)
	}

	if _, err := certificate.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		CurrentTime:   verifyAt,
	}); err != nil {
		return errors.Wrap(err, "certificate chain verification failed")
	}
	return nil
}

// VerifyFulcioCertificateChain verifies the certificate against the pinned Sigstore Fulcio trust anchors.
func VerifyFulcioCertificateChain(certificate *x509.Certificate) error {
	return VerifyCertificateChain(certificate, FulcioRootPEM, FulcioIntermediatePEM)
}

// ValidateCosignCertificate validates a Sigstore Fulcio certificate's identity, OIDC issuer, and chain.
func ValidateCosignCertificate(certificate *x509.Certificate, expectedIdentity, expectedOIDCIssuer string, chainVerifier func(*x509.Certificate) error) error {
	if certificate == nil {
		return errors.New("certificate is nil")
	}
	if strings.TrimSpace(expectedIdentity) == "" {
		return errors.New("expected certificate identity cannot be empty")
	}
	if strings.TrimSpace(expectedOIDCIssuer) == "" {
		return errors.New("expected certificate OIDC issuer cannot be empty")
	}
	if chainVerifier == nil {
		return errors.New("certificate chain verifier function is nil")
	}
	// Sigstore Fulcio certificates are intentionally short-lived (typically ~10 minutes).
	// Downloads often happen after certificate expiry, so we only reject not-yet-valid certs
	// and rely on signature + identity + OIDC issuer validation here.
	now := time.Now()
	if now.Before(certificate.NotBefore) {
		return errors.Errorf(
			"certificate is not yet valid: valid starting at %s",
			certificate.NotBefore.UTC().Format(time.RFC3339),
		)
	}
	if err := chainVerifier(certificate); err != nil {
		return err
	}

	hasCodeSigningUsage := false
	for _, usage := range certificate.ExtKeyUsage {
		if usage == x509.ExtKeyUsageCodeSigning {
			hasCodeSigningUsage = true
			break
		}
	}
	if !hasCodeSigningUsage {
		return errors.New("certificate is missing code signing extended key usage")
	}

	hasExpectedIdentity := false
	for _, uri := range certificate.URIs {
		if uri != nil && uri.String() == expectedIdentity {
			hasExpectedIdentity = true
			break
		}
	}
	if !hasExpectedIdentity {
		return errors.Errorf("certificate identity does not match expected release identity %s", expectedIdentity)
	}

	issuerOID := OIDCIssuerExtensionOID()
	issuerValue, foundIssuer, err := CertificateExtensionValue(certificate, issuerOID)
	if err != nil {
		return err
	}
	if !foundIssuer {
		return errors.Errorf("certificate missing OIDC issuer extension %s", issuerOID.String())
	}
	if issuerValue != expectedOIDCIssuer {
		return errors.Errorf("certificate OIDC issuer mismatch: expected %s, got %s", expectedOIDCIssuer, issuerValue)
	}
	return nil
}

// VerifySignedChecksums verifies a cosign-signed checksums file.
func VerifySignedChecksums(checksumsPath, signaturePath, certificatePath, expectedIdentity, expectedOIDCIssuer string, chainVerifier func(*x509.Certificate) error) error {
	if strings.TrimSpace(expectedIdentity) == "" || strings.TrimSpace(expectedOIDCIssuer) == "" {
		return errors.New("expectedIdentity and expectedOIDCIssuer must be non-empty")
	}

	checksumsBytes, err := os.ReadFile(checksumsPath)
	if err != nil {
		return errors.Wrap(err, "failed to read checksum file")
	}

	signature, err := ReadBase64EncodedFile(signaturePath)
	if err != nil {
		return errors.Wrap(err, "failed to decode checksum signature")
	}

	certificate, err := ReadCosignCertificate(certificatePath)
	if err != nil {
		return errors.Wrap(err, "failed to parse checksum certificate")
	}

	if err := ValidateCosignCertificate(certificate, expectedIdentity, expectedOIDCIssuer, chainVerifier); err != nil {
		return errors.Wrap(err, "certificate validation failed")
	}

	if err := VerifyBlobSignature(certificate, checksumsBytes, signature); err != nil {
		return errors.Wrap(err, "invalid checksum signature")
	}
	return nil
}
