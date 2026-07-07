#!/usr/bin/env bash
set -eu

bash <(curl -fsSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash) \
  latest \
  "$HOME/.local/bin"

actionlint --version
