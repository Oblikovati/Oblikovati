#!/usr/bin/env bash
# Pin oblikovati.org/api to the latest published release across this repo's modules
# (the root application module and the cgo `head` submodule), so the committed
# `require` never drifts behind the auto-released contract.
#
# CI runs this on every PR and auto-commits the result to the PR branch; run it
# locally the same way. It edits only the `require` version string (a textual
# `go mod edit`, no network, no go.sum change — local builds still resolve the
# contract through go.work). Prints the version it pinned.
#
#   $ scripts/sync-api-version.sh           # pin to the latest release
#   $ scripts/sync-api-version.sh v0.2.0    # pin to a specific release
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
version="${1:-$("$here/latest-api-version.sh")}"

case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "sync-api-version: $version is not a vX.Y.Z release tag" >&2; exit 1 ;;
esac

# The repo's two Go modules; both depend on the contract and must stay in lockstep.
for mod in . head; do
  go -C "$here/../$mod" mod edit -require="oblikovati.org/api@$version"
done

echo "$version"
