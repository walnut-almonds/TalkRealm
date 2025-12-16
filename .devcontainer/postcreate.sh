#!/bin/bash

# Bashrc setup
echo '' >>~/.bashrc
echo 'source /workspaces/TalkRealm/.devcontainer/bashrc.bash' >>~/.bashrc

# Install apt utility
sudo apt update
sudo apt install -y iputils-ping

# Install ASDF
go install github.com/asdf-vm/asdf/cmd/asdf@v0.18.0

# ASDF Package Manager Plugins
asdf plugin add golang
asdf plugin add golangci-lint
asdf plugin add swag
