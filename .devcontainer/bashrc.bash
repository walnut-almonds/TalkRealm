# GO
export GOPATH="${HOME}/go"
export GOMODCACHE="${GOPATH}/pkg/mod"
export GOBIN="${GOPATH}/bin"
export PATH="${PATH}:${GOBIN}"

# ASDF
export PATH="${ASDF_DATA_DIR:-${HOME}/.asdf}/shims:${PATH}"
