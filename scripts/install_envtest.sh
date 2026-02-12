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
    echo "Installing envtest binaries for Kubernetes ${version}..."
    
    # Use setup-envtest tool from controller-runtime (official method)
    go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use "${version}" --bin-dir "bin/${target_dir_name}/bin" -p path >/dev/null
    
    if [ ! -d "bin/${target_dir_name}/bin" ]; then
        echo "Failed to install envtest binaries for version ${version}" >&2
        exit 1
    fi
    
    echo "Envtest binaries installed successfully"
fi
