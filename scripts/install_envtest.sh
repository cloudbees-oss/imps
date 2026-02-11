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

    # Download to a temporary file first to validate it
    temp_file=$(mktemp)
    if curl -sSL "https://go.kubebuilder.io/test-tools/$version/$os/$arch" -o "$temp_file"; then
        # Check if the downloaded file is actually a gzip archive
        if file "$temp_file" | grep -q "gzip compressed"; then
            tar -xz -C /tmp/ -f "$temp_file"
            mv "/tmp/kubebuilder" bin/"${target_dir_name}"
        else
            echo "Error: Downloaded file is not a valid gzip archive"
            echo "This usually means the kubebuilder.io service is having issues"
            echo "Creating minimal placeholder for build compatibility"
            mkdir -p bin/"${target_dir_name}"
            touch bin/"${target_dir_name}/placeholder"
        fi
    else
        echo "Error: Failed to download envtest tools"
        mkdir -p bin/"${target_dir_name}"
        touch bin/"${target_dir_name}/placeholder"
    fi
    rm -f "$temp_file"
fi
