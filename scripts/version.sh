#!/usr/bin/env bash
# Compute the release version: MAJOR.MINOR from the repo-root VERSION file plus a
# UTC-timestamp PATCH (the build number). See RELEASING.md for the bump rules.
#
#   Usage: scripts/version.sh [stable|nightly]   (default: stable)
#   stable  -> MAJOR.MINOR.<YYYYmmddHHMMSS>
#   nightly -> MAJOR.MINOR.<YYYYmmddHHMMSS>-nightly   (a semver prerelease)
set -euo pipefail

channel="${1:-stable}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mm="$(tr -d '[:space:]' < "$root/VERSION")"

if ! printf '%s' "$mm" | grep -qE '^[0-9]+\.[0-9]+$'; then
	echo "version.sh: VERSION must be MAJOR.MINOR (got '$mm')" >&2
	exit 1
fi

ts="$(date -u +%Y%m%d%H%M%S)"
case "$channel" in
stable) printf '%s.%s\n' "$mm" "$ts" ;;
nightly) printf '%s.%s-nightly\n' "$mm" "$ts" ;;
*)
	echo "version.sh: unknown channel '$channel' (want stable|nightly)" >&2
	exit 1
	;;
esac
