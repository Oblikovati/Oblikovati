#!/usr/bin/env bash
# Print the root module's Go packages that GIT TRACKS, one import path per line — what `./...` should
# have meant.
#
# WHY this exists: `./...` walks the working tree, and the working tree holds directories git ignores.
# `experiments/` is ignored scratch with zero tracked files, so a half-finished experiment on your disk
# fails a gate about the code you are committing, while CI — which never sees those files — passes.
# That is a gate's opposite, and it cost a 35-minute `make gate` run to an experiment whose fixture
# path was a /tmp directory from another session (#3557).
#
# The package set comes from `go list ./...` rather than from `git ls-files`, because the pattern is
# the authority on what a package IS: it already drops the nested modules (`head/` and the exchange
# translators each own a go.mod, and the root module cannot compile them) and directories whose files
# are all excluded by build constraints. This only removes what git does not track.
#
#   go test $(scripts/tracked-packages.sh)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

MODULE=$(go list -m)

mapfile -t pkgs < <(go list ./...)
[ "${#pkgs[@]}" -gt 0 ] || {
	echo "tracked-packages: go list ./... returned nothing — refusing to report an empty set" >&2
	exit 1
}

pkg_dir() { # import path -> path relative to the module root
	local dir="${1#"$MODULE"}"
	dir="${dir#/}"
	printf '%s\n' "${dir:-.}"
}

# One `git check-ignore --stdin` for the whole set: it prints the paths it considers ignored, and
# exits 1 when none is, which is the ordinary case on a clean checkout.
ignored=$(for pkg in "${pkgs[@]}"; do pkg_dir "$pkg"; done | git check-ignore --stdin || true)

for pkg in "${pkgs[@]}"; do
	dir=$(pkg_dir "$pkg")
	grep -qxF "$dir" <<<"$ignored" || printf '%s\n' "$pkg"
done
