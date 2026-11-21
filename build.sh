#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
set -euo pipefail
BOOTSTRAP_ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=tools/bootstrap/unix.sh
source "$BOOTSTRAP_ROOT/tools/bootstrap/unix.sh"
bootstrap_main "$@"
