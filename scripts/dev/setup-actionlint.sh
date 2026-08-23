#!/usr/bin/env bash
set -eu

sudo bash <(curl -fsSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash) \
  latest \
  "/usr/local/bin"

actionlint --version
