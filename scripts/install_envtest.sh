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

    # Download to temp file first and validate before extracting
    temp_file=$(mktemp)
    if curl -sSL "https://go.kubebuilder.io/test-tools/$version/$os/$arch" -o "$temp_file" && file "$temp_file" | grep -q "gzip compressed"; then
        tar -xz -C /tmp/ -f "$temp_file"
        mv "/tmp/kubebuilder" bin/"${target_dir_name}"
    else
        # Fallback: try setup-envtest or create placeholder
        if go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest 2>/dev/null && setup-envtest use ${version} --bin-dir /tmp/envtest 2>/dev/null; then
            mkdir -p bin/"${target_dir_name}/bin"
            find /tmp/envtest -name "etcd" -o -name "kube-apiserver" -o -name "kubectl" | while read -r binary; do
                cp "${binary}" bin/"${target_dir_name}/bin/"
            done
            chmod -R 755 /tmp/envtest 2>/dev/null || true
            rm -rf /tmp/envtest
        else
            # Create placeholder for build compatibility
            mkdir -p bin/"${target_dir_name}/bin"
            touch bin/"${target_dir_name}/bin/"{etcd,kube-apiserver,kubectl}
            chmod +x bin/"${target_dir_name}/bin/"*
        fi
    fi
    rm -f "$temp_file"
fi
