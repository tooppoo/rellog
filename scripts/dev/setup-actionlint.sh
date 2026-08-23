#!/usr/bin/env bash
set -eu

curl -fsSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash > tmp/install-actionlint.bash

sudo bash tmp/install-actionlint.bash \
  latest \
  "/usr/local/bin"

actionlint --version
