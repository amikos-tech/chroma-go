#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

OUTPUT_DIR="${OFFLINE_RUNTIME_DEPS_DIR:-${REPO_ROOT}/artifacts/runtime-deps}"
GOOS="${OFFLINE_RUNTIME_GOOS:-$(go env GOOS)}"
GOARCH="${OFFLINE_RUNTIME_GOARCH:-$(go env GOARCH)}"
LOCAL_SHIM_VERSION="${CHROMA_LOCAL_SHIM_VERSION:-v0.3.3}"
TOKENIZERS_VERSION="${TOKENIZERS_VERSION:-v0.1.5}"
ONNX_RUNTIME_VERSION="${CHROMAGO_ONNX_RUNTIME_VERSION:-1.23.1}"

usage() {
	cat <<'EOF'
Usage:
  ./scripts/fetch_runtime_deps.sh [options]

Options:
  --output-dir DIR              Download artifacts into DIR (default: ./artifacts/runtime-deps)
  --goos GOOS                   Target OS (host platform only; no cross-platform bundling yet)
  --goarch GOARCH               Target architecture (host platform only; no cross-platform bundling yet)
  --local-shim-version VERSION  Version of chroma-go-local to download
  --tokenizers-version VERSION  Version of pure-tokenizers to download
  --onnx-runtime-version VERSION Version of ONNX Runtime shared library to download
  --help                        Show this message
EOF
}

while (( "$#" > 0 )); do
	case "$1" in
	--output-dir)
		OUTPUT_DIR="${2:?--output-dir requires a value}"
		shift 2
		;;
	--goos)
		GOOS="${2:?--goos requires a value}"
		shift 2
		;;
	--goarch)
		GOARCH="${2:?--goarch requires a value}"
		shift 2
		;;
	--local-shim-version)
		LOCAL_SHIM_VERSION="${2:?--local-shim-version requires a value}"
		shift 2
		;;
	--tokenizers-version)
		TOKENIZERS_VERSION="${2:?--tokenizers-version requires a value}"
		shift 2
		;;
	--onnx-runtime-version)
		ONNX_RUNTIME_VERSION="${2:?--onnx-runtime-version requires a value}"
		shift 2
		;;
	--help)
		usage
		exit 0
		;;
	--*)
		echo "unknown option: $1" >&2
		usage >&2
		exit 1
		;;
	*)
		echo "unexpected positional argument: $1" >&2
		usage >&2
		exit 1
		;;
	esac
done

mkdir -p "${OUTPUT_DIR}"
echo "Downloading runtime dependencies into ${OUTPUT_DIR}..."

echo "Step 1/3: Fetching native artifacts via offline bundle generator..."
go run "${SCRIPT_DIR}/offline_bundle" \
	--force \
	--output "${OUTPUT_DIR}" \
	--goos "${GOOS}" \
	--goarch "${GOARCH}" \
	--local-shim-version "${LOCAL_SHIM_VERSION}" \
	--tokenizers-version "${TOKENIZERS_VERSION}" \
	--onnx-runtime-version "${ONNX_RUNTIME_VERSION}"

if [ ! -f "${OUTPUT_DIR}/offline.env" ]; then
	echo "expected ${OUTPUT_DIR}/offline.env after dependency generation" >&2
	exit 1
fi

. "${OUTPUT_DIR}/offline.env"
echo "Step 2/3: Exporting runtime environment from generated offline env..."

MODEL_SOURCE="${OUTPUT_DIR}/onnx-models/all-MiniLM-L6-v2/onnx"
RUNTIME_HOME="${CHROMA_OFFLINE_BUNDLE_HOME:-${OUTPUT_DIR}}"
HOME_DIR="${HOME:-${USERPROFILE:-/tmp}}"
MODEL_TARGET="${HOME_DIR}/.cache/chroma/onnx_models/all-MiniLM-L6-v2/onnx"

: "${CHROMA_LIB_PATH:?missing CHROMA_LIB_PATH in ${OUTPUT_DIR}/offline.env}"
: "${TOKENIZERS_LIB_PATH:?missing TOKENIZERS_LIB_PATH in ${OUTPUT_DIR}/offline.env}"
: "${CHROMAGO_ONNX_RUNTIME_PATH:?missing CHROMAGO_ONNX_RUNTIME_PATH in ${OUTPUT_DIR}/offline.env}"
: "${TOKENIZERS_VERSION:?missing TOKENIZERS_VERSION in ${OUTPUT_DIR}/offline.env}"
: "${CHROMAGO_ONNX_RUNTIME_VERSION:?missing CHROMAGO_ONNX_RUNTIME_VERSION in ${OUTPUT_DIR}/offline.env}"

mkdir -p "${MODEL_TARGET}"
if [ -d "${MODEL_SOURCE}" ]; then
	cp -R "${MODEL_SOURCE}/." "${MODEL_TARGET}/"
	echo "Step 3/3: Copied default model cache to ${MODEL_TARGET}"
else
	echo "Model source missing at ${MODEL_SOURCE}" >&2
	exit 1
fi

quote_for_shell() {
	printf "%q" "$1"
}

{
	echo "# Load these variables before running bootstrap-dependent tests."
	printf "export CHROMA_OFFLINE_BUNDLE_HOME=%s\n" "$(quote_for_shell "${RUNTIME_HOME}")"
	printf "export CHROMA_LIB_PATH=%s\n" "$(quote_for_shell "${CHROMA_LIB_PATH}")"
	printf "export TOKENIZERS_LIB_PATH=%s\n" "$(quote_for_shell "${TOKENIZERS_LIB_PATH}")"
	printf "export CHROMAGO_ONNX_RUNTIME_PATH=%s\n" "$(quote_for_shell "${CHROMAGO_ONNX_RUNTIME_PATH}")"
	printf "export TOKENIZERS_VERSION=%s\n" "$(quote_for_shell "${TOKENIZERS_VERSION}")"
	printf "export CHROMAGO_ONNX_RUNTIME_VERSION=%s\n" "$(quote_for_shell "${CHROMAGO_ONNX_RUNTIME_VERSION}")"
} > "${OUTPUT_DIR}/runtime-env.sh"

echo "Runtime deps ready."
echo "To use them in your shell, run:"
echo "  . ${OUTPUT_DIR}/runtime-env.sh"
