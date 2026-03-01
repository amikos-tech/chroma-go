package v2

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultLocalLibraryVersion                = "v0.3.1"
	defaultLocalLibraryReleaseBaseURL         = "https://releases.amikos.tech/chroma-go-local"
	defaultLocalLibraryReleaseFallbackBaseURL = "https://github.com/amikos-tech/chroma-go-local/releases/download"
	localLibraryModulePath                    = "github.com/amikos-tech/chroma-go-local"
	localLibraryChecksumsAsset                = "SHA256SUMS"
	localLibraryChecksumsSignatureAsset       = "SHA256SUMS.sig"
	localLibraryChecksumsCertificateAsset     = "SHA256SUMS.pem"
	localLibraryCosignOIDCIssuer              = "https://token.actions.githubusercontent.com"
	localLibraryCosignIdentityTemplate        = "https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/tags/%s"
	localLibraryLockFileName                  = ".download.lock"
	localLibraryCacheDirPerm                  = os.FileMode(0700)
	localLibraryLockFilePerm                  = os.FileMode(0600)
	localLibraryArtifactFilePerm              = os.FileMode(0700)
)

var (
	// Intentionally mutable for tests that use httptest servers.
	localLibraryReleaseBaseURL         = defaultLocalLibraryReleaseBaseURL
	localLibraryReleaseFallbackBaseURL = defaultLocalLibraryReleaseFallbackBaseURL
	localLibraryDownloadMu             sync.Mutex
	localLibraryDownloadAttempts             = 3
	localLibraryLockWaitTimeout              = 45 * time.Second
	localLibraryLockStaleAfter               = 10 * time.Minute
	localLibraryLockHeartbeatInterval        = 30 * time.Second
	localLibraryMaxArtifactBytes       int64 = 500 * 1024 * 1024
	localGetenvFunc                          = os.Getenv
	localUserHomeDirFunc                     = os.UserHomeDir
	localReadBuildInfoFunc                   = debug.ReadBuildInfo
	localDownloadFileFunc                    = localDownloadFileWithRetry
	localEnsureLibraryDownloadedFunc         = ensureLocalLibraryDownloaded
	localDetectLibraryVersionFunc            = detectLocalLibraryVersion
	localDefaultLibraryCacheDirFunc          = defaultLocalLibraryCacheDir
)

var localLibraryCosignOIDCIssuerExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

type localLibraryAsset struct {
	platform        string
	libraryFileName string
}

func resolveLocalLibraryPath(cfg *localClientConfig) (string, error) {
	if cfg == nil {
		return "", errors.New("local client config cannot be nil")
	}

	if p := strings.TrimSpace(cfg.libraryPath); p != "" {
		return p, nil
	}

	if envPath := strings.TrimSpace(localGetenvFunc("CHROMA_LIB_PATH")); envPath != "" {
		return envPath, nil
	}

	if !cfg.autoDownloadLibrary {
		return "", errors.New("local runtime library path is not configured: set WithLocalLibraryPath(...), CHROMA_LIB_PATH, or enable WithLocalLibraryAutoDownload(true)")
	}

	version, err := normalizeLocalLibraryTag(cfg.libraryVersion)
	if err != nil {
		return "", errors.Wrap(err, "invalid local library version")
	}
	if version == "" {
		detectedVersion, detectErr := localDetectLibraryVersionFunc()
		if detectErr != nil {
			return "", errors.Wrap(detectErr, "failed to detect local library version")
		}
		version, err = normalizeLocalLibraryTag(detectedVersion)
		if err != nil {
			return "", errors.Wrap(err, "invalid detected local library version")
		}
	}
	if version == "" {
		version = defaultLocalLibraryVersion
	}

	cacheDir := strings.TrimSpace(cfg.libraryCacheDir)
	if cacheDir == "" {
		var err error
		cacheDir, err = localDefaultLibraryCacheDirFunc()
		if err != nil {
			return "", errors.Wrap(err, "failed to determine local library cache dir")
		}
	}

	libPath, err := localEnsureLibraryDownloadedFunc(version, cacheDir)
	if err != nil {
		return "", err
	}
	return libPath, nil
}

func detectLocalLibraryVersion() (string, error) {
	defaultVersion, err := normalizeLocalLibraryTag(defaultLocalLibraryVersion)
	if err != nil {
		return "", errors.Wrap(err, "invalid default local library version")
	}
	buildInfo, ok := localReadBuildInfoFunc()
	if !ok || buildInfo == nil {
		return defaultVersion, nil
	}

	for _, dep := range buildInfo.Deps {
		if dep == nil || dep.Path != localLibraryModulePath {
			continue
		}
		version := dep.Version
		if dep.Replace != nil && dep.Replace.Version != "" {
			version = dep.Replace.Version
		}
		version, err = normalizeLocalLibraryTag(version)
		if err != nil {
			return "", errors.Wrap(err, "invalid local library version in build info")
		}
		if version != "" {
			return version, nil
		}
	}
	return defaultVersion, nil
}

func defaultLocalLibraryCacheDir() (string, error) {
	homeDir, err := localUserHomeDirFunc()
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve home directory")
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", errors.New("home directory is empty")
	}
	return filepath.Join(homeDir, ".cache", "chroma", "local_shim"), nil
}

func ensureLocalLibraryDownloaded(version, cacheDir string) (libPath string, retErr error) {
	var err error
	version, err = normalizeLocalLibraryTag(version)
	if err != nil {
		return "", errors.Wrap(err, "invalid local library version")
	}
	if version == "" {
		return "", errors.New("local library version cannot be empty")
	}
	if strings.TrimSpace(cacheDir) == "" {
		return "", errors.New("local library cache dir cannot be empty")
	}

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	archiveName := localLibraryArchiveName(version, asset.platform)

	targetDir := filepath.Join(cacheDir, version, asset.platform)
	targetLibraryPath := filepath.Join(targetDir, asset.libraryFileName)
	exists, err := localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	localLibraryDownloadMu.Lock()
	defer localLibraryDownloadMu.Unlock()

	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	if err := os.MkdirAll(targetDir, localLibraryCacheDirPerm); err != nil {
		return "", errors.Wrap(err, "failed to create local library cache dir")
	}

	lockFile, err := localAcquireDownloadLock(filepath.Join(targetDir, localLibraryLockFileName))
	if err != nil {
		return "", err
	}
	defer func() {
		if releaseErr := localReleaseDownloadLock(lockFile); releaseErr != nil {
			wrapped := errors.Wrapf(releaseErr, "failed to release download lock %s", lockFile.Name())
			if retErr != nil {
				retErr = stderrors.Join(retErr, wrapped)
			} else {
				retErr = wrapped
			}
		}
	}()
	stopHeartbeat := localStartDownloadLockHeartbeat(lockFile)
	defer func() {
		if heartbeatErr := stopHeartbeat(); heartbeatErr != nil {
			wrapped := errors.Wrapf(heartbeatErr, "failed to stop download lock heartbeat for %s", lockFile.Name())
			if retErr != nil {
				retErr = stderrors.Join(retErr, wrapped)
			} else {
				retErr = wrapped
			}
		}
	}()

	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	releaseBases := localReleaseBaseURLs()
	if len(releaseBases) == 0 {
		return "", errors.New("no release base URL configured")
	}
	var expectedChecksum string
	var selectedReleaseBase string
	var checksumsDownloadErrs []error
	for _, releaseBase := range releaseBases {
		checksum, prepareErr := localPrepareSignedChecksumsFromBase(releaseBase, version, targetDir, archiveName)
		if prepareErr != nil {
			checksumsDownloadErrs = append(checksumsDownloadErrs, errors.Wrapf(prepareErr, "release mirror %s failed", releaseBase))
			continue
		}
		expectedChecksum = checksum
		selectedReleaseBase = releaseBase
		break
	}
	if selectedReleaseBase == "" {
		return "", errors.Wrap(stderrors.Join(checksumsDownloadErrs...), "failed to prepare signed local library checksums")
	}

	archivePath := filepath.Join(targetDir, archiveName)
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if exists {
		if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
			_ = os.Remove(archivePath)
		}
	}
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if !exists {
		if err := localDownloadReleaseAssetFromBase(selectedReleaseBase, version, archiveName, archivePath); err != nil {
			return "", errors.Wrap(err, "failed to download local library archive")
		}
	}

	if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
		_ = os.Remove(archivePath)
		return "", errors.Wrap(err, "local library archive checksum verification failed")
	}
	if err := localVerifyTarGzFile(archivePath); err != nil {
		_ = os.Remove(archivePath)
		return "", errors.Wrap(err, "local library archive verification failed")
	}

	tempLibraryPath := targetLibraryPath + ".tmp"
	_ = os.Remove(tempLibraryPath)
	if err := localExtractLibraryFromTarGz(archivePath, asset.libraryFileName, tempLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to extract local runtime library")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempLibraryPath, localLibraryArtifactFilePerm); err != nil {
			_ = os.Remove(tempLibraryPath)
			return "", errors.Wrap(err, "failed to set permissions on local runtime library")
		}
	}
	_ = os.Remove(targetLibraryPath)
	if err := os.Rename(tempLibraryPath, targetLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to finalize local runtime library")
	}
	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat extracted local runtime library at %s", targetLibraryPath)
	}
	if !exists {
		return "", errors.Errorf("local runtime library not found after extraction: %s", targetLibraryPath)
	}

	return targetLibraryPath, nil
}

func localLibraryAssetForRuntime(goos, goarch string) (localLibraryAsset, error) {
	var platformOS, requiredArch, libraryFileName string
	switch goos {
	case "linux":
		platformOS, requiredArch, libraryFileName = "linux", "amd64", "libchroma_shim.so"
	case "darwin":
		platformOS, requiredArch, libraryFileName = "darwin", "arm64", "libchroma_shim.dylib"
	case "windows":
		platformOS, requiredArch, libraryFileName = "windows", "amd64", "chroma_shim.dll"
	default:
		return localLibraryAsset{}, errors.Errorf("unsupported OS for local runtime download: %s", goos)
	}
	if goarch != requiredArch {
		return localLibraryAsset{}, errors.Errorf("unsupported architecture for %s local runtime download: %s", goos, goarch)
	}
	platform := platformOS + "-" + goarch
	return localLibraryAsset{
		platform:        platform,
		libraryFileName: libraryFileName,
	}, nil
}

func localLibraryArchiveName(version, platform string) string {
	return fmt.Sprintf("chroma-go-local-%s-%s.tar.gz", version, platform)
}

func validateLocalLibraryTag(version string) error {
	for _, r := range version {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		isAllowedPunct := r == '.' || r == '_' || r == '-'
		if !isLetter && !isDigit && !isAllowedPunct {
			return errors.New("local library version must contain only ASCII letters, digits, '.', '_' and '-'")
		}
	}
	return nil
}

func normalizeLocalLibraryTag(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return "", nil
	}
	if err := validateLocalLibraryTag(version); err != nil {
		return "", err
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version, nil
	}
	return version, nil
}

func localShouldEvictStaleLock(lockPath string, initialInfo os.FileInfo) (bool, error) {
	if initialInfo == nil || time.Since(initialInfo.ModTime()) <= localLibraryLockStaleAfter {
		return false, nil
	}

	currentInfo, err := os.Stat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to stat stale lock file %s", lockPath)
	}

	if !os.SameFile(initialInfo, currentInfo) {
		return false, nil
	}
	if !currentInfo.ModTime().Equal(initialInfo.ModTime()) || currentInfo.Size() != initialInfo.Size() {
		return false, nil
	}
	if time.Since(currentInfo.ModTime()) <= localLibraryLockStaleAfter {
		return false, nil
	}
	return true, nil
}

func localAcquireDownloadLock(lockPath string) (*os.File, error) {
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, localLibraryCacheDirPerm); err != nil {
		return nil, errors.Wrap(err, "failed to create lock directory")
	}

	deadline := time.Now().Add(localLibraryLockWaitTimeout)
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, localLibraryLockFilePerm)
		if err == nil {
			_, _ = fmt.Fprintf(lockFile, "%d", os.Getpid())
			_ = lockFile.Sync()
			return lockFile, nil
		}
		if !os.IsExist(err) {
			return nil, errors.Wrap(err, "failed to create lock file")
		}

		if info, statErr := os.Stat(lockPath); statErr == nil {
			evictStale, staleErr := localShouldEvictStaleLock(lockPath, info)
			if staleErr != nil {
				return nil, staleErr
			}
			if evictStale {
				if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
					return nil, errors.Wrapf(removeErr, "failed to remove stale lock file %s", lockPath)
				}
				continue
			}
		}

		if time.Now().After(deadline) {
			return nil, errors.Errorf("timeout waiting for lock: %s", lockPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func localReleaseDownloadLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	lockPath := lockFile.Name()
	var errs []error
	if closeErr := lockFile.Close(); closeErr != nil {
		errs = append(errs, errors.Wrap(closeErr, "failed to close download lock file"))
	}
	if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
		errs = append(errs, errors.Wrapf(removeErr, "failed to remove download lock file %s", lockPath))
	}
	if len(errs) > 0 {
		return stderrors.Join(errs...)
	}
	return nil
}

func localStartDownloadLockHeartbeat(lockFile *os.File) func() error {
	if lockFile == nil || localLibraryLockHeartbeatInterval <= 0 {
		return func() error { return nil }
	}

	lockPath := lockFile.Name()
	stopCh := make(chan struct{})
	doneCh := make(chan error, 1)
	var stopOnce sync.Once
	var stopErr error
	go func() {
		ticker := time.NewTicker(localLibraryLockHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if err := os.Chtimes(lockPath, now, now); err != nil {
					if os.IsNotExist(err) {
						doneCh <- nil
					} else {
						doneCh <- errors.Wrapf(err, "failed to refresh download lock file %s", lockPath)
					}
					return
				}
			case <-stopCh:
				doneCh <- nil
				return
			}
		}
	}()

	return func() error {
		stopOnce.Do(func() {
			close(stopCh)
			stopErr = <-doneCh
		})
		return stopErr
	}
}

func localReleaseBaseURLs() []string {
	candidates := []string{
		strings.TrimSpace(localLibraryReleaseBaseURL),
		strings.TrimSpace(localLibraryReleaseFallbackBaseURL),
	}
	seen := make(map[string]struct{}, len(candidates))
	bases := make([]string, 0, len(candidates))
	for _, base := range candidates {
		base = strings.TrimRight(base, "/")
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		bases = append(bases, base)
	}
	return bases
}

func localPrepareSignedChecksumsFromBase(baseURL, version, targetDir, archiveName string) (string, error) {
	checksumsPath := filepath.Join(targetDir, localLibraryChecksumsAsset)
	if err := localDownloadReleaseAssetFromBase(baseURL, version, localLibraryChecksumsAsset, checksumsPath); err != nil {
		return "", errors.Wrap(err, "failed to download local library checksums")
	}
	checksumsSignaturePath := filepath.Join(targetDir, localLibraryChecksumsSignatureAsset)
	if err := localDownloadReleaseAssetFromBase(baseURL, version, localLibraryChecksumsSignatureAsset, checksumsSignaturePath); err != nil {
		return "", errors.Wrap(err, "failed to download local library checksums signature")
	}
	checksumsCertificatePath := filepath.Join(targetDir, localLibraryChecksumsCertificateAsset)
	if err := localDownloadReleaseAssetFromBase(baseURL, version, localLibraryChecksumsCertificateAsset, checksumsCertificatePath); err != nil {
		return "", errors.Wrap(err, "failed to download local library checksums certificate")
	}
	if err := localVerifySignedChecksums(version, checksumsPath, checksumsSignaturePath, checksumsCertificatePath); err != nil {
		return "", errors.Wrap(err, "failed to verify local library checksums signature")
	}

	expectedChecksum, err := localChecksumFromSumsFile(checksumsPath, archiveName)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve local library checksum")
	}
	return expectedChecksum, nil
}

func localDownloadReleaseAssetFromBase(baseURL, version, assetName, destinationPath string) error {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return errors.New("release base URL cannot be empty")
	}
	url := fmt.Sprintf("%s/%s/%s", baseURL, version, assetName)
	if err := localDownloadFileFunc(destinationPath, url); err != nil {
		return errors.Wrapf(err, "download from %s failed", baseURL)
	}
	return nil
}

func localVerifySignedChecksums(version, checksumsPath, signaturePath, certificatePath string) error {
	checksumsBytes, err := os.ReadFile(checksumsPath)
	if err != nil {
		return errors.Wrap(err, "failed to read checksum file")
	}

	signature, err := localReadBase64EncodedFile(signaturePath)
	if err != nil {
		return errors.Wrap(err, "failed to decode checksum signature")
	}

	certificate, err := localReadCosignCertificate(certificatePath)
	if err != nil {
		return errors.Wrap(err, "failed to parse checksum certificate")
	}

	expectedIdentity := fmt.Sprintf(localLibraryCosignIdentityTemplate, version)
	if err := localValidateCosignCertificate(certificate, expectedIdentity, localLibraryCosignOIDCIssuer); err != nil {
		return errors.Wrap(err, "certificate validation failed")
	}

	if err := localVerifyBlobSignature(certificate, checksumsBytes, signature); err != nil {
		return errors.Wrap(err, "invalid checksum signature")
	}
	return nil
}

func localReadBase64EncodedFile(filePath string) ([]byte, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return localDecodeBase64Bytes(raw)
}

func localDecodeBase64Bytes(raw []byte) ([]byte, error) {
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

func localReadCosignCertificate(filePath string) (*x509.Certificate, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	certPEM := bytes.TrimSpace(raw)
	if len(certPEM) == 0 {
		return nil, errors.New("certificate payload is empty")
	}
	if !bytes.Contains(certPEM, []byte("BEGIN CERTIFICATE")) {
		decoded, decodeErr := localDecodeBase64Bytes(certPEM)
		if decodeErr != nil {
			return nil, decodeErr
		}
		certPEM = decoded
	}
	block, remainder := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}
	if len(bytes.TrimSpace(remainder)) > 0 {
		return nil, errors.New("certificate PEM contains unexpected trailing data")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

func localValidateCosignCertificate(certificate *x509.Certificate, expectedIdentity, expectedOIDCIssuer string) error {
	if certificate == nil {
		return errors.New("certificate is nil")
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

	issuerValue, foundIssuer, err := localCertificateExtensionValue(certificate, localLibraryCosignOIDCIssuerExtensionOID)
	if err != nil {
		return err
	}
	if !foundIssuer {
		return errors.Errorf("certificate missing OIDC issuer extension %s", localLibraryCosignOIDCIssuerExtensionOID.String())
	}
	if issuerValue != expectedOIDCIssuer {
		return errors.Errorf("certificate OIDC issuer mismatch: expected %s, got %s", expectedOIDCIssuer, issuerValue)
	}
	return nil
}

func localCertificateExtensionValue(certificate *x509.Certificate, oid asn1.ObjectIdentifier) (string, bool, error) {
	for _, extension := range certificate.Extensions {
		if !extension.Id.Equal(oid) {
			continue
		}
		var value string
		if _, err := asn1.Unmarshal(extension.Value, &value); err != nil {
			return "", false, errors.Wrapf(err, "failed to decode certificate extension %s", oid.String())
		}
		return value, true, nil
	}
	return "", false, nil
}

func localVerifyBlobSignature(certificate *x509.Certificate, payload, signature []byte) error {
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

func localChecksumFromSumsFile(sumsFilePath, assetName string) (string, error) {
	f, err := os.Open(sumsFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open checksum file")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		checksumAssetName := strings.TrimPrefix(fields[1], "*")
		if checksumAssetName == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read checksum file")
	}
	return "", errors.Errorf("checksum entry not found for asset %s", assetName)
}

func localVerifyFileChecksum(filePath, expectedChecksum string) error {
	expectedChecksum = strings.TrimSpace(strings.ToLower(expectedChecksum))
	if expectedChecksum == "" {
		return errors.New("expected checksum cannot be empty")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file for checksum verification")
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return errors.Wrap(err, "failed to hash downloaded file")
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return errors.Errorf("checksum mismatch for %s: expected %s, got %s", filePath, expectedChecksum, actualChecksum)
	}
	return nil
}

func localDownloadFileWithRetry(filePath, url string) error {
	var lastErr error
	for attempt := 1; attempt <= localLibraryDownloadAttempts; attempt++ {
		if err := localDownloadFile(filePath, url); err != nil {
			lastErr = err
			if attempt < localLibraryDownloadAttempts {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}
		return nil
	}
	return errors.Wrap(lastErr, "download failed after retries")
}

func localDownloadFile(filePath, url string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		CheckRedirect: localRejectHTTPSDowngradeRedirect,
	}

	resp, err := client.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}
	if resp.ContentLength > 0 && resp.ContentLength > localLibraryMaxArtifactBytes {
		return errors.Errorf(
			"downloaded artifact is too large: %d bytes exceeds max %d bytes",
			resp.ContentLength,
			localLibraryMaxArtifactBytes,
		)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), localLibraryCacheDirPerm); err != nil {
		return errors.Wrap(err, "failed to create destination directory")
	}

	out, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".download-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	tempPath := out.Name()

	limitedBody := io.LimitReader(resp.Body, localLibraryMaxArtifactBytes+1)
	written, copyErr := io.Copy(out, limitedBody)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(copyErr, "failed to copy HTTP response")
	}
	if written > localLibraryMaxArtifactBytes {
		_ = os.Remove(tempPath)
		return errors.Errorf(
			"downloaded artifact exceeds max allowed size: got %d bytes, max %d bytes",
			written,
			localLibraryMaxArtifactBytes,
		)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(closeErr, "failed to close temp file")
	}
	if resp.ContentLength > 0 && written != resp.ContentLength {
		_ = os.Remove(tempPath)
		return errors.Errorf("download incomplete: expected %d bytes, got %d bytes", resp.ContentLength, written)
	}

	_ = os.Remove(filePath)
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(err, "failed to finalize downloaded file")
	}
	return nil
}

func localRejectHTTPSDowngradeRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if len(via) == 0 {
		return nil
	}
	previousReq := via[len(via)-1]
	if previousReq.URL != nil && req.URL != nil &&
		strings.EqualFold(previousReq.URL.Scheme, "https") &&
		strings.EqualFold(req.URL.Scheme, "http") {
		return errors.Errorf(
			"redirect from HTTPS to HTTP is not allowed: %s -> %s",
			previousReq.URL.String(),
			req.URL.String(),
		)
	}
	return nil
}

func localVerifyTarGzFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "could not open archive for verification")
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "invalid gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	if _, err := tarReader.Next(); err != nil {
		return errors.Wrap(err, "invalid tar archive")
	}
	return nil
}

func localExtractLibraryFromTarGz(archivePath, libraryFileName, destinationPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "failed to read gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar entry")
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != libraryFileName {
			continue
		}
		if header.Size <= 0 {
			return errors.Errorf("library %s has invalid size %d in archive", libraryFileName, header.Size)
		}
		if header.Size > localLibraryMaxArtifactBytes {
			return errors.Errorf(
				"library %s exceeds max allowed size: %d bytes > %d bytes",
				libraryFileName,
				header.Size,
				localLibraryMaxArtifactBytes,
			)
		}

		out, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, localLibraryArtifactFilePerm)
		if err != nil {
			return errors.Wrap(err, "failed to create extracted library file")
		}

		limitedReader := io.LimitReader(tarReader, localLibraryMaxArtifactBytes+1)
		written, copyErr := io.Copy(out, limitedReader)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(copyErr, "failed to extract library from archive")
		}
		if written > localLibraryMaxArtifactBytes {
			_ = os.Remove(destinationPath)
			return errors.Errorf(
				"extracted library exceeds max allowed size: got %d bytes, max %d bytes",
				written,
				localLibraryMaxArtifactBytes,
			)
		}
		if written != header.Size {
			_ = os.Remove(destinationPath)
			return errors.Errorf(
				"extracted library size mismatch: expected %d bytes, got %d bytes",
				header.Size,
				written,
			)
		}
		if syncErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(syncErr, "failed to sync extracted library")
		}
		if closeErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(closeErr, "failed to close extracted library")
		}
		return nil
	}
	return errors.Errorf("library %s not found in archive", libraryFileName)
}

func localFileExistsNonEmpty(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !info.IsDir() && info.Size() > 0, nil
}
