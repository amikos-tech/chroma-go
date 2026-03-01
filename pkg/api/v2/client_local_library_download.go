package v2

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/internal/cosignutil"
	downloadutil "github.com/amikos-tech/chroma-go/pkg/internal/downloadutil"
)

const (
	defaultLocalLibraryVersion                = "v0.3.1"
	defaultLocalLibraryReleaseBaseURL         = "https://releases.amikos.tech/chroma-go-local"
	defaultLocalLibraryReleaseFallbackBaseURL = "https://github.com/amikos-tech/chroma-go-local/releases/download"
	localLibraryModulePath                    = "github.com/amikos-tech/chroma-go-local"
	localLibraryChecksumsAsset                = "SHA256SUMS"
	localLibraryChecksumsSignatureAsset       = "SHA256SUMS.sig"
	localLibraryChecksumsCertificateAsset     = "SHA256SUMS.pem"
	localLibraryArchivePrefixLegacy           = "chroma-go-local"
	localLibraryArchivePrefixLocalChroma      = "local-chroma"
	localLibraryCosignOIDCIssuer              = "https://token.actions.githubusercontent.com"
	localLibraryCosignIdentityTemplate        = "https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/tags/%s"
	localLibraryLockFileName                  = ".download.lock"
	localLibraryCacheDirPerm                  = os.FileMode(0700)
	localLibraryLockFilePerm                  = os.FileMode(0600)
	localLibraryArtifactFilePerm              = os.FileMode(0700)
)

var (
	// Intentionally mutable for tests that use httptest servers.
	localLibraryReleaseBaseURL            = defaultLocalLibraryReleaseBaseURL
	localLibraryReleaseFallbackBaseURL    = defaultLocalLibraryReleaseFallbackBaseURL
	localLibraryDownloadMu                sync.Mutex
	localLibraryDownloadAttempts                = 3
	localLibraryLockWaitTimeout                 = 45 * time.Second
	localLibraryLockStaleAfter                  = 10 * time.Minute
	localLibraryLockHeartbeatInterval           = 30 * time.Second
	localLibraryMaxArtifactBytes          int64 = 500 * 1024 * 1024
	localGetenvFunc                             = os.Getenv
	localUserHomeDirFunc                        = os.UserHomeDir
	localReadBuildInfoFunc                      = debug.ReadBuildInfo
	localDownloadFileFunc                       = localDownloadFileWithRetry
	localEnsureLibraryDownloadedFunc            = ensureLocalLibraryDownloaded
	localDetectLibraryVersionFunc               = detectLocalLibraryVersion
	localDefaultLibraryCacheDirFunc             = defaultLocalLibraryCacheDir
	localVerifyCosignCertificateChainFunc       = cosignutil.VerifyFulcioCertificateChain
	localValidateReleaseBaseURLFunc             = localValidateReleaseBaseURL
)

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
	archiveNames := localLibraryArchiveNames(version, asset.platform)
	if len(archiveNames) == 0 {
		return "", errors.New("no local runtime archive names available for selected platform")
	}

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
	var selectedArchiveName string
	var checksumsDownloadErrs []error
	for _, releaseBase := range releaseBases {
		archiveName, checksum, prepareErr := localPrepareSignedChecksumsFromBase(releaseBase, version, targetDir, archiveNames)
		if prepareErr != nil {
			checksumsDownloadErrs = append(checksumsDownloadErrs, errors.Wrapf(prepareErr, "release mirror %s failed", releaseBase))
			continue
		}
		selectedArchiveName = archiveName
		expectedChecksum = checksum
		selectedReleaseBase = releaseBase
		break
	}
	if selectedReleaseBase == "" || selectedArchiveName == "" || expectedChecksum == "" {
		return "", errors.Wrap(stderrors.Join(checksumsDownloadErrs...), "failed to prepare signed local library checksums")
	}

	archivePath := filepath.Join(targetDir, selectedArchiveName)
	archiveChecksumVerified := false
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if exists {
		if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
			verifyErr := errors.Wrap(err, "existing local runtime archive checksum verification failed")
			if removeErr := localRemoveCorruptedArchive(archivePath); removeErr != nil {
				return "", stderrors.Join(verifyErr, removeErr)
			}
			// Existing archive was corrupted and successfully removed; continue to re-download.
		} else {
			archiveChecksumVerified = true
		}
	}
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if !exists {
		archiveChecksumVerified = false
		if err := localDownloadReleaseAssetFromBase(selectedReleaseBase, version, selectedArchiveName, archivePath); err != nil {
			return "", errors.Wrap(err, "failed to download local library archive")
		}
	}

	if !archiveChecksumVerified {
		if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
			return "", localFailWithArchiveCleanup(archivePath, err, "local library archive checksum verification failed")
		}
	}
	if err := localVerifyTarGzContainsLibrary(archivePath, asset.libraryFileName); err != nil {
		return "", localFailWithArchiveCleanup(archivePath, err, "local library archive verification failed")
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

func localLibraryArchiveNames(version, platform string) []string {
	return []string{
		fmt.Sprintf("%s-%s-%s.tar.gz", localLibraryArchivePrefixLegacy, version, platform),
		fmt.Sprintf("%s-%s-%s.tar.gz", localLibraryArchivePrefixLocalChroma, version, platform),
	}
}

func localRemoveCorruptedArchive(archivePath string) error {
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove corrupted local runtime archive %s", archivePath)
	}
	return nil
}

func localFailWithArchiveCleanup(archivePath string, err error, msg string) error {
	verifyErr := errors.Wrap(err, msg)
	if removeErr := localRemoveCorruptedArchive(archivePath); removeErr != nil {
		return stderrors.Join(verifyErr, removeErr)
	}
	return verifyErr
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
		normalized, err := localValidateReleaseBaseURLFunc(base)
		if err != nil {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		bases = append(bases, normalized)
	}
	return bases
}

func localValidateReleaseBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return "", errors.New("release base URL cannot be empty")
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "invalid release base URL")
	}
	if !parsedURL.IsAbs() {
		return "", errors.New("release base URL must be absolute")
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return "", errors.Errorf("release base URL must use https scheme: %s", parsedURL.Redacted())
	}
	if strings.TrimSpace(parsedURL.Host) == "" {
		return "", errors.Errorf("release base URL host cannot be empty: %s", parsedURL.Redacted())
	}
	return baseURL, nil
}

func localPrepareSignedChecksumsFromBase(baseURL, version, targetDir string, archiveNames []string) (string, string, error) {
	checksumsPath := filepath.Join(targetDir, localLibraryChecksumsAsset)
	checksumsSignaturePath := filepath.Join(targetDir, localLibraryChecksumsSignatureAsset)
	checksumsCertificatePath := filepath.Join(targetDir, localLibraryChecksumsCertificateAsset)

	type metaDownload struct {
		asset, dest, errMsg string
	}
	downloads := []metaDownload{
		{localLibraryChecksumsAsset, checksumsPath, "failed to download local library checksums"},
		{localLibraryChecksumsSignatureAsset, checksumsSignaturePath, "failed to download local library checksums signature"},
		{localLibraryChecksumsCertificateAsset, checksumsCertificatePath, "failed to download local library checksums certificate"},
	}
	errs := make([]error, len(downloads))
	var wg sync.WaitGroup
	for i, dl := range downloads {
		wg.Add(1)
		go func(index int, download metaDownload) {
			defer wg.Done()
			if err := localDownloadReleaseAssetFromBase(baseURL, version, download.asset, download.dest); err != nil {
				errs[index] = errors.Wrap(err, download.errMsg)
			}
		}(i, dl)
	}
	wg.Wait()
	downloadErrs := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			downloadErrs = append(downloadErrs, err)
		}
	}
	if len(downloadErrs) > 0 {
		return "", "", errors.Wrap(stderrors.Join(downloadErrs...), "failed to download local library checksum metadata")
	}

	if err := localVerifySignedChecksums(version, checksumsPath, checksumsSignaturePath, checksumsCertificatePath); err != nil {
		return "", "", errors.Wrap(err, "failed to verify local library checksums signature")
	}

	resolvedArchiveName, expectedChecksum, err := localChecksumFromSumsFileAny(checksumsPath, archiveNames)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to resolve local library checksum")
	}
	return resolvedArchiveName, expectedChecksum, nil
}

func localDownloadReleaseAssetFromBase(baseURL, version, assetName, destinationPath string) error {
	normalizedBaseURL, err := localValidateReleaseBaseURLFunc(baseURL)
	if err != nil {
		return err
	}
	assetURL := fmt.Sprintf("%s/%s/%s", normalizedBaseURL, version, assetName)
	if err := localDownloadFileFunc(destinationPath, assetURL); err != nil {
		return errors.Wrapf(err, "download from %s failed", normalizedBaseURL)
	}
	return nil
}

func localVerifySignedChecksums(version, checksumsPath, signaturePath, certificatePath string) error {
	expectedIdentity := fmt.Sprintf(localLibraryCosignIdentityTemplate, version)
	return cosignutil.VerifySignedChecksums(
		checksumsPath, signaturePath, certificatePath,
		expectedIdentity, localLibraryCosignOIDCIssuer,
		localVerifyCosignCertificateChainFunc,
	)
}

// localChecksumFromSumsFileAny matches checksum entries in file order.
// If multiple candidate asset names are present, the first matching line in the checksums file wins.
func localChecksumFromSumsFileAny(sumsFilePath string, assetNames []string) (string, string, error) {
	candidates := make(map[string]string, len(assetNames))
	candidateList := make([]string, 0, len(assetNames))
	for _, assetName := range assetNames {
		original := strings.TrimSpace(assetName)
		normalized := localNormalizedChecksumAssetName(original)
		if normalized == "" {
			continue
		}
		if _, ok := candidates[normalized]; ok {
			continue
		}
		candidates[normalized] = original
		candidateList = append(candidateList, original)
	}
	if len(candidates) == 0 {
		return "", "", errors.New("asset names cannot be empty")
	}

	f, err := os.Open(sumsFilePath)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to open checksum file")
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
		checksumAssetName := localNormalizedChecksumAssetName(fields[1])
		if checksumAssetName == "" {
			continue
		}
		if originalAssetName, ok := candidates[checksumAssetName]; ok {
			checksum := strings.ToLower(fields[0])
			if !localLooksLikeSHA256(checksum) {
				return "", "", errors.Errorf("invalid checksum format for asset %s: %q", originalAssetName, fields[0])
			}
			return originalAssetName, checksum, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", errors.Wrap(err, "failed to read checksum file")
	}
	return "", "", errors.Errorf("checksum entry not found for assets [%s]", strings.Join(candidateList, ", "))
}

func localNormalizedChecksumAssetName(assetName string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(assetName), "*")
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = path.Base(normalized)
	if normalized == "." || normalized == "/" || normalized == ".." {
		return ""
	}
	return strings.TrimSpace(normalized)
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

func localLooksLikeSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func localDownloadFileWithRetry(filePath, url string) error {
	return errors.WithStack(downloadutil.DownloadFileWithRetry(
		filePath,
		url,
		localLibraryDownloadAttempts,
		downloadutil.Config{
			MaxBytes: localLibraryMaxArtifactBytes,
			DirPerm:  localLibraryCacheDirPerm,
		},
	))
}

func localVerifyTarGzContainsLibrary(filePath, libraryFileName string) error {
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
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "invalid tar archive")
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
		return nil
	}
	return errors.Errorf("library %s not found in archive", libraryFileName)
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
