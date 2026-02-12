#!/usr/bin/env bash

set -euo pipefail

[ -z "${1:-}" ] && { echo "Usage: $0 <version>"; exit 1; }

version=$1

target_dir_name=envtest-${version}
link_path=bin/envtest

[ -L ${link_path} ] && rm -r ${link_path}

mkdir -p bin
ln -s "${target_dir_name}" ${link_path}

if [ ! -e bin/"${target_dir_name}" ]; then
    os=$(go env GOOS)
    arch=$(go env GOARCH)

    # Temporary fix for Apple M1 until envtest is released for darwin-arm64 arch
    if [ "$os" == "darwin" ] && [ "$arch" == "arm64" ]; then
        arch="amd64"
    fi
    
    tmpfile=$(mktemp)
    trap "rm -f $tmpfile" EXIT
    
    # Try direct storage URL first, fall back to go.kubebuilder.io
    url1="https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${version}-${os}-${arch}.tar.gz"
    url2="https://go.kubebuilder.io/test-tools/$version/$os/$arch"
    
    if curl -fsSL -o "$tmpfile" "$url1" 2>/dev/null && tar -tzf "$tmpfile" >/dev/null 2>&1; then
        echo "Downloaded from GCS" >&2
    elif curl -fsSL -o "$tmpfile" "$url2" 2>/dev/null && tar -tzf "$tmpfile" >/dev/null 2>&1; then
        echo "Downloaded from go.kubebuilder.io" >&2
    else
        echo "Failed to download envtest tools for ${version} ${os}/${arch}" >&2
        echo "Neither ${url1} nor ${url2} returned valid tar.gz" >&2
        exit 1
    fi
    
    tar -xzf "$tmpfile" -C /tmp/
    mv "/tmp/kubebuilder" bin/"${target_dir_name}"
fi
