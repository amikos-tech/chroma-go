package tokenizers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/internal/cosignutil"
)

func TestNormalizeTokenizerTag(t *testing.T) {
	tooLongSemver := "v1.2.3-" + strings.Repeat("a", tokenizerMaxVersionTagLength)

	tests := []struct {
		name      string
		in        string
		expected  string
		expectErr bool
	}{
		{name: "empty", in: "", expectErr: true},
		{name: "latest", in: "latest", expected: "latest"},
		{name: "latest case-insensitive", in: "LATEST", expected: "latest"},
		{name: "semver", in: "0.1.4", expected: "rust-v0.1.4"},
		{name: "go tag", in: "v0.1.4", expected: "rust-v0.1.4"},
		{name: "rust prefix", in: "rust-v0.1.4", expected: "rust-v0.1.4"},
		{name: "bare rust prefix", in: "rust-0.1.4", expected: "rust-v0.1.4"},
		{name: "prerelease", in: "v0.1.4-rc.1", expected: "rust-v0.1.4-rc.1"},
		{name: "build metadata", in: "rust-v0.1.4+build.7", expected: "rust-v0.1.4+build.7"},
		{name: "empty rust suffix", in: "rust-", expectErr: true},
		{name: "non-digit rust suffix", in: "rust-abc", expectErr: true},
		{name: "empty go v suffix", in: "v", expectErr: true},
		{name: "partial go v suffix", in: "v1", expectErr: true},
		{name: "partial semver", in: "1.2", expectErr: true},
		{name: "default non-digit", in: "abc", expectErr: true},
		{name: "invalid chars", in: "v0.1.4/../../", expectErr: true},
		{name: "too long", in: tooLongSemver, expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTokenizerTag(tc.in)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestTokenizerReleaseBaseURLsDeduplicates(t *testing.T) {
	lockTokenizerTestHooks(t)

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
	})

	tokenizerReleaseBaseURL = "https://releases.amikos.tech/pure-tokenizers/"
	tokenizerFallbackReleaseBaseURL = "https://releases.amikos.tech/pure-tokenizers"

	require.Equal(t, []string{"https://releases.amikos.tech/pure-tokenizers"}, tokenizerReleaseBaseURLs())
}

func TestTokenizerReleaseBaseURLsRejectsHTTP(t *testing.T) {
	lockTokenizerTestHooks(t)

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
	})

	tokenizerReleaseBaseURL = "http://insecure.example.com/pure-tokenizers"
	tokenizerFallbackReleaseBaseURL = "https://releases.amikos.tech/pure-tokenizers"

	urls := tokenizerReleaseBaseURLs()
	require.Equal(t, []string{"https://releases.amikos.tech/pure-tokenizers"}, urls)
}

func TestTokenizerChecksumFromSumsFile(t *testing.T) {
	t.Run("find checksum", func(t *testing.T) {
		dir := t.TempDir()
		sumsPath := filepath.Join(dir, "SHA256SUMS")
		contents := "1111111111111111111111111111111111111111111111111111111111111111  libtokenizers-x86_64-apple-darwin.tar.gz\n"
		require.NoError(t, os.WriteFile(sumsPath, []byte(contents), 0600))

		checksum, err := tokenizerChecksumFromSumsFile(sumsPath, "libtokenizers-x86_64-apple-darwin.tar.gz")
		require.NoError(t, err)
		require.Equal(t, "1111111111111111111111111111111111111111111111111111111111111111", checksum)
	})

	t.Run("missing checksum", func(t *testing.T) {
		dir := t.TempDir()
		sumsPath := filepath.Join(dir, "SHA256SUMS")
		contents := "2222222222222222222222222222222222222222222222222222222222222222  other.tar.gz\n"
		require.NoError(t, os.WriteFile(sumsPath, []byte(contents), 0600))

		_, err := tokenizerChecksumFromSumsFile(sumsPath, "libtokenizers-x86_64-apple-darwin.tar.gz")
		require.Error(t, err)
		require.Contains(t, err.Error(), "checksum entry not found")
	})

	t.Run("invalid checksum format", func(t *testing.T) {
		dir := t.TempDir()
		sumsPath := filepath.Join(dir, "SHA256SUMS")
		contents := "not-a-sha256  libtokenizers-x86_64-apple-darwin.tar.gz\n"
		require.NoError(t, os.WriteFile(sumsPath, []byte(contents), 0600))

		_, err := tokenizerChecksumFromSumsFile(sumsPath, "libtokenizers-x86_64-apple-darwin.tar.gz")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid checksum format")
	})
}

func TestTokenizerDecodeJSONResponseSizeLimit(t *testing.T) {
	payload := `{"version":"rust-v0.1.4"}`
	var out tokenizerLatestRelease
	require.NoError(t, tokenizerDecodeJSONResponse(strings.NewReader(payload), int64(len(payload)), &out))
	require.Equal(t, "rust-v0.1.4", out.Version)

	err := tokenizerDecodeJSONResponse(strings.NewReader(payload), 8, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata response is too large")
}

func TestTokenizerGetMetadataHTTPClient_TransportHardening(t *testing.T) {
	client := tokenizerGetMetadataHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "metadata client transport must be *http.Transport")
	require.Equal(t, 90*time.Second, transport.IdleConnTimeout)
	require.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
	require.Equal(t, 30*time.Second, transport.ResponseHeaderTimeout)
}

func TestTokenizerFetchLatestVersionFromBase_UsesInjectableClient(t *testing.T) {
	lockTokenizerTestHooks(t)

	originalClientFunc := tokenizerGetMetadataHTTPClientFunc
	t.Cleanup(func() {
		tokenizerGetMetadataHTTPClientFunc = originalClientFunc
	})

	var requestedURL string
	var requestedAccept string
	var requestedUserAgent string
	tokenizerGetMetadataHTTPClientFunc = func() *http.Client {
		return &http.Client{
			Transport: tokenizerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requestedURL = req.URL.String()
				requestedAccept = req.Header.Get("Accept")
				requestedUserAgent = req.Header.Get("User-Agent")
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"version":"rust-v9.9.9"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}
	}

	version, err := tokenizerFetchLatestVersionFromBase("https://releases.amikos.tech/pure-tokenizers")
	require.NoError(t, err)
	require.Equal(t, "rust-v9.9.9", version)
	require.Equal(t, "https://releases.amikos.tech/pure-tokenizers/latest.json", requestedURL)
	require.Equal(t, "application/json", requestedAccept)
	require.Equal(t, "chroma-go-tokenizers-downloader", requestedUserAgent)
}

func TestTokenizerResolveLatestVersion_FallsBackToGitHub(t *testing.T) {
	lockTokenizerTestHooks(t)

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	originalClientFunc := tokenizerGetMetadataHTTPClientFunc
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
		tokenizerGetMetadataHTTPClientFunc = originalClientFunc
	})

	tokenizerReleaseBaseURL = "https://mirror-a.invalid/pure-tokenizers"
	tokenizerFallbackReleaseBaseURL = "https://mirror-b.invalid/pure-tokenizers"

	var latestMetadataRequests int
	var githubRequests int
	tokenizerGetMetadataHTTPClientFunc = func() *http.Client {
		return &http.Client{
			Transport: tokenizerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case strings.HasSuffix(req.URL.Path, "/latest.json"):
					latestMetadataRequests++
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Status:     "500 Internal Server Error",
						Body:       io.NopCloser(strings.NewReader(`{"error":"metadata unavailable"}`)),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				case req.URL.String() == tokenizerGitHubReleasesAPI:
					githubRequests++
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(`[{"tag_name":"rust-v9.9.9"}]`)),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				default:
					return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
				}
			}),
		}
	}

	version, err := tokenizerResolveLatestVersion()
	require.NoError(t, err)
	require.Equal(t, "rust-v9.9.9", version)
	require.Equal(t, 2, latestMetadataRequests)
	require.Equal(t, 1, githubRequests)
}

func TestTokenizerVerifySignedChecksums(t *testing.T) {
	lockTokenizerTestHooks(t)

	bypassTokenizerCosignChainVerification(t)

	version := "rust-v0.1.4"
	checksumBody := []byte("1111111111111111111111111111111111111111111111111111111111111111  artifact.tar.gz\n")
	signatureBody, certificateBody := newTokenizerSignedChecksumArtifacts(t, version, checksumBody)

	dir := t.TempDir()
	checksumPath := filepath.Join(dir, tokenizerChecksumsAsset)
	signaturePath := filepath.Join(dir, tokenizerChecksumsSignatureAsset)
	certificatePath := filepath.Join(dir, tokenizerChecksumsCertificateAsset)
	require.NoError(t, os.WriteFile(checksumPath, checksumBody, 0600))
	require.NoError(t, os.WriteFile(signaturePath, signatureBody, 0600))
	require.NoError(t, os.WriteFile(certificatePath, certificateBody, 0600))

	require.NoError(t, tokenizerVerifySignedChecksums(version, checksumPath, signaturePath, certificatePath))
}

func TestTokenizerVerifySignedChecksums_RejectsInvalidSignature(t *testing.T) {
	lockTokenizerTestHooks(t)

	bypassTokenizerCosignChainVerification(t)

	version := "rust-v0.1.4"
	checksumBody := []byte("1111111111111111111111111111111111111111111111111111111111111111  artifact.tar.gz\n")
	_, certificateBody := newTokenizerSignedChecksumArtifacts(t, version, checksumBody)
	invalidSignature := []byte(base64.StdEncoding.EncodeToString([]byte("definitely-invalid-signature")))

	dir := t.TempDir()
	checksumPath := filepath.Join(dir, tokenizerChecksumsAsset)
	signaturePath := filepath.Join(dir, tokenizerChecksumsSignatureAsset)
	certificatePath := filepath.Join(dir, tokenizerChecksumsCertificateAsset)
	require.NoError(t, os.WriteFile(checksumPath, checksumBody, 0600))
	require.NoError(t, os.WriteFile(signaturePath, invalidSignature, 0600))
	require.NoError(t, os.WriteFile(certificatePath, certificateBody, 0600))

	err := tokenizerVerifySignedChecksums(version, checksumPath, signaturePath, certificatePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid checksum signature")
}

func TestEnsureTokenizerLibraryDownloadedRetriesAcrossMirrors(t *testing.T) {
	lockTokenizerTestHooks(t)

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	originalHomeDir := tokenizerUserHomeDirFunc
	originalArtifactDownloadFunc := tokenizerDownloadArtifactFileFunc
	originalMetadataDownloadFunc := tokenizerDownloadMetadataFileFunc
	originalBuildInfo := tokenizerReadBuildInfoFunc
	originalVerifyChainFunc := tokenizerVerifyCosignCertificateChainFunc
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
		tokenizerUserHomeDirFunc = originalHomeDir
		tokenizerDownloadArtifactFileFunc = originalArtifactDownloadFunc
		tokenizerDownloadMetadataFileFunc = originalMetadataDownloadFunc
		tokenizerReadBuildInfoFunc = originalBuildInfo
		tokenizerVerifyCosignCertificateChainFunc = originalVerifyChainFunc
	})

	tempHome := t.TempDir()
	tokenizerUserHomeDirFunc = func() (string, error) { return tempHome, nil }
	tokenizerReadBuildInfoFunc = func() (*debug.BuildInfo, bool) { return nil, false }
	tokenizerVerifyCosignCertificateChainFunc = func(*x509.Certificate) error { return nil }
	tokenizerReleaseBaseURL = "https://mirror-a.invalid/pure-tokenizers"
	tokenizerFallbackReleaseBaseURL = "https://mirror-b.invalid/pure-tokenizers"
	t.Setenv("TOKENIZERS_VERSION", "rust-v0.1.4")

	asset, err := tokenizerLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	require.NoError(t, err)

	archiveBytes, err := buildTestTokenizerArchive(asset.libraryFileName, []byte("dummy-tokenizer-library"))
	require.NoError(t, err)
	archiveChecksum := sha256.Sum256(archiveBytes)
	sumsContents := fmt.Sprintf("%s  %s\n", hex.EncodeToString(archiveChecksum[:]), asset.archiveFileName)
	sumsSignatureBody, sumsCertificateBody := newTokenizerSignedChecksumArtifacts(t, "rust-v0.1.4", []byte(sumsContents))

	stubDownload := func(filePath, url string) error {
		switch {
		case strings.Contains(url, "mirror-a.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsAsset):
			return os.WriteFile(filePath, []byte(sumsContents), 0600)
		case strings.Contains(url, "mirror-a.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsSignatureAsset):
			return os.WriteFile(filePath, sumsSignatureBody, 0600)
		case strings.Contains(url, "mirror-a.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsCertificateAsset):
			return os.WriteFile(filePath, sumsCertificateBody, 0600)
		case strings.Contains(url, "mirror-a.invalid") && strings.HasSuffix(url, "/"+asset.archiveFileName):
			return errors.New("simulated archive download failure on primary mirror")
		case strings.Contains(url, "mirror-b.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsAsset):
			return os.WriteFile(filePath, []byte(sumsContents), 0600)
		case strings.Contains(url, "mirror-b.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsSignatureAsset):
			return os.WriteFile(filePath, sumsSignatureBody, 0600)
		case strings.Contains(url, "mirror-b.invalid") && strings.HasSuffix(url, "/"+tokenizerChecksumsCertificateAsset):
			return os.WriteFile(filePath, sumsCertificateBody, 0600)
		case strings.Contains(url, "mirror-b.invalid") && strings.HasSuffix(url, "/"+asset.archiveFileName):
			return os.WriteFile(filePath, archiveBytes, 0600)
		default:
			return fmt.Errorf("unexpected URL: %s", url)
		}
	}
	tokenizerDownloadArtifactFileFunc = stubDownload
	tokenizerDownloadMetadataFileFunc = stubDownload

	libPath, err := ensureTokenizerLibraryDownloaded()
	require.NoError(t, err)
	require.FileExists(t, libPath)
	data, err := os.ReadFile(libPath)
	require.NoError(t, err)
	require.Equal(t, []byte("dummy-tokenizer-library"), data)
}

func TestEnsureTokenizerLibraryDownloaded_FailsOnChecksumMismatchAfterDownload(t *testing.T) {
	lockTokenizerTestHooks(t)
	bypassTokenizerCosignChainVerification(t)

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	originalHomeDir := tokenizerUserHomeDirFunc
	originalArtifactDownloadFunc := tokenizerDownloadArtifactFileFunc
	originalMetadataDownloadFunc := tokenizerDownloadMetadataFileFunc
	originalBuildInfo := tokenizerReadBuildInfoFunc
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
		tokenizerUserHomeDirFunc = originalHomeDir
		tokenizerDownloadArtifactFileFunc = originalArtifactDownloadFunc
		tokenizerDownloadMetadataFileFunc = originalMetadataDownloadFunc
		tokenizerReadBuildInfoFunc = originalBuildInfo
	})

	tempHome := t.TempDir()
	tokenizerUserHomeDirFunc = func() (string, error) { return tempHome, nil }
	tokenizerReadBuildInfoFunc = func() (*debug.BuildInfo, bool) { return nil, false }
	tokenizerReleaseBaseURL = "https://mirror.invalid/pure-tokenizers"
	tokenizerFallbackReleaseBaseURL = ""
	t.Setenv("TOKENIZERS_VERSION", "rust-v0.1.4")

	asset, err := tokenizerLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for tokenizer artifacts: %v", err)
	}

	archiveBytes, err := buildTestTokenizerArchive(asset.libraryFileName, []byte("dummy-tokenizer-library"))
	require.NoError(t, err)
	incorrectChecksum := sha256.Sum256([]byte("different-archive-bytes"))
	sumsContents := fmt.Sprintf("%s  %s\n", hex.EncodeToString(incorrectChecksum[:]), asset.archiveFileName)
	sumsSignatureBody, sumsCertificateBody := newTokenizerSignedChecksumArtifacts(t, "rust-v0.1.4", []byte(sumsContents))

	stubDownload := func(filePath, url string) error {
		switch {
		case strings.HasSuffix(url, "/"+tokenizerChecksumsAsset):
			return os.WriteFile(filePath, []byte(sumsContents), 0600)
		case strings.HasSuffix(url, "/"+tokenizerChecksumsSignatureAsset):
			return os.WriteFile(filePath, sumsSignatureBody, 0600)
		case strings.HasSuffix(url, "/"+tokenizerChecksumsCertificateAsset):
			return os.WriteFile(filePath, sumsCertificateBody, 0600)
		case strings.HasSuffix(url, "/"+asset.archiveFileName):
			return os.WriteFile(filePath, archiveBytes, 0600)
		default:
			return fmt.Errorf("unexpected URL: %s", url)
		}
	}
	tokenizerDownloadArtifactFileFunc = stubDownload
	tokenizerDownloadMetadataFileFunc = stubDownload

	_, err = ensureTokenizerLibraryDownloaded()
	require.Error(t, err)
	require.Contains(t, err.Error(), "tokenizers archive checksum verification failed")

	archivePath := filepath.Join(tempHome, ".cache", "chroma", "pure_tokenizers", "rust-v0.1.4", asset.platform, asset.archiveFileName)
	_, statErr := os.Stat(archivePath)
	require.True(t, os.IsNotExist(statErr), "corrupted archive should be cleaned up after checksum mismatch")
}

func TestEnsureTokenizerLibraryDownloadedFromPrimaryWhenFallbackUnavailable(t *testing.T) {
	lockTokenizerTestHooks(t)

	if os.Getenv("RUN_LIVE_TOKENIZERS_DOWNLOAD_TESTS") != "1" {
		t.Skip("set RUN_LIVE_TOKENIZERS_DOWNLOAD_TESTS=1 to run live download integration test")
	}

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	originalHomeDir := tokenizerUserHomeDirFunc
	originalArtifactDownloadFunc := tokenizerDownloadArtifactFileFunc
	originalMetadataDownloadFunc := tokenizerDownloadMetadataFileFunc
	originalBuildInfo := tokenizerReadBuildInfoFunc
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
		tokenizerUserHomeDirFunc = originalHomeDir
		tokenizerDownloadArtifactFileFunc = originalArtifactDownloadFunc
		tokenizerDownloadMetadataFileFunc = originalMetadataDownloadFunc
		tokenizerReadBuildInfoFunc = originalBuildInfo
	})

	tempHome := t.TempDir()
	tokenizerUserHomeDirFunc = func() (string, error) { return tempHome, nil }
	tokenizerDownloadArtifactFileFunc = tokenizerDownloadArtifactFileWithRetry
	tokenizerDownloadMetadataFileFunc = tokenizerDownloadMetadataFileWithRetry
	tokenizerReadBuildInfoFunc = func() (*debug.BuildInfo, bool) { return nil, false }
	tokenizerReleaseBaseURL = defaultTokenizerReleaseBaseURL
	tokenizerFallbackReleaseBaseURL = "https://127.0.0.1.invalid/pure-tokenizers"

	t.Setenv("TOKENIZERS_VERSION", "rust-v0.1.4")

	libPath, err := ensureTokenizerLibraryDownloaded()
	require.NoError(t, err)
	require.FileExists(t, libPath)
	require.Contains(t, libPath, filepath.Join(".cache", "chroma", "pure_tokenizers", "rust-v0.1.4"))
}

func buildTestTokenizerArchive(libraryFileName string, libraryContents []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)

	header := &tar.Header{
		Name: libraryFileName,
		Mode: 0600,
		Size: int64(len(libraryContents)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := io.Copy(tw, bytes.NewReader(libraryContents)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type tokenizerRoundTripFunc func(*http.Request) (*http.Response, error)

func (f tokenizerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func bypassTokenizerCosignChainVerification(t *testing.T) {
	t.Helper()
	original := tokenizerVerifyCosignCertificateChainFunc
	tokenizerVerifyCosignCertificateChainFunc = func(*x509.Certificate) error { return nil }
	t.Cleanup(func() {
		tokenizerVerifyCosignCertificateChainFunc = original
	})
}

func newTokenizerSignedChecksumArtifacts(t *testing.T, version string, checksumBody []byte) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	identity, err := url.Parse(fmt.Sprintf(tokenizerCosignIdentityTemplate, version))
	require.NoError(t, err)
	oidcIssuerValue, err := asn1.Marshal(tokenizerCosignOIDCIssuer)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "tokenizer-test-signer",
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(10 * time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{identity},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    cosignutil.OIDCIssuerExtensionOID(),
				Value: oidcIssuerValue,
			},
		},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	digest := sha256.Sum256(checksumBody)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	require.NoError(t, err)

	signatureBody := []byte(base64.StdEncoding.EncodeToString(signature))
	certificateBody := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	require.NotEmpty(t, certificateBody)
	return signatureBody, certificateBody
}
