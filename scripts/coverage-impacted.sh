#!/usr/bin/env bash
# Coverage for a pre-commit gate: instrument only the packages that CHANGED, and run only the
# tests that can reach them.
#
# WHY this is not CI's recipe: CI measures the whole project, so it instruments every package and
# runs every test — about forty minutes here. But the number that fails a PR is coverage on NEW
# code, and new code lives only in the packages you edited. Instrumenting those and running the
# tests `cmd/testimpact` says can reach them measures the same lines in a fraction of the time.
#
#   scripts/coverage-impacted.sh [base-ref]     default base: HEAD (i.e. the working tree)
#
# Writes coverage.out, in the same format and with the same `count` mode CI produces, so
# sonar-project.properties' sonar.go.coverage.reportPaths picks it up unchanged.
#
# TIER: this runs tier 1 (`-short`), the tier a pre-commit hook is allowed to cost. Corpus and
# oracle tests skip themselves there, so a line only exercised by the corpus reads as uncovered.
# That makes this an EARLY WARNING, not the CI number: it catches "nothing reaches this line at
# all", which is the common case, and it will understate coverage for corpus-driven code. Before a
# PR, `make sonar` runs the full recipe and that is the number to believe.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

BASE="${1:-HEAD}"

# The packages whose non-test sources changed: these hold the new lines, so these are what must be
# instrumented. A test-only change adds no new production lines and needs no -coverpkg entry.
#
# Plain word-split strings rather than `mapfile` arrays: macOS ships bash 3.2, which has no mapfile,
# and the pre-commit hook runs this on every developer's machine. Package paths hold no spaces.
changed=$(
	git diff --name-only "$BASE" -- '*.go' |
		{ grep -v '_test\.go$' || true; } | # 1 = only tests changed: an answer, not a failure
		while read -r f; do dirname "$f"; done |
		sort -u |
		while read -r d; do if [ -d "$d" ]; then echo "./$d"; fi; done
)

if [ -z "$changed" ]; then
	echo "no production Go package changed — nothing to measure"
	exit 0
fi

# The tests that can reach them, from the repo's own impact analysis rather than a guess.
#
# Dropping git-IGNORED packages is not a convenience: `experiments/` is git-ignored scratch that
# `go list ./...` still walks, so a half-finished experiment on your disk fails a gate about code
# you are committing — and CI, which never sees those files, passes. The gate measures what git
# tracks. (The same directory fails `make test` today, for the same reason; that is #3557.)
# cmd/testimpact already drops git-ignored packages itself (#3557), so its list is used as given.
impacted=$(go run ./cmd/testimpact -base "$BASE")
if [ -z "$impacted" ]; then
	echo "testimpact says no package owns the change — nothing to run"
	exit 0
fi

coverpkg=$(printf '%s\n' "$changed" | paste -sd, -)

printf 'instrumenting %d changed package(s), running %d impacted test package(s)\n' \
	"$(printf '%s\n' "$changed" | wc -l | tr -d ' ')" "$(printf '%s\n' "$impacted" | wc -l | tr -d ' ')"
printf '  → %s\n' $impacted

# -covermode=count to match CI: Sonar reads hit counts, and `set` mode would change the profile's
# semantics for anything that merges it.
CGO_ENABLED=0 go test -short -covermode=count \
	-coverpkg="$coverpkg" \
	-coverprofile=coverage.out \
	$impacted

# -coverpkg emits every instrumented block once per test binary, so the same block repeats. Summing
# the counts is exactly what `count` mode means — CI's own awk, so the file is byte-comparable.
awk 'NR==1 && /^mode:/ {print; next} {n[$1]=$2; c[$1]+=$3} END {for (k in n) print k, n[k], c[k]}' \
	coverage.out >coverage.merged && mv coverage.merged coverage.out

echo "wrote coverage.out ($(wc -l <coverage.out) blocks)"
