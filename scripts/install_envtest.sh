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
    
    # Try direct storage URL first, fall back to go.kubebuilder.io
    url="https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${version}-${os}-${arch}.tar.gz"
    if ! curl -fsSL "$url" | tar -xz -C /tmp/; then
        echo "Primary URL failed, trying go.kubebuilder.io..." >&2
        curl -fsSL "https://go.kubebuilder.io/test-tools/$version/$os/$arch" | tar -xz -C /tmp/
    fi
    mv "/tmp/kubebuilder" bin/"${target_dir_name}"
fi
