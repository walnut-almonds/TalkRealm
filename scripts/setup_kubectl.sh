#!/bin/bash

# Abort on any error.
set -eo pipefail

# Abort on undefined variable.
set -u

[[ -n "${TRACE:-}" ]] && set -x

main() {
    if [[ ! -d /home/vscode/.kube-host ]]; then
        echo "Skipping kubeconfig setup: /home/vscode/.kube-host does not exist."
        return 0
    fi

    cp -r /home/vscode/.kube-host ~/.kube
    chmod 600 ~/.kube/config
    kubectl config set-cluster docker-desktop --server=https://kubernetes.docker.internal:6443
}

main "$@"
