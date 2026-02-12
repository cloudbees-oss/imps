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

    mkdir -p bin/"${target_dir_name}"
    curl -sSL "https://github.com/kubernetes-sigs/controller-tools/releases/download/envtest-v${version}/envtest-v${version}-${os}-${arch}.tar.gz" | tar -xz -C /tmp/
    mv "/tmp/controller-tools/envtest" bin/"${target_dir_name}"/bin
fi