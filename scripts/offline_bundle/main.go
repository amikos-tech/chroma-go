package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/amikos-tech/pure-onnx/ort"

	"github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef" //nolint:staticcheck
)

const (
	defaultLocalShimVersion   = "v0.3.3"
	defaultTokenizersVersion  = "v0.1.5"
	defaultOnnxRuntimeVersion = "1.23.1"
	maxMetadataBytes          = 5 * 1024 * 1024
	maxArtifactBytes          = 500 * 1024 * 1024
	downloadTimeout           = 5 * time.Minute

	localLibraryBaseURL         = "https://releases.amikos.tech/chroma-go-local"
	localLibraryFallbackBaseURL = "https://github.com/amikos-tech/chroma-go-local/releases/download"
	localArchivePrefix          = "chroma-go-local"
	localArchivePrefixAlt       = "local-chroma"

	tokenizersBaseURL         = "https://releases.amikos.tech/pure-tokenizers"
	tokenizersFallbackBaseURL = "https://github.com/amikos-tech/pure-tokenizers/releases/download"
	tokenizersReleasesAPI     = "https://api.github.com/repos/amikos-tech/pure-tokenizers/releases?per_page=100&page=1"

	checksumsAsset   = "SHA256SUMS"
	onnxModelTag     = "all-MiniLM-L6-v2"
	onnxBundleSubdir = "onnx-runtime"
	onnxModelSubdir  = "onnx-models"
	tokenizersSubdir = "tokenizers"
	localShimSubdir  = "local-shim"
	offlineEnvFile   = "offline.env"
	offlineSetupSh   = "offline-setup.sh"
	offlineSetupPs1  = "offline-setup.ps1"
	manifestFile     = "manifest.json"
	checksumsFile    = "checksums.sha256"
)

type artifactInfo struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
	Size int64  `json:"size"`
	Kind string `json:"kind"`
}

type offlineManifest struct {
	GeneratedAt string `json:"generated_at"`
	Platform    struct {
		GOOS   string `json:"goos"`
		GOARCH string `json:"goarch"`
	} `json:"platform"`
	Versions struct {
		LocalShim    string `json:"local_shim"`
		Tokenizers   string `json:"tokenizers"`
		OnnxRuntime  string `json:"onnx_runtime"`
		OnnxModelTag string `json:"onnx_model"`
	} `json:"versions"`
	Files []artifactInfo `json:"files"`
}

type offlineBundleConfig struct {
	goos               string
	goarch             string
	localShimVersion   string
	tokenizersVersion  string
	onnxRuntimeVersion string
	output             string
	force              bool
}

type localArtifact struct {
	platform    string
	libraryFile string
}

type tokenizerArtifact struct {
	platform    string
	archiveName string
	libraryFile string
}

func main() {
	goos := flag.String("goos", runtime.GOOS, "target GOOS (host platform only)")
	goarch := flag.String("goarch", runtime.GOARCH, "target GOARCH (host platform only)")
	localVersion := flag.String("local-shim-version", defaultLocalShimVersion, "local shim version")
	tokenizersVersion := flag.String("tokenizers-version", "", "TOKENIZERS_VERSION value")
	onnxRuntimeVersion := flag.String("onnx-runtime-version", "", "CHROMAGO_ONNX_RUNTIME_VERSION value")
	output := flag.String("output", "./offline-bundle", "output directory")
	force := flag.Bool("force", false, "overwrite existing output directory")
	flag.Parse()

	cfg := offlineBundleConfig{
		goos:               strings.TrimSpace(*goos),
		goarch:             strings.TrimSpace(*goarch),
		localShimVersion:   strings.TrimSpace(*localVersion),
		tokenizersVersion:  strings.TrimSpace(*tokenizersVersion),
		onnxRuntimeVersion: strings.TrimSpace(*onnxRuntimeVersion),
		output:             strings.TrimSpace(*output),
		force:              *force,
	}

	if cfg.localShimVersion == "" {
		cfg.localShimVersion = defaultLocalShimVersion
	}
	if cfg.tokenizersVersion == "" {
		cfg.tokenizersVersion = strings.TrimSpace(os.Getenv("TOKENIZERS_VERSION"))
	}
	if cfg.tokenizersVersion == "" {
		cfg.tokenizersVersion = defaultTokenizersVersion
	}
	if cfg.onnxRuntimeVersion == "" {
		cfg.onnxRuntimeVersion = strings.TrimSpace(os.Getenv("CHROMAGO_ONNX_RUNTIME_VERSION"))
	}
	if cfg.onnxRuntimeVersion == "" {
		cfg.onnxRuntimeVersion = defaultOnnxRuntimeVersion
	}

	if err := createOfflineBundle(cfg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to generate offline bundle: %v\n", err)
		os.Exit(1)
	}
}

func createOfflineBundle(cfg offlineBundleConfig) error {
	fmt.Fprintln(os.Stderr, "Preparing offline dependency bundle...")
	if cfg.goos == "" {
		return errors.New("goos cannot be empty")
	}
	if cfg.goarch == "" {
		return errors.New("goarch cannot be empty")
	}
	if cfg.output == "" {
		return errors.New("output cannot be empty")
	}
	// This tool bundles host-native runtime artifacts (native shared libraries + model cache).
	if cfg.goos != runtime.GOOS || cfg.goarch != runtime.GOARCH {
		return fmt.Errorf("cross-platform bundles are not supported yet; got %s/%s, host platform is %s/%s", cfg.goos, cfg.goarch, runtime.GOOS, runtime.GOARCH)
	}

	localAsset, err := localRuntimeArtifact(cfg.goos, cfg.goarch)
	if err != nil {
		return err
	}
	tokenizerAsset, err := tokenizerRuntimeArtifact(cfg.goos, cfg.goarch)
	if err != nil {
		return err
	}

	localVersion, err := normalizeTag(cfg.localShimVersion)
	if err != nil {
		return err
	}
	tokenizersVersion, err := normalizeTokenizersVersion(cfg.tokenizersVersion)
	if err != nil {
		return err
	}
	onnxRuntimeVersion, err := normalizeOnnxRuntimeVersion(cfg.onnxRuntimeVersion)
	if err != nil {
		return err
	}

	absOutput, err := filepath.Abs(cfg.output)
	if err != nil {
		return err
	}

	if err := prepareOutput(absOutput, cfg.force); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Output directory ready: %s\n", absOutput)

	workDir := filepath.Join(absOutput, ".offline-work")
	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("failed to clean work dir %s: %w", workDir, err)
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	artifacts := make([]artifactInfo, 0, 32)
	fmt.Fprintf(os.Stderr, "Downloading local shim artifact (%s)...\n", localVersion)

	localOutDir := filepath.Join(absOutput, localShimSubdir, localAsset.platform)
	localOut := filepath.Join(localOutDir, localAsset.libraryFile)
	if err := downloadAndExtractLocalBundle(workDir, localOut, localVersion, localAsset); err != nil {
		return err
	}
	if err := addArtifact(absOutput, localOut, "local-shim", &artifacts); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Downloading tokenizers artifact (%s)...\n", tokenizersVersion)

	tokenizerOutDir := filepath.Join(absOutput, tokenizersSubdir, tokenizerAsset.platform)
	tokenizerOut := filepath.Join(tokenizerOutDir, tokenizerAsset.libraryFile)
	if err := downloadAndExtractTokenizerBundle(workDir, tokenizerOut, tokenizersVersion, tokenizerAsset); err != nil {
		return err
	}
	if err := addArtifact(absOutput, tokenizerOut, "tokenizers", &artifacts); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Downloading default ONNX model...")

	if err := bundleDefaultModel(workDir, absOutput, tokenizersVersion); err != nil {
		return err
	}
	modelDir := filepath.Join(absOutput, onnxModelSubdir, onnxModelTag, "onnx")
	if err := addDirArtifacts(absOutput, modelDir, "onnx-model", &artifacts); err != nil {
		return err
	}

	localRel := filepath.ToSlash(filepath.Join(localShimSubdir, localAsset.platform, filepath.Base(localOut)))
	tokenizerRel := filepath.ToSlash(filepath.Join(tokenizersSubdir, tokenizerAsset.platform, filepath.Base(tokenizerOut)))
	fmt.Fprintln(os.Stderr, "Resolving ONNX runtime library...")
	localOnnxRuntimeRel, err := ensureOnnxRuntimeArtifact(workDir, absOutput, onnxRuntimeVersion, cfg.goos, cfg.goarch, &artifacts)
	if err != nil {
		return err
	}
	onnxRel := filepath.ToSlash(filepath.Join(onnxBundleSubdir, cfg.goos+"-"+cfg.goarch))
	if localOnnxRuntimeRel != "" {
		onnxRel = filepath.ToSlash(filepath.Join(onnxRel, filepath.Base(localOnnxRuntimeRel)))
	}

	if err := createOfflineEnvFile(absOutput, localRel, tokenizerRel, onnxRel, tokenizersVersion, onnxRuntimeVersion); err != nil {
		return err
	}
	if err := addArtifact(absOutput, filepath.Join(absOutput, offlineEnvFile), "offline-config", &artifacts); err != nil {
		return err
	}

	if err := writeOfflineSetupShell(filepath.Join(absOutput, offlineSetupSh), localRel, tokenizerRel, onnxRel, tokenizersVersion, onnxRuntimeVersion); err != nil {
		return err
	}
	if err := addArtifact(absOutput, filepath.Join(absOutput, offlineSetupSh), "offline-setup", &artifacts); err != nil {
		return err
	}

	if err := writeOfflineSetupPowershell(
		filepath.Join(absOutput, offlineSetupPs1),
		localRel,
		tokenizerRel,
		onnxRel,
		tokenizersVersion,
		onnxRuntimeVersion,
	); err != nil {
		return err
	}
	if err := addArtifact(absOutput, filepath.Join(absOutput, offlineSetupPs1), "offline-setup", &artifacts); err != nil {
		return err
	}

	manifest := offlineManifest{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Platform: struct {
			GOOS   string `json:"goos"`
			GOARCH string `json:"goarch"`
		}{
			GOOS:   cfg.goos,
			GOARCH: cfg.goarch,
		},
		Versions: struct {
			LocalShim    string `json:"local_shim"`
			Tokenizers   string `json:"tokenizers"`
			OnnxRuntime  string `json:"onnx_runtime"`
			OnnxModelTag string `json:"onnx_model"`
		}{
			LocalShim:    localVersion,
			Tokenizers:   tokenizersVersion,
			OnnxRuntime:  onnxRuntimeVersion,
			OnnxModelTag: onnxModelTag,
		},
	}
	if err := writeManifestAndChecksums(absOutput, manifest, artifacts); err != nil {
		return err
	}

	fmt.Printf("Generated offline bundle at %s\n", absOutput)
	return nil
}

func ensureOnnxRuntimeArtifact(workDir, output, version, goos, goarch string, artifacts *[]artifactInfo) (string, error) {
	restoreOnnxVersion, err := withEnv("CHROMAGO_ONNX_RUNTIME_VERSION", version)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = restoreOnnxVersion()
	}()

	cacheDir := filepath.Join(workDir, "onnx-runtime-cache")
	fmt.Fprintf(os.Stderr, "Ensuring ONNX runtime %s in cache %s...\n", version, cacheDir)
	if _, err := ort.EnsureOnnxRuntimeSharedLibrary(
		ort.WithBootstrapCacheDir(cacheDir),
		ort.WithBootstrapVersion(version),
	); err != nil {
		return "", err
	}
	runtimePath, err := findOnnxRuntimeLibrary(cacheDir, goos)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "Resolved ONNX runtime artifact at %s\n", runtimePath)
	dstDir := filepath.Join(output, onnxBundleSubdir, goos+"-"+goarch)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return "", err
	}
	dstPath := filepath.Join(dstDir, filepath.Base(runtimePath))
	if err := copyFile(runtimePath, dstPath); err != nil {
		return "", err
	}
	if err := addArtifact(output, dstPath, "onnx-runtime", artifacts); err != nil {
		return "", err
	}

	return filepath.ToSlash(filepath.Join(onnxBundleSubdir, goos+"-"+goarch, filepath.Base(dstPath))), nil
}

func bundleDefaultModel(workDir, output, tokenizersVersion string) error {
	fmt.Fprintln(os.Stderr, "Preparing to bundle default ONNX model...")
	onnxHome := filepath.Join(workDir, "onnx-home")
	if err := os.MkdirAll(onnxHome, 0o700); err != nil {
		return err
	}

	restoreHome, err := withEnv("HOME", onnxHome)
	if err != nil {
		return err
	}
	restoreProfile, err := withEnv("USERPROFILE", onnxHome)
	if err != nil {
		_ = restoreHome()
		return err
	}
	restoreTokenizersVersion, err := withEnv("TOKENIZERS_VERSION", tokenizersVersion)
	if err != nil {
		_ = restoreProfile()
		_ = restoreHome()
		return err
	}
	defer func() {
		_ = restoreTokenizersVersion()
		_ = restoreProfile()
		_ = restoreHome()
	}()

	if err := defaultef.EnsureDefaultEmbeddingFunctionModel(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Default ONNX model bootstrap complete in %s/.cache/chroma/onnx_models/%s/onnx\n", onnxHome, onnxModelTag)

	sourceModel := filepath.Join(onnxHome, ".cache", "chroma", "onnx_models", onnxModelTag, "onnx")
	targetModel := filepath.Join(output, onnxModelSubdir, onnxModelTag, "onnx")
	return copyDir(sourceModel, targetModel)
}

func normalizeTag(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("version cannot be empty")
	}
	if strings.EqualFold(raw, "(devel)") {
		return "", errors.New("invalid version: (devel)")
	}
	if !strings.HasPrefix(raw, "v") {
		raw = "v" + raw
	}
	if raw == "v" {
		return "", errors.New("version cannot be bare 'v'")
	}
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return "", fmt.Errorf("invalid version: %q", raw)
		}
	}
	return raw, nil
}

func normalizeTokenizersVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("tokenizers version cannot be empty")
	}
	if strings.EqualFold(raw, "latest") {
		latest, err := resolveLatestTokenizersVersion()
		if err != nil {
			return "", err
		}
		return latest, nil
	}

	semverPart := strings.TrimPrefix(raw, "rust-")
	semverPart = strings.TrimPrefix(semverPart, "v")
	parsed, err := semver.StrictNewVersion(semverPart)
	if err != nil {
		return "", err
	}
	return "rust-v" + parsed.String(), nil
}

func normalizeOnnxRuntimeVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("onnx runtime version cannot be empty")
	}
	if strings.EqualFold(raw, "(devel)") {
		return "", errors.New("invalid onnx runtime version: (devel)")
	}
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return "", fmt.Errorf("invalid onnx runtime version: %q", raw)
		}
	}
	return raw, nil
}

func resolveLatestTokenizersVersion() (string, error) {
	client := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequest(http.MethodGet, tokenizersReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "chroma-go-offline-bundle")
	req.Header.Set("Accept", "application/json")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tokenizers release API returned %s", resp.Status)
	}

	type rel struct {
		TagName string `json:"tag_name"`
	}

	var releases []rel
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes))
	if err := decoder.Decode(&releases); err != nil {
		return "", err
	}
	for _, item := range releases {
		tag := strings.TrimSpace(item.TagName)
		if strings.HasPrefix(tag, "rust-v") {
			if _, err := semver.StrictNewVersion(strings.TrimPrefix(tag, "rust-v")); err != nil {
				continue
			}
			return tag, nil
		}
	}
	return "", errors.New("unable to resolve latest tokenizers version")
}

func prepareOutput(path string, force bool) error {
	if !force {
		if entries, err := os.ReadDir(path); err == nil {
			if len(entries) > 0 {
				return fmt.Errorf("output directory is not empty: %s", path)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func localRuntimeArtifact(goos, goarch string) (localArtifact, error) {
	switch goos {
	case "linux":
		if goarch != "amd64" {
			return localArtifact{}, fmt.Errorf("unsupported architecture for local shim: %s", goarch)
		}
		return localArtifact{platform: "linux-amd64", libraryFile: "libchroma_shim.so"}, nil
	case "darwin":
		if goarch != "arm64" {
			return localArtifact{}, fmt.Errorf("unsupported architecture for local shim: %s", goarch)
		}
		return localArtifact{platform: "darwin-arm64", libraryFile: "libchroma_shim.dylib"}, nil
	case "windows":
		if goarch != "amd64" {
			return localArtifact{}, fmt.Errorf("unsupported architecture for local shim: %s", goarch)
		}
		return localArtifact{platform: "windows-amd64", libraryFile: "chroma_shim.dll"}, nil
	default:
		return localArtifact{}, fmt.Errorf("unsupported os for local shim: %s", goos)
	}
}

func tokenizerRuntimeArtifact(goos, goarch string) (tokenizerArtifact, error) {
	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return tokenizerArtifact{}, fmt.Errorf("unsupported architecture for tokenizers: %s", goarch)
	}

	var platform, triple, lib string
	switch goos {
	case "darwin":
		platform = "darwin-" + goarch
		triple = "apple-darwin"
		lib = "libtokenizers.dylib"
	case "linux":
		platform = "linux-" + goarch
		musl, err := isMuslLinux()
		if err != nil {
			return tokenizerArtifact{}, err
		}
		if musl {
			platform += "-musl"
			triple = "unknown-linux-musl"
		} else {
			triple = "unknown-linux-gnu"
		}
		lib = "libtokenizers.so"
	case "windows":
		platform = "windows-" + goarch
		triple = "pc-windows-msvc"
		lib = "tokenizers.dll"
	default:
		return tokenizerArtifact{}, fmt.Errorf("unsupported os for tokenizers: %s", goos)
	}

	return tokenizerArtifact{
		platform:    platform,
		archiveName: fmt.Sprintf("libtokenizers-%s-%s.tar.gz", arch, triple),
		libraryFile: lib,
	}, nil
}

func isMuslLinux() (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed checking /etc/alpine-release: %w", err)
	}

	lddPaths := []string{"/usr/bin/ldd", "/bin/ldd"}
	var f *os.File
	var openErr error
	for _, p := range lddPaths {
		f, openErr = os.Open(p)
		if openErr == nil {
			break
		}
		if os.IsNotExist(openErr) {
			continue
		}
		return false, fmt.Errorf("failed opening %s: %w", p, openErr)
	}
	if f == nil {
		return false, fmt.Errorf("failed to detect musl: none of %v is available", lddPaths)
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, 128*1024))
	if err != nil {
		return false, fmt.Errorf("failed reading ldd contents: %w", err)
	}
	return strings.Contains(strings.ToLower(string(content)), "musl"), nil
}

func downloadAndExtractLocalBundle(workDir, output string, version string, artifact localArtifact) error {
	fmt.Fprintf(os.Stderr, "Resolving local shim archive for %s from configured checksums...\n", artifact.platform)
	base, archive, checksum, err := resolveArtifactFromChecksums(
		workDir,
		version,
		[]string{localLibraryBaseURL, localLibraryFallbackBaseURL},
		[]string{
			fmt.Sprintf("%s-%s-%s.tar.gz", localArchivePrefix, version, artifact.platform),
			fmt.Sprintf("%s-%s-%s.tar.gz", localArchivePrefixAlt, version, artifact.platform),
		},
	)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(workDir, archive)
	fmt.Fprintf(os.Stderr, "Downloading local shim archive %s\n", archive)
	if err := downloadFile(archivePath, buildArtifactURL(base, version, archive), maxArtifactBytes); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Verifying local shim checksum for %s\n", archive)
	if err := verifySHA256(archivePath, checksum); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return err
	}
	return extractTarMember(archivePath, artifact.libraryFile, output)
}

func downloadAndExtractTokenizerBundle(workDir, output, version string, artifact tokenizerArtifact) error {
	fmt.Fprintf(os.Stderr, "Resolving tokenizers archive for %s from configured checksums...\n", artifact.platform)
	base, archive, checksum, err := resolveArtifactFromChecksums(
		workDir,
		version,
		[]string{tokenizersBaseURL, tokenizersFallbackBaseURL},
		[]string{artifact.archiveName},
	)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(workDir, archive)
	fmt.Fprintf(os.Stderr, "Downloading tokenizers archive %s\n", archive)
	if err := downloadFile(archivePath, buildArtifactURL(base, version, archive), maxArtifactBytes); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Verifying tokenizers checksum for %s\n", archive)
	if err := verifySHA256(archivePath, checksum); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return err
	}
	return extractTarMember(archivePath, artifact.libraryFile, output)
}

func resolveArtifactFromChecksums(workDir, version string, bases []string, candidates []string) (selectedBase, archiveName, checksum string, err error) {
	if version == "" {
		return "", "", "", errors.New("version cannot be empty")
	}
	if len(candidates) == 0 {
		return "", "", "", errors.New("artifact candidates cannot be empty")
	}

	candidateMap := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		normalized := normalizeChecksumAssetName(candidate)
		if normalized != "" {
			candidateMap[normalized] = struct{}{}
		}
	}
	if len(candidateMap) == 0 {
		return "", "", "", errors.New("artifact candidates cannot be empty")
	}

	attemptErrors := make([]error, 0, len(bases))
	for _, base := range bases {
		base = strings.TrimSpace(strings.TrimRight(base, "/"))
		if base == "" {
			continue
		}
		base, normalizeErr := normalizeHTTPS(base)
		if normalizeErr != nil {
			attemptErrors = append(attemptErrors, fmt.Errorf("invalid checksum base URL %q: %w", base, normalizeErr))
			continue
		}
		fmt.Fprintf(os.Stderr, "Checking checksum index at %s for version %s\n", base, version)
		sumsURL := buildArtifactURL(base, version, checksumsAsset)
		sumsPath := filepath.Join(workDir, checksumsAsset)
		if downloadErr := downloadFile(sumsPath, sumsURL, maxMetadataBytes); downloadErr != nil {
			attemptErrors = append(attemptErrors, fmt.Errorf("failed downloading %s from %s: %w", checksumsAsset, base, downloadErr))
			continue
		}
		name, fileChecksum, parseErr := checksumFromSumsFile(sumsPath, candidateMap)
		if parseErr != nil {
			attemptErrors = append(attemptErrors, fmt.Errorf("failed parsing %s from %s: %w", checksumsAsset, base, parseErr))
			continue
		}
		return base, name, fileChecksum, nil
	}
	if len(attemptErrors) == 0 {
		return "", "", "", errors.New("failed to resolve artifact checksums")
	}
	return "", "", "", fmt.Errorf("failed to resolve artifact checksums: %w", errors.Join(attemptErrors...))
}

func buildArtifactURL(base, version, asset string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(base, "/"), version, asset)
}

func normalizeHTTPS(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() {
		return "", errors.New("base url must be absolute")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", errors.New("base url must use https")
	}
	if parsed.Host == "" {
		return "", errors.New("base url missing host")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func checksumFromSumsFile(path string, candidates map[string]struct{}) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := normalizeChecksumAssetName(parts[1])
		if name == "" {
			continue
		}
		if _, ok := candidates[name]; !ok {
			continue
		}
		checksum := strings.ToLower(strings.TrimSpace(parts[0]))
		if len(checksum) != 64 {
			continue
		}
		if _, err := hex.DecodeString(checksum); err != nil {
			continue
		}
		return name, checksum, nil
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return "", "", errors.New("checksum entry not found")
}

func normalizeChecksumAssetName(raw string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(raw), "*")
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = path.Base(normalized)
	if normalized == "." || normalized == "/" || normalized == ".." {
		return ""
	}
	return normalized
}

func downloadFile(dest, source string, maxBytes int64) error {
	fmt.Fprintf(os.Stderr, "Downloading %s\n", source)
	client := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "chroma-go-offline-bundle")
	applyGitHubAuthHeader(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed for %s: %s", source, resp.Status)
	}
	if maxBytes > 0 && resp.ContentLength > maxBytes {
		return fmt.Errorf("download too large: %d bytes", resp.ContentLength)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	reporter := &downloadReporter{
		label:         filepath.Base(dest),
		totalExpected: resp.ContentLength,
	}
	reader := io.Reader(resp.Body)
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes+1)
	}
	n, err := io.Copy(io.MultiWriter(tmp, reporter), reader)
	if closeErr := tmp.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if n > 0 {
		fmt.Fprintf(os.Stderr, "\r%s: %s complete\n", reporter.label, byteSize(n))
	}
	if err != nil {
		return err
	}
	if maxBytes > 0 && n > maxBytes {
		return fmt.Errorf("download too large for %s", source)
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return fmt.Errorf("download size mismatch for %s", source)
	}

	if err := os.Rename(tmp.Name(), dest); err != nil {
		if removeErr := os.Remove(dest); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if err := os.Rename(tmp.Name(), dest); err != nil {
			return err
		}
	}
	return nil
}

func applyGitHubAuthHeader(req *http.Request) {
	if req == nil || req.URL == nil {
		return
	}
	host := strings.ToLower(req.URL.Hostname())
	if host != "github.com" && host != "api.github.com" && host != "objects.githubusercontent.com" &&
		!strings.HasSuffix(host, ".github.com") && !strings.HasSuffix(host, ".githubusercontent.com") {
		return
	}
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GH_TOKEN"))
	}
	if token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

type downloadReporter struct {
	label         string
	total         int64
	totalExpected int64
	next          int64
}

func (dr *downloadReporter) Write(p []byte) (int, error) {
	n := len(p)
	if dr.label == "" {
		dr.label = "file"
	}
	dr.total += int64(n)
	if dr.next == 0 {
		dr.next = 4 * 1024 * 1024
	}
	for dr.total >= dr.next {
		if dr.totalExpected > 0 {
			percent := (float64(dr.total) / float64(dr.totalExpected)) * 100
			fmt.Fprintf(os.Stderr, "\r%s: %s (%0.1f%%)", dr.label, byteSize(dr.total), percent)
		} else {
			fmt.Fprintf(os.Stderr, "\r%s: %s", dr.label, byteSize(dr.total))
		}
		dr.next += 4 * 1024 * 1024
	}
	return n, nil
}

func byteSize(n int64) string {
	const (
		KB = 1024.0
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case n >= int64(GB):
		return fmt.Sprintf("%.1f GiB", float64(n)/GB)
	case n >= int64(MB):
		return fmt.Sprintf("%.1f MiB", float64(n)/MB)
	case n >= int64(KB):
		return fmt.Sprintf("%.1f KiB", float64(n)/KB)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func verifySHA256(path, expected string) error {
	h, err := checksumFile(path)
	if err != nil {
		return err
	}
	actual := strings.TrimSpace(h)
	expected = strings.TrimSpace(expected)
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", path, expected, actual)
	}
	return nil
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractTarMember(archivePath, memberName, outPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	t := tar.NewReader(gz)
	for {
		hdr, err := t.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		if filepath.Base(hdr.Name) != memberName {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != 0 {
			return fmt.Errorf("tar member %q is not a regular file", memberName)
		}
		if hdr.Size < 0 || hdr.Size > maxArtifactBytes {
			return fmt.Errorf("tar member %q has invalid size %d", memberName, hdr.Size)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			return err
		}
		out, err := os.Create(outPath)
		if err != nil {
			return err
		}
		if _, err := io.CopyN(out, t, hdr.Size); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(outPath, 0o755); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("member %q not found in archive", memberName)
}

func findOnnxRuntimeLibrary(root, goos string) (string, error) {
	matches := make([]string, 0, 8)
	predicate := onnxRuntimeLibPredicate(goos)

	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", p, err)
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !predicate(name) {
			return nil
		}
		if !isSafeRuntimeLibraryFilename(name) {
			return fmt.Errorf("unsafe ONNX runtime library filename: %s", name)
		}
		matches = append(matches, p)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed scanning ONNX runtime cache %s: %w", root, err)
	}
	if len(matches) == 0 {
		return "", errors.New("onnxruntime library not found in cache")
	}
	// Prefer the lexicographically latest match, which corresponds to the newest versioned library filename.
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func onnxRuntimeLibPredicate(goos string) func(string) bool {
	switch goos {
	case "windows":
		return func(name string) bool {
			lname := strings.ToLower(name)
			return strings.HasSuffix(lname, ".dll") && strings.Contains(lname, "onnx")
		}
	case "darwin":
		return func(name string) bool {
			lname := strings.ToLower(name)
			return strings.HasPrefix(lname, "libonnxruntime") && strings.HasSuffix(lname, ".dylib")
		}
	default:
		return func(name string) bool {
			lname := strings.ToLower(name)
			return strings.HasPrefix(lname, "libonnxruntime") && strings.Contains(lname, ".so")
		}
	}
}

func isSafeRuntimeLibraryFilename(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		return errors.New("expected file, got directory")
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func addArtifact(root, path, kind string, list *[]artifactInfo) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	if info.Size() <= 0 {
		return fmt.Errorf("artifact is empty: %s", path)
	}
	h, err := checksumFile(path)
	if err != nil {
		return err
	}
	*list = append(*list, artifactInfo{
		Path: filepath.ToSlash(filepath.Clean(rel)),
		SHA:  h,
		Size: info.Size(),
		Kind: kind,
	})
	return nil
}

func addDirArtifacts(root, dir, kind string, list *[]artifactInfo) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("artifact directory is not a directory: %s", dir)
	}
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return addArtifact(root, p, kind, list)
	})
}

func createOfflineEnvFile(root, localRel, tokenizerRel, onnxRuntimeRel, tokenizersVersion, onnxRuntimeVersion string) error {
	payload := []byte(fmt.Sprintf(
		"CHROMA_OFFLINE_BUNDLE_HOME=%s\nCHROMA_LIB_PATH=%s\nTOKENIZERS_LIB_PATH=%s\nCHROMAGO_ONNX_RUNTIME_PATH=%s\nTOKENIZERS_VERSION=%s\nCHROMAGO_ONNX_RUNTIME_VERSION=%s\n",
		shellSingleQuote(filepath.ToSlash(filepath.Clean(root))),
		shellSingleQuote(filepath.ToSlash(filepath.Join(filepath.ToSlash(root), localRel))),
		shellSingleQuote(filepath.ToSlash(filepath.Join(filepath.ToSlash(root), tokenizerRel))),
		shellSingleQuote(filepath.ToSlash(filepath.Join(filepath.ToSlash(root), onnxRuntimeRel))),
		shellSingleQuote(tokenizersVersion),
		shellSingleQuote(onnxRuntimeVersion),
	))
	return os.WriteFile(filepath.Join(root, offlineEnvFile), payload, 0o644)
}

func writeOfflineSetupShell(path, localRel, tokenizerRel, onnxRuntimeRel, tokenizersVersion, onnxRuntimeVersion string) error {
	contents := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

BUNDLE_DIR="$(cd "$(dirname "$0")" && pwd)"
DEFAULT_CHROMA_LIB_PATH="$BUNDLE_DIR/%s"
DEFAULT_TOKENIZERS_LIB_PATH="$BUNDLE_DIR/%s"
DEFAULT_ONNX_RUNTIME_PATH="$BUNDLE_DIR/%s"
DEFAULT_TOKENIZERS_VERSION=%s
DEFAULT_ONNX_RUNTIME_VERSION=%s

export CHROMA_OFFLINE_BUNDLE_HOME="${CHROMA_OFFLINE_BUNDLE_HOME:-$BUNDLE_DIR}"
export CHROMA_LIB_PATH="${CHROMA_LIB_PATH:-$DEFAULT_CHROMA_LIB_PATH}"
export TOKENIZERS_LIB_PATH="${TOKENIZERS_LIB_PATH:-$DEFAULT_TOKENIZERS_LIB_PATH}"
export CHROMAGO_ONNX_RUNTIME_PATH="${CHROMAGO_ONNX_RUNTIME_PATH:-$DEFAULT_ONNX_RUNTIME_PATH}"

# Prefer these values if not already overridden.
export TOKENIZERS_VERSION="${TOKENIZERS_VERSION:-$DEFAULT_TOKENIZERS_VERSION}"
export CHROMAGO_ONNX_RUNTIME_VERSION="${CHROMAGO_ONNX_RUNTIME_VERSION:-$DEFAULT_ONNX_RUNTIME_VERSION}"

MODEL_SOURCE="$BUNDLE_DIR/onnx-models/%s/onnx"
MODEL_TARGET="${HOME:-$BUNDLE_DIR}/.cache/chroma/onnx_models/%s/onnx"
if [ -d "$MODEL_SOURCE" ]; then
  mkdir -p "$MODEL_TARGET"
  cp -R "$MODEL_SOURCE"/. "$MODEL_TARGET"/
else
  echo "Warning: model source missing at $MODEL_SOURCE" >&2
fi

echo "Loaded offline runtime environment from $BUNDLE_DIR"
`, localRel, tokenizerRel, onnxRuntimeRel, shellSingleQuote(tokenizersVersion), shellSingleQuote(onnxRuntimeVersion), onnxModelTag, onnxModelTag)
	return os.WriteFile(path, []byte(contents), 0o755)
}

func writeOfflineSetupPowershell(path, localRel, tokenizerRel, onnxRuntimeRel, tokenizersVersion, onnxRuntimeVersion string) error {
	contents := fmt.Sprintf(`$bundleRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$env:CHROMA_OFFLINE_BUNDLE_HOME = $bundleRoot
$env:CHROMA_LIB_PATH = if ([string]::IsNullOrWhiteSpace($env:CHROMA_LIB_PATH)) { Join-Path $bundleRoot '%s' } else { $env:CHROMA_LIB_PATH }
$env:TOKENIZERS_LIB_PATH = if ([string]::IsNullOrWhiteSpace($env:TOKENIZERS_LIB_PATH)) { Join-Path $bundleRoot '%s' } else { $env:TOKENIZERS_LIB_PATH }
$env:CHROMAGO_ONNX_RUNTIME_PATH = if ([string]::IsNullOrWhiteSpace($env:CHROMAGO_ONNX_RUNTIME_PATH)) { Join-Path $bundleRoot '%s' } else { $env:CHROMAGO_ONNX_RUNTIME_PATH }
if ([string]::IsNullOrWhiteSpace($env:TOKENIZERS_VERSION)) {
  $env:TOKENIZERS_VERSION = '%s'
}
if ([string]::IsNullOrWhiteSpace($env:CHROMAGO_ONNX_RUNTIME_VERSION)) {
  $env:CHROMAGO_ONNX_RUNTIME_VERSION = '%s'
}
$homeForCache = $env:USERPROFILE
if ([string]::IsNullOrWhiteSpace($homeForCache)) {
  $homeForCache = $env:HOME
}
$modelSource = Join-Path $bundleRoot 'onnx-models/%s/onnx'
$modelTarget = Join-Path $homeForCache '.cache/chroma/onnx_models/%s/onnx'
if (Test-Path $modelSource) {
  New-Item -ItemType Directory -Force -Path $modelTarget | Out-Null
  Copy-Item -Path (Join-Path $modelSource '*') -Destination $modelTarget -Recurse -Force
} else {
  Write-Warning "Model source missing at $modelSource"
}
`, powershellSingleQuote(localRel), powershellSingleQuote(tokenizerRel), powershellSingleQuote(onnxRuntimeRel), powershellSingleQuote(tokenizersVersion), powershellSingleQuote(onnxRuntimeVersion), powershellSingleQuote(onnxModelTag), powershellSingleQuote(onnxModelTag))
	return os.WriteFile(path, []byte(contents), 0o755)
}

func withEnv(key, value string) (func() error, error) {
	previous, exists := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		return nil, err
	}
	return func() error {
		if exists {
			return os.Setenv(key, previous)
		}
		return os.Unsetenv(key)
	}, nil
}

func shellSingleQuote(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 8)
	b.WriteByte('\'')
	for _, r := range value {
		if r == '\'' {
			b.WriteString(`'"'"'`)
			continue
		}
		b.WriteRune(r)
	}
	b.WriteByte('\'')
	return b.String()
}

func powershellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func writeManifestAndChecksums(output string, manifest offlineManifest, artifacts []artifactInfo) error {
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Kind == artifacts[j].Kind {
			return artifacts[i].Path < artifacts[j].Path
		}
		return artifacts[i].Kind < artifacts[j].Kind
	})
	manifest.Files = artifacts

	manifestPath := filepath.Join(output, manifestFile)
	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, append(bytes, '\n'), 0o644); err != nil {
		return err
	}
	if err := addArtifact(output, manifestPath, "manifest", &artifacts); err != nil {
		return err
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Path == artifacts[j].Path {
			return artifacts[i].SHA < artifacts[j].SHA
		}
		return artifacts[i].Path < artifacts[j].Path
	})

	lines := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		lines = append(lines, fmt.Sprintf("%s  %s", artifact.SHA, artifact.Path))
	}
	return os.WriteFile(filepath.Join(output, checksumsFile), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
