#!/bin/bash

# Abort on any error.
set -eo pipefail

# Abort on undefined variable.
set -u

[[ -n "${TRACE:-}" ]] && set -x

main() {
    if ! command -v asdf &>/dev/null; then
        echo "Skipping ASDF setup: asdf command not found."
        return 0
    fi

    asdf plugin add golang
    asdf plugin add golangci-lint
    asdf plugin add swag
    asdf plugin add kubectl
    asdf plugin add k9s

    asdf install
}

main "$@"
