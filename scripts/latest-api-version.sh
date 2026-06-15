#!/usr/bin/env bash
# Print the newest published oblikovati.org/api release tag (e.g. "v0.2.0").
#
# The contract is a sibling repo that auto-releases a vX.Y.Z tag on every merge
# (Oblikovati.API/RELEASING.md). This is the single source of truth both the CI
# composite action and scripts/sync-api-version.sh use to resolve "the latest
# release", so the resolution logic lives in exactly one place.
#
#   $ scripts/latest-api-version.sh
#   v0.2.0
set -euo pipefail

API_REPO="${API_REPO:-https://github.com/Oblikovati/Oblikovati.API}"

# --refs drops the dereferenced "^{}" peel lines; -v:refname sorts semver newest-first,
# so the first clean vMAJOR.MINOR.PATCH tag is the latest stable release (pre-releases
# carrying a -suffix are intentionally excluded).
tag=$(git ls-remote --tags --refs --sort=-v:refname "$API_REPO" 'refs/tags/v*' \
  | sed -nE 's@.*refs/tags/(v[0-9]+\.[0-9]+\.[0-9]+)$@\1@p' \
  | head -1)

if [ -z "$tag" ]; then
  echo "latest-api-version: no vX.Y.Z release tag found at $API_REPO" >&2
  exit 1
fi

echo "$tag"
