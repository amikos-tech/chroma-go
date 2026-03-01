package tokenizers

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultTokenizerLibraryVersion               = "v0.1.4"
	defaultTokenizerReleaseBaseURL               = "https://releases.amikos.tech/pure-tokenizers"
	defaultTokenizerFallbackReleaseBaseURL       = "https://github.com/amikos-tech/pure-tokenizers/releases/download"
	tokenizerModulePath                          = "github.com/amikos-tech/pure-tokenizers"
	tokenizerLatestTag                           = "latest"
	tokenizerChecksumsAsset                      = "SHA256SUMS"
	tokenizerGitHubReleasesAPI                   = "https://api.github.com/repos/amikos-tech/pure-tokenizers/releases?per_page=100&page=1"
	tokenizerGitHubAPIVersion                    = "2022-11-28"
	tokenizerCacheDirPerm                        = os.FileMode(0700)
	tokenizerArtifactFilePerm                    = os.FileMode(0700)
	tokenizerMaxLibraryBytes               int64 = 200 * 1024 * 1024
	tokenizerMaxArtifactBytes              int64 = 500 * 1024 * 1024
)

var (
	tokenizerReleaseBaseURL         = defaultTokenizerReleaseBaseURL
	tokenizerFallbackReleaseBaseURL = defaultTokenizerFallbackReleaseBaseURL
	tokenizerDownloadMu             sync.Mutex
	tokenizerDownloadAttempts       = 3
	tokenizerDownloadFileFunc       = tokenizerDownloadFileWithRetry
	tokenizerReadBuildInfoFunc      = debug.ReadBuildInfo
	tokenizerUserHomeDirFunc        = os.UserHomeDir
)

type tokenizerLibraryAsset struct {
	platform        string
	archiveFileName string
	libraryFileName string
}

type tokenizerLatestRelease struct {
	Version string `json:"version"`
}

type tokenizerGitHubRelease struct {
	TagName string `json:"tag_name"`
}

func ensureTokenizerLibraryDownloaded() (string, error) {
	version, err := tokenizerResolveDownloadVersion()
	if err != nil {
		return "", err
	}
	if version == "" || version == tokenizerLatestTag {
		return "", errors.Errorf("failed to resolve concrete tokenizers version from %q", version)
	}

	cacheDir, err := defaultTokenizerLibraryCacheDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to determine tokenizer cache dir")
	}

	asset, err := tokenizerLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(cacheDir, version, asset.platform)
	targetLibraryPath := filepath.Join(targetDir, asset.libraryFileName)

	exists, err := tokenizerFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat tokenizers shared library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	tokenizerDownloadMu.Lock()
	defer tokenizerDownloadMu.Unlock()

	exists, err = tokenizerFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat tokenizers shared library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	if err := os.MkdirAll(targetDir, tokenizerCacheDirPerm); err != nil {
		return "", errors.Wrap(err, "failed to create tokenizers cache dir")
	}

	releaseBases := tokenizerReleaseBaseURLs()
	if len(releaseBases) == 0 {
		return "", errors.New("no tokenizers release base URL configured")
	}

	var selectedReleaseBase string
	var expectedChecksum string
	var checksumsDownloadErrs []error
	for _, releaseBase := range releaseBases {
		checksum, prepareErr := tokenizerPrepareChecksumFromBase(releaseBase, version, asset.archiveFileName, targetDir)
		if prepareErr != nil {
			checksumsDownloadErrs = append(checksumsDownloadErrs, errors.Wrapf(prepareErr, "release mirror %s failed", releaseBase))
			continue
		}
		selectedReleaseBase = releaseBase
		expectedChecksum = checksum
		break
	}
	if selectedReleaseBase == "" || expectedChecksum == "" {
		return "", errors.Wrap(stderrors.Join(checksumsDownloadErrs...), "failed to prepare tokenizers checksums")
	}

	archivePath := filepath.Join(targetDir, asset.archiveFileName)
	exists, err = tokenizerFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat tokenizers archive at %s", archivePath)
	}
	if exists {
		if err := tokenizerVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
			verifyErr := errors.Wrap(err, "existing tokenizers archive checksum verification failed")
			if removeErr := tokenizerRemoveCorruptedArchive(archivePath); removeErr != nil {
				return "", stderrors.Join(verifyErr, removeErr)
			}
		}
	}
	exists, err = tokenizerFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat tokenizers archive at %s", archivePath)
	}
	if !exists {
		if err := tokenizerDownloadReleaseAssetFromBase(selectedReleaseBase, version, asset.archiveFileName, archivePath); err != nil {
			return "", errors.Wrap(err, "failed to download tokenizers archive")
		}
	}

	if err := tokenizerVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
		return "", tokenizerFailWithArchiveCleanup(archivePath, err, "tokenizers archive checksum verification failed")
	}
	if err := tokenizerVerifyTarGzFile(archivePath); err != nil {
		return "", tokenizerFailWithArchiveCleanup(archivePath, err, "tokenizers archive verification failed")
	}

	tempLibraryPath := targetLibraryPath + ".tmp"
	_ = os.Remove(tempLibraryPath)
	if err := tokenizerExtractLibraryFromTarGz(archivePath, asset.libraryFileName, tempLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to extract tokenizers shared library")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempLibraryPath, tokenizerArtifactFilePerm); err != nil {
			_ = os.Remove(tempLibraryPath)
			return "", errors.Wrap(err, "failed to set permissions on tokenizers shared library")
		}
	}
	_ = os.Remove(targetLibraryPath)
	if err := os.Rename(tempLibraryPath, targetLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to finalize tokenizers shared library")
	}

	exists, err = tokenizerFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat extracted tokenizers shared library at %s", targetLibraryPath)
	}
	if !exists {
		return "", errors.Errorf("tokenizers shared library not found after extraction: %s", targetLibraryPath)
	}

	return targetLibraryPath, nil
}

func tokenizerResolveDownloadVersion() (string, error) {
	version := strings.TrimSpace(os.Getenv("TOKENIZERS_VERSION"))
	if version == "" {
		detectedVersion, err := detectTokenizerLibraryVersion()
		if err != nil {
			return "", errors.Wrap(err, "failed to detect tokenizers library version")
		}
		version = detectedVersion
	}
	if version == "" {
		version = defaultTokenizerLibraryVersion
	}

	version, err := normalizeTokenizerTag(version)
	if err != nil {
		return "", errors.Wrap(err, "invalid tokenizers library version")
	}
	if version == tokenizerLatestTag {
		latestVersion, latestErr := tokenizerResolveLatestVersion()
		if latestErr != nil {
			return "", errors.Wrap(latestErr, "failed to resolve latest tokenizers version")
		}
		return latestVersion, nil
	}
	return version, nil
}

func detectTokenizerLibraryVersion() (string, error) {
	defaultVersion := defaultTokenizerLibraryVersion
	buildInfo, ok := tokenizerReadBuildInfoFunc()
	if !ok || buildInfo == nil {
		return defaultVersion, nil
	}

	for _, dep := range buildInfo.Deps {
		if dep == nil || dep.Path != tokenizerModulePath {
			continue
		}
		version := dep.Version
		if dep.Replace != nil && dep.Replace.Version != "" {
			version = dep.Replace.Version
		}
		version = strings.TrimSpace(version)
		if version != "" {
			return version, nil
		}
	}
	return defaultVersion, nil
}

func defaultTokenizerLibraryCacheDir() (string, error) {
	homeDir, err := tokenizerUserHomeDirFunc()
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve home directory")
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", errors.New("home directory is empty")
	}
	return filepath.Join(homeDir, ".cache", "chroma", "pure_tokenizers"), nil
}

func tokenizerLibraryAssetForRuntime(goos, goarch string) (tokenizerLibraryAsset, error) {
	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return tokenizerLibraryAsset{}, errors.Errorf("unsupported architecture for tokenizers download: %s", goarch)
	}

	var targetTriple string
	var libraryFileName string
	var platform string
	switch goos {
	case "darwin":
		targetTriple = "apple-darwin"
		platform = "darwin-" + goarch
		libraryFileName = "libtokenizers.dylib"
	case "linux":
		if tokenizerIsMuslLinux() {
			targetTriple = "unknown-linux-musl"
			platform = "linux-" + goarch + "-musl"
		} else {
			targetTriple = "unknown-linux-gnu"
			platform = "linux-" + goarch
		}
		libraryFileName = "libtokenizers.so"
	case "windows":
		targetTriple = "pc-windows-msvc"
		platform = "windows-" + goarch
		libraryFileName = "tokenizers.dll"
	default:
		return tokenizerLibraryAsset{}, errors.Errorf("unsupported OS for tokenizers download: %s", goos)
	}

	return tokenizerLibraryAsset{
		platform:        platform,
		archiveFileName: fmt.Sprintf("libtokenizers-%s-%s.tar.gz", arch, targetTriple),
		libraryFileName: libraryFileName,
	}, nil
}

func tokenizerIsMuslLinux() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return true
	}
	lddContents, err := os.ReadFile("/usr/bin/ldd")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(lddContents)), "musl")
}

func tokenizerReleaseBaseURLs() []string {
	candidates := []string{
		strings.TrimSpace(tokenizerReleaseBaseURL),
		strings.TrimSpace(tokenizerFallbackReleaseBaseURL),
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

func tokenizerPrepareChecksumFromBase(baseURL, version, archiveName, targetDir string) (string, error) {
	checksumsPath := filepath.Join(targetDir, tokenizerChecksumsAsset)
	if err := tokenizerDownloadReleaseAssetFromBase(baseURL, version, tokenizerChecksumsAsset, checksumsPath); err != nil {
		return "", errors.Wrap(err, "failed to download tokenizers checksums")
	}
	checksum, err := tokenizerChecksumFromSumsFile(checksumsPath, archiveName)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve tokenizers checksum")
	}
	return checksum, nil
}

func tokenizerDownloadReleaseAssetFromBase(baseURL, version, assetName, destinationPath string) error {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return errors.New("release base URL cannot be empty")
	}
	url := fmt.Sprintf("%s/%s/%s", baseURL, version, assetName)
	if err := tokenizerDownloadFileFunc(destinationPath, url); err != nil {
		return errors.Wrapf(err, "download from %s failed", baseURL)
	}
	return nil
}

func tokenizerResolveLatestVersion() (string, error) {
	var errs []error

	for _, base := range tokenizerReleaseBaseURLs() {
		release, err := tokenizerFetchLatestVersionFromBase(base)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to fetch latest tokenizers version from %s", base))
			continue
		}
		release, err = normalizeTokenizerTag(release)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "latest tokenizers version from release metadata is invalid"))
			continue
		}
		if release == tokenizerLatestTag {
			errs = append(errs, errors.New("latest release metadata did not return a concrete version"))
			continue
		}
		return release, nil
	}

	githubVersion, githubErr := tokenizerFetchLatestVersionFromGitHub()
	if githubErr != nil {
		errs = append(errs, githubErr)
		return "", stderrors.Join(errs...)
	}

	githubVersion, err := normalizeTokenizerTag(githubVersion)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "latest tokenizers version from GitHub is invalid"))
		return "", stderrors.Join(errs...)
	}
	if githubVersion == tokenizerLatestTag {
		errs = append(errs, errors.New("latest tokenizers version from GitHub is not concrete"))
		return "", stderrors.Join(errs...)
	}
	return githubVersion, nil
}

func tokenizerFetchLatestVersionFromBase(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return "", errors.New("release base URL cannot be empty")
	}
	url := baseURL + "/latest.json"
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		CheckRedirect: tokenizerRejectHTTPSDowngradeRedirect,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to build latest version request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "chroma-go-tokenizers-downloader")
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch latest version metadata")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}

	var latest tokenizerLatestRelease
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return "", errors.Wrap(err, "failed to decode latest version metadata")
	}
	if strings.TrimSpace(latest.Version) == "" {
		return "", errors.Errorf("latest release metadata at %s is missing version", url)
	}
	return latest.Version, nil
}

func tokenizerFetchLatestVersionFromGitHub() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		CheckRedirect: tokenizerRejectHTTPSDowngradeRedirect,
	}
	req, err := http.NewRequest(http.MethodGet, tokenizerGitHubReleasesAPI, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to build GitHub latest version request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "chroma-go-tokenizers-downloader")
	req.Header.Set("X-GitHub-Api-Version", tokenizerGitHubAPIVersion)
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	} else if tok := os.Getenv("GH_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch GitHub releases")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected response %s for URL %s", resp.Status, tokenizerGitHubReleasesAPI)
	}

	var releases []tokenizerGitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", errors.Wrap(err, "failed to decode GitHub releases response")
	}
	for _, release := range releases {
		tag := strings.TrimSpace(release.TagName)
		if strings.HasPrefix(tag, "rust-v") {
			return tag, nil
		}
	}
	return "", errors.New("no rust-v* release found in GitHub releases")
}

func normalizeTokenizerTag(version string) (string, error) {
	version = strings.TrimSpace(version)
	switch {
	case version == "":
		return "", errors.New("tokenizers library version cannot be empty")
	case strings.EqualFold(version, tokenizerLatestTag):
		return tokenizerLatestTag, nil
	case strings.HasPrefix(version, "rust-v"):
		if err := validateTokenizerTag(version); err != nil {
			return "", err
		}
		return version, nil
	case strings.HasPrefix(version, "rust-"):
		suffix := strings.TrimPrefix(version, "rust-")
		if suffix == "" {
			return tokenizerLatestTag, nil
		}
		if suffix[0] >= '0' && suffix[0] <= '9' {
			version = "rust-v" + suffix
		}
	case strings.HasPrefix(version, "v"):
		version = "rust-" + version
	default:
		version = "rust-v" + version
	}

	if err := validateTokenizerTag(version); err != nil {
		return "", err
	}
	return version, nil
}

func validateTokenizerTag(version string) error {
	for _, r := range version {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		isAllowedPunct := r == '.' || r == '_' || r == '-'
		if !isLetter && !isDigit && !isAllowedPunct {
			return errors.New("tokenizers library version must contain only ASCII letters, digits, '.', '_' and '-'")
		}
	}
	return nil
}

func tokenizerNormalizedChecksumAssetName(assetName string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(assetName), "*")
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = path.Base(normalized)
	if normalized == "." || normalized == "/" || normalized == ".." {
		return ""
	}
	return strings.TrimSpace(normalized)
}

func tokenizerChecksumFromSumsFile(sumsFilePath, assetName string) (string, error) {
	assetName = tokenizerNormalizedChecksumAssetName(assetName)
	if assetName == "" {
		return "", errors.New("asset name cannot be empty")
	}

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
		checksumAssetName := tokenizerNormalizedChecksumAssetName(fields[1])
		if checksumAssetName == "" || checksumAssetName != assetName {
			continue
		}
		return strings.ToLower(fields[0]), nil
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read checksum file")
	}
	return "", errors.Errorf("checksum entry not found for asset %s", assetName)
}

func tokenizerVerifyFileChecksum(filePath, expectedChecksum string) error {
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

func tokenizerDownloadFileWithRetry(filePath, url string) error {
	var lastErr error
	for attempt := 1; attempt <= tokenizerDownloadAttempts; attempt++ {
		if err := tokenizerDownloadFile(filePath, url); err != nil {
			lastErr = err
			if attempt < tokenizerDownloadAttempts {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}
		return nil
	}
	return errors.Wrap(lastErr, "download failed after retries")
}

func tokenizerDownloadFile(filePath, url string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		CheckRedirect: tokenizerRejectHTTPSDowngradeRedirect,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "chroma-go-tokenizers-downloader")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}
	if resp.ContentLength > 0 && resp.ContentLength > tokenizerMaxArtifactBytes {
		return errors.Errorf(
			"downloaded artifact is too large: %d bytes exceeds max %d bytes",
			resp.ContentLength,
			tokenizerMaxArtifactBytes,
		)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), tokenizerCacheDirPerm); err != nil {
		return errors.Wrap(err, "failed to create destination directory")
	}

	out, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".download-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	tempPath := out.Name()

	limitedBody := io.LimitReader(resp.Body, tokenizerMaxArtifactBytes+1)
	written, copyErr := io.Copy(out, limitedBody)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(copyErr, "failed to copy HTTP response")
	}
	if written > tokenizerMaxArtifactBytes {
		_ = os.Remove(tempPath)
		return errors.Errorf(
			"downloaded artifact exceeds max allowed size: got %d bytes, max %d bytes",
			written,
			tokenizerMaxArtifactBytes,
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

func tokenizerRejectHTTPSDowngradeRedirect(req *http.Request, via []*http.Request) error {
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

func tokenizerVerifyTarGzFile(filePath string) error {
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

func tokenizerExtractLibraryFromTarGz(archivePath, libraryFileName, destinationPath string) error {
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
		if header.Size > tokenizerMaxLibraryBytes {
			return errors.Errorf(
				"library %s exceeds max allowed size: %d bytes > %d bytes",
				libraryFileName,
				header.Size,
				tokenizerMaxLibraryBytes,
			)
		}

		out, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, tokenizerArtifactFilePerm)
		if err != nil {
			return errors.Wrap(err, "failed to create extracted library file")
		}

		limitedReader := io.LimitReader(tarReader, tokenizerMaxLibraryBytes+1)
		written, copyErr := io.Copy(out, limitedReader)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(copyErr, "failed to extract library from archive")
		}
		if written > tokenizerMaxLibraryBytes {
			_ = os.Remove(destinationPath)
			return errors.Errorf(
				"extracted library exceeds max allowed size: got %d bytes, max %d bytes",
				written,
				tokenizerMaxLibraryBytes,
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

func tokenizerFileExistsNonEmpty(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !info.IsDir() && info.Size() > 0, nil
}

func tokenizerRemoveCorruptedArchive(archivePath string) error {
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove corrupted tokenizers archive %s", archivePath)
	}
	return nil
}

func tokenizerFailWithArchiveCleanup(archivePath string, err error, msg string) error {
	verifyErr := errors.Wrap(err, msg)
	if removeErr := tokenizerRemoveCorruptedArchive(archivePath); removeErr != nil {
		return stderrors.Join(verifyErr, removeErr)
	}
	return verifyErr
}
