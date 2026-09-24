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
# Two constraints the first version broke, both on the first CI run:
#   - bash 3.2: macOS ships it, and it has no `mapfile`. The script died, `$(shell)` in the Makefile
#     swallowed the error, and the gate's package list came out EMPTY — `go test` with no package
#     argument then tests only the current directory, a silent green. The archguard guard caught it.
#   - no module-path arithmetic: `go list -m` prints EVERY module in a go.work workspace, so a prefix
#     trimmed from it matched nothing. Git is handed each package's absolute directory instead, and
#     `git check-ignore` echoes back the paths it ignores exactly as it was given them.
#
#   go test $(scripts/tracked-packages.sh)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

listing=$(go list -f '{{.ImportPath}} {{.Dir}}' ./...)
if [ -z "$listing" ]; then
	echo "tracked-packages: go list ./... returned nothing — refusing to report an empty set" >&2
	exit 1
fi

# One `git check-ignore --stdin` for the whole set. It exits 1 when nothing is ignored, which is the
# ordinary case on a clean checkout, so that status is an answer, not a failure.
ignored=$(printf '%s\n' "$listing" | cut -d' ' -f2- | git check-ignore --stdin || true)

printf '%s\n' "$listing" | while read -r pkg dir; do
	if [ -n "$ignored" ] && printf '%s\n' "$ignored" | grep -qxF -- "$dir"; then
		continue
	fi
	printf '%s\n' "$pkg"
done
