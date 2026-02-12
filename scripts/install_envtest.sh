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
    if curl -sSL "https://go.kubebuilder.io/test-tools/$version/$os/$arch" -o "$temp_file" && tar -tzf "$temp_file" >/dev/null 2>&1; then
        tar -xz -C /tmp/ -f "$temp_file"
        mv "/tmp/kubebuilder" bin/"${target_dir_name}"
    else
        # Fallback: try setup-envtest to obtain the binaries if the download fails or is not a valid gzip file
        echo "Attempting to install setup-envtest@release-0.22..."
        if go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.22; then
            echo "setup-envtest installed successfully"

            # Use explicit path instead of relying on PATH
            setup_envtest="$(go env GOPATH)/bin/setup-envtest"
            temp_envtest_dir=$(mktemp -d)
            if "$setup_envtest" use ${version} --bin-dir "${temp_envtest_dir}"; then
                mkdir -p bin/"${target_dir_name}/bin"
                find "${temp_envtest_dir}" -type f \( -name "etcd" -o -name "kube-apiserver" -o -name "kubectl" \) -exec cp {} bin/"${target_dir_name}/bin/" \;
                chmod -R 755 bin/"${target_dir_name}/bin/"

                # Quick verification that binaries exist
                if [ ! -f "bin/${target_dir_name}/bin/etcd" ] || [ ! -f "bin/${target_dir_name}/bin/kube-apiserver" ] || [ ! -f "bin/${target_dir_name}/bin/kubectl" ]; then
                    echo "ERROR: Required binaries missing after setup-envtest"
                    exit 1
                fi

                # Fix permissions before cleanup to avoid "permission denied" errors
                chmod -R u+w "${temp_envtest_dir}" 2>/dev/null || true
                rm -rf "${temp_envtest_dir}"
                echo "Successfully installed envtest tools via setup-envtest"
            else
                # Fail explicitly when we can't obtain real binaries
                echo "ERROR: Failed to install envtest tools. setup-envtest installed but could not download version ${version}."
                echo "This likely means version ${version} is not available from the envtest repository."
                chmod -R u+w "${temp_envtest_dir}" 2>/dev/null || true
                rm -rf "${temp_envtest_dir}"
                exit 1
            fi
        else
            # Fail explicitly when we can't obtain real binaries
            echo "ERROR: Failed to install envtest tools. setup-envtest installation failed."
            echo "This likely means:"
            echo "1. The kubebuilder.io URL is not accessible or returns invalid data"
            echo "2. setup-envtest tool installation failed"
            exit 1
        fi
    fi
    rm -f "$temp_file"
fi