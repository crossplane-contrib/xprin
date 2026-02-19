#!/bin/sh

set -eu

# Use "latest" if not specified, otherwise accept explicit tag like v0.1.0
VERSION=${VERSION:-latest}
# Binary to download: "xprin" (default) or "xprin-helpers". Used in release artifact name and output file.
PACKAGE=${PACKAGE:-xprin}
# Set to "true" (case-insensitive) to download the .tar.gz bundle instead of the raw binary
COMPRESSED=${COMPRESSED:-"False"}
# Set to "true" (case-insensitive) to download .sha256 and verify checksums. Recommended when COMPRESSED=true.
VERIFY_SHA=${VERIFY_SHA:-"False"}

os=$(uname -s)
arch=$(uname -m)
OS=${OS:-"${os}"}
ARCH=${ARCH:-"${arch}"}
OS_ARCH=""
BIN="${PACKAGE}"
EXT=""

unsupported_arch() {
	os="$1"
	arch="$2"
	echo "${PACKAGE} does not support $os / $arch at this time."
	exit 1
}

# verify_sha256 verifies that the given file matches the checksum in the .sha256 file.
# The .sha256 file contains only the 64-char checksum (no filename).
verify_sha256() {
	downloaded_file="$1"
	checksum_file="$2"
	if [ ! -f "$downloaded_file" ] || [ ! -f "$checksum_file" ]; then
		echo "SHA256 verification failed: missing downloaded file or checksum file."
		exit 1
	fi
	expected=$(tr -d '\n\r ' < "$checksum_file" | head -c 64)
	if [ -z "$expected" ] || [ ${#expected} -ne 64 ]; then
		echo "SHA256 verification failed: invalid checksum file."
		exit 1
	fi
	line="${expected}  ${downloaded_file}"
	if command -v sha256sum >/dev/null 2>&1; then
		echo "$line" | sha256sum --check
	elif command -v shasum >/dev/null 2>&1; then
		echo "$line" | shasum -a 256 --check
	else
		echo "SHA256 verification failed: need sha256sum or shasum."
		exit 1
	fi
}

case $OS in
CYGWIN* | MINGW64* | Windows*)
	if [ "$ARCH" = "x86_64" ]; then
		EXT=".exe"
		OS_ARCH="windows_amd64"
		BIN="${PACKAGE}.exe"
	else
		unsupported_arch "$OS" "$ARCH"
	fi
	;;
Darwin)
	case $ARCH in
	x86_64 | amd64)
		OS_ARCH="darwin_amd64"
		;;
	arm64)
		OS_ARCH="darwin_arm64"
		;;
	*)
		unsupported_arch "$OS" "$ARCH"
		;;
	esac
	;;
Linux)
	case $ARCH in
	x86_64 | amd64)
		OS_ARCH="linux_amd64"
		;;
	arm64 | aarch64)
		OS_ARCH="linux_arm64"
		;;
	arm)
		OS_ARCH="linux_arm"
		;;
	ppc64le)
		OS_ARCH="linux_ppc64le"
		;;
	*)
		unsupported_arch "$OS" "$ARCH"
		;;
	esac
	;;
*)
	unsupported_arch "$OS" "$ARCH"
	;;
esac

_compr=$(echo "$COMPRESSED" | tr '[:upper:]' '[:lower:]')
_verify=$(echo "$VERIFY_SHA" | tr '[:upper:]' '[:lower:]')

if [ "${_compr}" = "true" ]; then
	url_file="${PACKAGE}_${OS_ARCH}.tar.gz"
	url_error="a compressed file for "
else
	url_file="${PACKAGE}_${OS_ARCH}${EXT}"
	url_error=""
fi

if [ "$VERSION" = "latest" ]; then
	url="https://github.com/crossplane-contrib/xprin/releases/latest/download/${url_file}"
else
	url="https://github.com/crossplane-contrib/xprin/releases/download/${VERSION}/${url_file}"
fi

if ! curl -sfL "${url}" -o "${url_file}"; then
	echo "Failed to download ${PACKAGE}. Please make sure ${url_error}version ${VERSION} exists."
	echo "  https://github.com/crossplane-contrib/xprin/releases"
	exit 1
fi

if [ "${_verify}" = "true" ]; then
	url_sha256="${url_file}.sha256"
	if [ "$VERSION" = "latest" ]; then
		url_sha256_full="https://github.com/crossplane-contrib/xprin/releases/latest/download/${url_sha256}"
	else
		url_sha256_full="https://github.com/crossplane-contrib/xprin/releases/download/${VERSION}/${url_sha256}"
	fi
	if ! curl -sfL "${url_sha256_full}" -o "${url_sha256}"; then
		echo "Failed to download ${url_sha256} for verification."
		exit 1
	fi
	verify_sha256 "${url_file}" "${url_sha256}"
fi

if [ "${_compr}" = "true" ]; then
	if ! tar xzf "${url_file}"; then
		echo "Failed to unpack the ${PACKAGE} compressed file."
		exit 1
	fi
	if [ "${_verify}" = "true" ]; then
		# Verify the extracted binary, then remove tarball and all .sha256 files
		verify_sha256 "${BIN}" "${BIN}.sha256"
		rm -f "${BIN}.sha256" "${url_file}.sha256" "${url_file}"
	else
		rm -f "${BIN}.sha256" "${url_file}"
	fi
else
	if [ "${_verify}" = "true" ]; then
		rm -f "${url_file}.sha256"
	fi
	mv "${url_file}" "${BIN}"
fi

chmod +x "${BIN}"

echo "${PACKAGE} downloaded successfully!"
echo
echo "To finish installation, run:"
echo "  sudo mv ${BIN} /usr/local/bin/"
echo "  ${PACKAGE} --help"
echo
echo "Visit https://github.com/crossplane-contrib/xprin for more info. 🚀"
