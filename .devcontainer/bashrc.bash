# GO
export GOPATH="${HOME}/go"
export GOMODCACHE="${GOPATH}/pkg/mod"
export GOBIN="${GOPATH}/bin"
export PATH="${PATH}:${GOBIN}"

# ASDF
export PATH="${ASDF_DATA_DIR:-${HOME}/.asdf}/shims:${PATH}"

if command -v asdf &> /dev/null; then
    # See: https://asdf-vm.com/guide/getting-started.html#_2-configure-asdf
    . <(asdf completion bash)
fi
