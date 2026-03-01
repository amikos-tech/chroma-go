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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/internal/cosignutil"
	downloadutil "github.com/amikos-tech/chroma-go/pkg/internal/downloadutil"
)

const (
	defaultTokenizerLibraryVersion               = "v0.1.4"
	defaultTokenizerReleaseBaseURL               = "https://releases.amikos.tech/pure-tokenizers"
	defaultTokenizerFallbackReleaseBaseURL       = "https://github.com/amikos-tech/pure-tokenizers/releases/download"
	tokenizerModulePath                          = "github.com/amikos-tech/pure-tokenizers"
	tokenizerLatestTag                           = "latest"
	tokenizerChecksumsAsset                      = "SHA256SUMS"
	tokenizerChecksumsSignatureAsset             = "SHA256SUMS.sig"
	tokenizerChecksumsCertificateAsset           = "SHA256SUMS.pem"
	tokenizerGitHubReleasesAPI                   = "https://api.github.com/repos/amikos-tech/pure-tokenizers/releases?per_page=100&page=1"
	tokenizerGitHubAPIVersion                    = "2022-11-28"
	tokenizerCacheDirPerm                        = os.FileMode(0700)
	tokenizerArtifactFilePerm                    = os.FileMode(0700)
	tokenizerCosignOIDCIssuer                    = "https://token.actions.githubusercontent.com"
	tokenizerCosignIdentityTemplate              = "https://github.com/amikos-tech/pure-tokenizers/.github/workflows/rust-release.yml@refs/tags/%s"
	tokenizerDownloaderUserAgent                 = "chroma-go-tokenizers-downloader"
	tokenizerMaxVersionTagLength                 = 128
	tokenizerMaxLibraryBytes               int64 = 200 * 1024 * 1024
	tokenizerMaxArtifactBytes              int64 = 500 * 1024 * 1024
	tokenizerMaxMetadataBytes              int64 = 5 * 1024 * 1024
)

var (
	tokenizerReleaseBaseURL                   = defaultTokenizerReleaseBaseURL
	tokenizerFallbackReleaseBaseURL           = defaultTokenizerFallbackReleaseBaseURL
	tokenizerDownloadMu                       sync.Mutex
	tokenizerDownloadAttempts                 = 3
	tokenizerDownloadArtifactFileFunc         = tokenizerDownloadArtifactFileWithRetry
	tokenizerDownloadMetadataFileFunc         = tokenizerDownloadMetadataFileWithRetry
	tokenizerReadBuildInfoFunc                = debug.ReadBuildInfo
	tokenizerUserHomeDirFunc                  = os.UserHomeDir
	tokenizerGetMetadataHTTPClientFunc        = tokenizerGetMetadataHTTPClient
	tokenizerVerifyCosignCertificateChainFunc = cosignutil.VerifyFulcioCertificateChain

	tokenizerMetadataClientOnce sync.Once
	tokenizerMetadataClient     *http.Client
)

func tokenizerGetMetadataHTTPClient() *http.Client {
	tokenizerMetadataClientOnce.Do(func() {
		tokenizerMetadataClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
			CheckRedirect: downloadutil.RejectHTTPSDowngradeRedirect,
		}
	})
	return tokenizerMetadataClient
}

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

	archivePath := filepath.Join(targetDir, asset.archiveFileName)
	var mirrorErrs []error
	for _, releaseBase := range releaseBases {
		checksum, prepareErr := tokenizerPrepareChecksumFromBase(releaseBase, version, asset.archiveFileName, targetDir)
		if prepareErr != nil {
			mirrorErrs = append(mirrorErrs, errors.Wrapf(prepareErr, "release mirror %s failed preparing checksums", releaseBase))
			continue
		}

		exists, err = tokenizerFileExistsNonEmpty(archivePath)
		if err != nil {
			mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed stat tokenizers archive at %s", releaseBase, archivePath))
			continue
		}
		if exists {
			if err := tokenizerVerifyFileChecksum(archivePath, checksum); err != nil {
				verifyErr := errors.Wrap(err, "existing tokenizers archive checksum verification failed")
				if removeErr := tokenizerRemoveCorruptedArchive(archivePath); removeErr != nil {
					mirrorErrs = append(mirrorErrs, errors.Wrapf(stderrors.Join(verifyErr, removeErr), "release mirror %s failed", releaseBase))
					continue
				}
				exists = false
			}
		}
		if !exists {
			if err := tokenizerDownloadReleaseAssetFromBase(releaseBase, version, asset.archiveFileName, archivePath); err != nil {
				mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed downloading tokenizers archive", releaseBase))
				continue
			}
		}

		if err := tokenizerVerifyFileChecksum(archivePath, checksum); err != nil {
			mirrorErrs = append(
				mirrorErrs,
				errors.Wrapf(
					tokenizerFailWithArchiveCleanup(archivePath, err, "tokenizers archive checksum verification failed"),
					"release mirror %s failed",
					releaseBase,
				),
			)
			continue
		}
		tempLibraryPath := targetLibraryPath + ".tmp"
		_ = os.Remove(tempLibraryPath)
		if err := tokenizerExtractLibraryFromTarGz(archivePath, asset.libraryFileName, tempLibraryPath); err != nil {
			_ = os.Remove(tempLibraryPath)
			mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed extracting tokenizers shared library", releaseBase))
			continue
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(tempLibraryPath, tokenizerArtifactFilePerm); err != nil {
				_ = os.Remove(tempLibraryPath)
				mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed setting permissions on tokenizers shared library", releaseBase))
				continue
			}
		}
		_ = os.Remove(targetLibraryPath)
		if err := os.Rename(tempLibraryPath, targetLibraryPath); err != nil {
			_ = os.Remove(tempLibraryPath)
			mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed finalizing tokenizers shared library", releaseBase))
			continue
		}

		exists, err = tokenizerFileExistsNonEmpty(targetLibraryPath)
		if err != nil {
			mirrorErrs = append(mirrorErrs, errors.Wrapf(err, "release mirror %s failed stat extracted tokenizers shared library at %s", releaseBase, targetLibraryPath))
			continue
		}
		if !exists {
			mirrorErrs = append(mirrorErrs, errors.Wrapf(errors.Errorf("tokenizers shared library not found after extraction: %s", targetLibraryPath), "release mirror %s failed", releaseBase))
			continue
		}
		return targetLibraryPath, nil
	}
	if len(mirrorErrs) == 0 {
		return "", errors.New("failed to download tokenizers library: no release mirrors available")
	}
	return "", errors.Wrap(stderrors.Join(mirrorErrs...), "failed to download and extract tokenizers library from all release mirrors")
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
	lddFile, err := os.Open("/usr/bin/ldd")
	if err != nil {
		return false
	}
	defer lddFile.Close()

	lddContents, err := io.ReadAll(io.LimitReader(lddFile, 128*1024))
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
		normalized, err := tokenizerValidateReleaseBaseURL(base)
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

func tokenizerValidateReleaseBaseURL(baseURL string) (string, error) {
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
		return "", errors.Errorf("release base URL must use https scheme: %s", baseURL)
	}
	if strings.TrimSpace(parsedURL.Host) == "" {
		return "", errors.Errorf("release base URL host cannot be empty: %s", baseURL)
	}
	return baseURL, nil
}

func tokenizerPrepareChecksumFromBase(baseURL, version, archiveName, targetDir string) (string, error) {
	checksumsPath := filepath.Join(targetDir, tokenizerChecksumsAsset)
	checksumsSignaturePath := filepath.Join(targetDir, tokenizerChecksumsSignatureAsset)
	checksumsCertificatePath := filepath.Join(targetDir, tokenizerChecksumsCertificateAsset)

	type metaDownload struct {
		asset, dest, errMsg string
	}
	downloads := []metaDownload{
		{tokenizerChecksumsAsset, checksumsPath, "failed to download tokenizers checksums"},
		{tokenizerChecksumsSignatureAsset, checksumsSignaturePath, "failed to download tokenizers checksums signature"},
		{tokenizerChecksumsCertificateAsset, checksumsCertificatePath, "failed to download tokenizers checksums certificate"},
	}
	errs := make([]error, len(downloads))
	var wg sync.WaitGroup
	for i, dl := range downloads {
		wg.Add(1)
		go func(index int, download metaDownload) {
			defer wg.Done()
			if err := tokenizerDownloadMetadataAssetFromBase(baseURL, version, download.asset, download.dest); err != nil {
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
		return "", errors.Wrap(stderrors.Join(downloadErrs...), "failed to download tokenizers checksum metadata")
	}

	if err := tokenizerVerifySignedChecksums(version, checksumsPath, checksumsSignaturePath, checksumsCertificatePath); err != nil {
		return "", errors.Wrap(err, "failed to verify tokenizers checksums signature")
	}

	checksum, err := tokenizerChecksumFromSumsFile(checksumsPath, archiveName)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve tokenizers checksum")
	}
	return checksum, nil
}

func tokenizerDownloadReleaseAssetFromBase(baseURL, version, assetName, destinationPath string) error {
	normalizedBaseURL, err := tokenizerValidateReleaseBaseURL(baseURL)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%s", normalizedBaseURL, version, assetName)
	if err := tokenizerDownloadArtifactFileFunc(destinationPath, url); err != nil {
		return errors.Wrapf(err, "download from %s failed", normalizedBaseURL)
	}
	return nil
}

func tokenizerDownloadMetadataAssetFromBase(baseURL, version, assetName, destinationPath string) error {
	normalizedBaseURL, err := tokenizerValidateReleaseBaseURL(baseURL)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%s", normalizedBaseURL, version, assetName)
	if err := tokenizerDownloadMetadataFileFunc(destinationPath, url); err != nil {
		return errors.Wrapf(err, "download from %s failed", normalizedBaseURL)
	}
	return nil
}

func tokenizerVerifySignedChecksums(version, checksumsPath, signaturePath, certificatePath string) error {
	expectedIdentity := fmt.Sprintf(tokenizerCosignIdentityTemplate, version)
	return cosignutil.VerifySignedChecksums(
		checksumsPath, signaturePath, certificatePath,
		expectedIdentity, tokenizerCosignOIDCIssuer,
		tokenizerVerifyCosignCertificateChainFunc,
	)
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
	normalizedBaseURL, err := tokenizerValidateReleaseBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	url := normalizedBaseURL + "/latest.json"
	client := tokenizerGetMetadataHTTPClientFunc()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to build latest version request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", tokenizerDownloaderUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch latest version metadata")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}

	var latest tokenizerLatestRelease
	if err := tokenizerDecodeJSONResponse(resp.Body, tokenizerMaxMetadataBytes, &latest); err != nil {
		return "", errors.Wrap(err, "failed to decode latest version metadata")
	}
	if strings.TrimSpace(latest.Version) == "" {
		return "", errors.Errorf("latest release metadata at %s is missing version", url)
	}
	return latest.Version, nil
}

func tokenizerFetchLatestVersionFromGitHub() (string, error) {
	client := tokenizerGetMetadataHTTPClientFunc()
	req, err := http.NewRequest(http.MethodGet, tokenizerGitHubReleasesAPI, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to build GitHub latest version request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", tokenizerDownloaderUserAgent)
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
	if err := tokenizerDecodeJSONResponse(resp.Body, tokenizerMaxMetadataBytes, &releases); err != nil {
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

func tokenizerDecodeJSONResponse(body io.Reader, maxBytes int64, out any) error {
	if body == nil {
		return errors.New("response body is nil")
	}
	if maxBytes <= 0 {
		return errors.New("max metadata size must be greater than zero")
	}

	payload, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return errors.Wrap(err, "failed to read metadata response")
	}
	if int64(len(payload)) > maxBytes {
		return errors.Errorf(
			"metadata response is too large: %d bytes exceeds max %d bytes",
			len(payload),
			maxBytes,
		)
	}

	if err := json.Unmarshal(payload, out); err != nil {
		return errors.Wrap(err, "failed to decode metadata JSON")
	}
	return nil
}

func normalizeTokenizerTag(version string) (string, error) {
	version = strings.TrimSpace(version)
	switch {
	case version == "":
		return "", errors.New("tokenizers library version cannot be empty")
	case strings.EqualFold(version, tokenizerLatestTag):
		return tokenizerLatestTag, nil
	}

	semverPart := version
	switch {
	case strings.HasPrefix(version, "rust-v"):
		semverPart = strings.TrimPrefix(version, "rust-v")
	case strings.HasPrefix(version, "rust-"):
		semverPart = strings.TrimPrefix(version, "rust-")
	case strings.HasPrefix(version, "v"):
		semverPart = strings.TrimPrefix(version, "v")
	}
	semverPart = strings.TrimSpace(semverPart)
	if semverPart == "" {
		return "", errors.Errorf("tokenizers library version %q has empty semantic version suffix", version)
	}

	parsedVersion, err := semver.StrictNewVersion(semverPart)
	if err != nil {
		return "", errors.Errorf("tokenizers library version %q must be a valid semantic version", version)
	}

	normalizedVersion := "rust-v" + parsedVersion.String()
	if len(normalizedVersion) > tokenizerMaxVersionTagLength {
		return "", errors.Errorf(
			"tokenizers library version exceeds max length of %d characters",
			tokenizerMaxVersionTagLength,
		)
	}

	return normalizedVersion, nil
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
		checksum := strings.ToLower(fields[0])
		if !tokenizerLooksLikeSHA256(checksum) {
			return "", errors.Errorf("invalid checksum format for asset %s: %q", assetName, fields[0])
		}
		return checksum, nil
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read checksum file")
	}
	return "", errors.Errorf("checksum entry not found for asset %s", assetName)
}

func tokenizerLooksLikeSHA256(v string) bool {
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

func tokenizerDownloadArtifactFileWithRetry(filePath, url string) error {
	return errors.WithStack(downloadutil.DownloadFileWithRetry(
		filePath,
		url,
		tokenizerDownloadAttempts,
		downloadutil.Config{
			MaxBytes:  tokenizerMaxArtifactBytes,
			DirPerm:   tokenizerCacheDirPerm,
			Accept:    "*/*",
			UserAgent: tokenizerDownloaderUserAgent,
		},
	))
}

func tokenizerDownloadMetadataFileWithRetry(filePath, url string) error {
	return errors.WithStack(downloadutil.DownloadFileWithRetry(
		filePath,
		url,
		tokenizerDownloadAttempts,
		downloadutil.Config{
			MaxBytes:  tokenizerMaxMetadataBytes,
			DirPerm:   tokenizerCacheDirPerm,
			Accept:    "text/plain,application/json;q=0.9,*/*;q=0.8",
			UserAgent: tokenizerDownloaderUserAgent,
		},
	))
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
