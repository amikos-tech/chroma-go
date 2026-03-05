package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "adds-v-prefix", input: "1.2.3", want: "v1.2.3"},
		{name: "keeps-v-prefix", input: "v1.2.3", want: "v1.2.3"},
		{name: "rejects-bare-v", input: "v", wantErr: true},
		{name: "rejects-devel", input: "(devel)", wantErr: true},
		{name: "rejects-invalid-char", input: "v1.2.3/abc", wantErr: true},
		{name: "rejects-empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeTag(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTokenizersVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain-semver", input: "0.1.5", want: "rust-v0.1.5"},
		{name: "v-prefixed", input: "v0.1.5", want: "rust-v0.1.5"},
		{name: "rust-prefixed", input: "rust-0.1.5", want: "rust-v0.1.5"},
		{name: "rust-v-prefixed", input: "rust-v0.1.5", want: "rust-v0.1.5"},
		{name: "invalid", input: "not-a-version", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeTokenizersVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeOnnxRuntimeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid-semver", input: "1.23.1", want: "1.23.1"},
		{name: "valid-with-suffix", input: "1.23.1-rc1", want: "1.23.1-rc1"},
		{name: "rejects-empty", input: "", wantErr: true},
		{name: "rejects-devel", input: "(devel)", wantErr: true},
		{name: "rejects-shell-payload", input: "1.23.1;echo pwned", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeOnnxRuntimeVersion(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeHTTPS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "keeps-https", input: "https://example.com/path", want: "https://example.com/path"},
		{name: "trims-trailing-slash", input: "https://example.com/path/", want: "https://example.com/path"},
		{name: "rejects-http", input: "http://example.com/path", wantErr: true},
		{name: "rejects-relative", input: "/path", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeHTTPS(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeChecksumAssetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "*libtokenizers-aarch64-apple-darwin.tar.gz", want: "libtokenizers-aarch64-apple-darwin.tar.gz"},
		{input: "folder\\SHA256SUMS", want: "SHA256SUMS"},
		{input: "./foo/bar.txt", want: "bar.txt"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := normalizeChecksumAssetName(tt.input)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChecksumFromSumsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	payload := "" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  foo.txt\n" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  bar.txt\n"
	if err := os.WriteFile(sumsPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write sums file: %v", err)
	}

	candidates := map[string]struct{}{
		"bar.txt": {},
	}
	name, sum, err := checksumFromSumsFile(sumsPath, candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "bar.txt" {
		t.Fatalf("got name %q, want bar.txt", name)
	}
	if sum != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("got sum %q", sum)
	}
}

func TestChecksumFromSumsFileMissingCandidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	payload := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  foo.txt\n"
	if err := os.WriteFile(sumsPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write sums file: %v", err)
	}

	candidates := map[string]struct{}{
		"bar.txt": {},
	}
	_, _, err := checksumFromSumsFile(sumsPath, candidates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChecksumFromSumsFileMalformedCandidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	payload := "not-a-checksum  bar.txt\n"
	if err := os.WriteFile(sumsPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write sums file: %v", err)
	}

	candidates := map[string]struct{}{
		"bar.txt": {},
	}
	_, _, err := checksumFromSumsFile(sumsPath, candidates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVerifySHA256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "payload.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	if err := verifySHA256(filePath, "2CF24DBA5FB0A30E26E83B2AC5B9E29E1B161E5C1FA7425E73043362938B9824"); err != nil {
		t.Fatalf("expected checksum match: %v", err)
	}
	if err := verifySHA256(filePath, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
}

func TestOnnxRuntimeLibPredicateLinuxVersioned(t *testing.T) {
	t.Parallel()

	predicate := onnxRuntimeLibPredicate("linux")
	if !predicate("libonnxruntime.so.1.23.1") {
		t.Fatal("expected versioned .so filename to match")
	}
	if !predicate("libonnxruntime.so") {
		t.Fatal("expected bare .so filename to match")
	}
	if predicate("libonnxruntime.dylib") {
		t.Fatal("did not expect .dylib filename to match linux predicate")
	}
}

func TestIsSafeRuntimeLibraryFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "safe-versioned-so", input: "libonnxruntime.so.1.23.1", want: true},
		{name: "safe-dylib", input: "libonnxruntime.1.23.1.dylib", want: true},
		{name: "safe-dll", input: "onnxruntime.dll", want: true},
		{name: "rejects-shell", input: "libonnxruntime.so.1$(whoami)", want: false},
		{name: "rejects-space", input: "libonnxruntime so", want: false},
		{name: "rejects-empty", input: "   ", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isSafeRuntimeLibraryFilename(tt.input)
			if got != tt.want {
				t.Fatalf("input=%q got=%v want=%v", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyGitHubAuthHeader(t *testing.T) {
	t.Run("github-host-uses-github-token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "token-a")
		t.Setenv("GH_TOKEN", "")
		req, err := http.NewRequest(http.MethodGet, "https://github.com/amikos-tech/chroma-go", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		applyGitHubAuthHeader(req)
		if got := req.Header.Get("Authorization"); got != "Bearer token-a" {
			t.Fatalf("got %q, want %q", got, "Bearer token-a")
		}
	})

	t.Run("github-host-falls-back-to-gh-token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "token-b")
		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/amikos-tech/chroma-go", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		applyGitHubAuthHeader(req)
		if got := req.Header.Get("Authorization"); got != "Bearer token-b" {
			t.Fatalf("got %q, want %q", got, "Bearer token-b")
		}
	})

	t.Run("non-github-host-no-auth-header", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "token-a")
		req, err := http.NewRequest(http.MethodGet, "https://releases.amikos.tech/chroma-go-local/v0.3.3/SHA256SUMS", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		applyGitHubAuthHeader(req)
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("expected empty authorization header, got %q", got)
		}
	})
}

func TestPowerShellSingleQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no-quote", input: "abc", want: "abc"},
		{name: "with-quote", input: "ab'cd", want: "ab''cd"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := powershellSingleQuote(tt.input)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTarMember(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "artifact.tar.gz")
	createTarGz(t, archivePath, map[string][]byte{
		"nested/libtokenizers.so": []byte("tokenizer-payload"),
	})

	outPath := filepath.Join(dir, "out", "libtokenizers.so")
	if err := extractTarMember(archivePath, "libtokenizers.so", outPath); err != nil {
		t.Fatalf("extractTarMember returned error: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if !bytes.Equal(got, []byte("tokenizer-payload")) {
		t.Fatalf("extracted payload mismatch: got=%q", string(got))
	}
}

func TestExtractTarMemberMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "artifact.tar.gz")
	createTarGz(t, archivePath, map[string][]byte{
		"nested/libtokenizers.so": []byte("tokenizer-payload"),
	})

	outPath := filepath.Join(dir, "out", "missing.so")
	if err := extractTarMember(archivePath, "missing.so", outPath); err == nil {
		t.Fatal("expected error for missing member, got nil")
	}
}

func TestExtractTarMemberRejectsSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "artifact.tar.gz")
	createTarGzWithSymlink(t, archivePath, "nested/libtokenizers.so", "/tmp/target")

	outPath := filepath.Join(dir, "out", "libtokenizers.so")
	if err := extractTarMember(archivePath, "libtokenizers.so", outPath); err == nil {
		t.Fatal("expected error for symlink member, got nil")
	}
}

func TestCopyDirRejectsSymlink(t *testing.T) {
	t.Parallel()

	src := t.TempDir()
	dst := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "real.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(src, "link.txt")); err != nil {
		t.Fatal(err)
	}

	err := copyDir(src, filepath.Join(dst, "out"))
	if err == nil {
		t.Fatal("expected error for symlink in source directory, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got: %v", err)
	}
}

func createTarGz(t *testing.T, archivePath string, files map[string][]byte) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		content := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write header %s: %v", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("write content %s: %v", name, err)
		}
	}
}

func createTarGzWithSymlink(t *testing.T, archivePath, name, target string) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	header := &tar.Header{
		Name:     name,
		Mode:     0o777,
		Typeflag: tar.TypeSymlink,
		Linkname: target,
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write symlink header: %v", err)
	}
}
