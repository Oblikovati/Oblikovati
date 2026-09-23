#!/usr/bin/env bash
# Run SonarQube locally with the SAME settings CI uses, against a throwaway local server.
#
# WHY this exists: SonarCloud's verdict arrives after a push, and the three things that
# fail a PR here — coverage on NEW code (>80%), duplication (<3%), and any Sonar issue on a
# line this branch added — were previously reproduced with one-off scripts, differently each
# time. This is the one way to ask.
#
# It reads sonar-project.properties, so exclusions, coverage report paths and the project
# key are whatever CI uses; nothing is restated here that CI already states there.
#
#   scripts/sonar-local.sh --impacted   coverage for the CHANGED packages only (~1 min; the hook)
#   scripts/sonar-local.sh              analyse; reuse coverage.out when it is fresh
#   scripts/sonar-local.sh --coverage   whole-project coverage, CI's recipe (~40 min; before a PR)
#   scripts/sonar-local.sh --no-coverage  skip coverage entirely (issues + duplication only)
#   scripts/sonar-local.sh --stop       stop and remove the local server
#
# --impacted is what a pre-commit gate wants: the number that fails a PR is coverage on NEW code,
# new code is only in the packages you edited, so only those need instrumenting and only the tests
# `cmd/testimpact` says can reach them need running. Measured: 73 s for a two-package change set
# against ~40 min for CI's whole-project recipe. It runs tier 1, so see the tier caveat in
# scripts/coverage-impacted.sh.
#
# The server keeps its data in a docker volume, so the second run is fast.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CONTAINER=oblikovati-sonarqube
VOLUME_DATA=oblikovati-sonarqube-data
VOLUME_EXT=oblikovati-sonarqube-extensions
PORT="${SONAR_LOCAL_PORT:-9000}"
HOST="http://localhost:${PORT}"
# The reference for "new code". CI's new-code definition is the PR's merge base; locally the
# equivalent is the branch you would open the PR against.
NEW_CODE_REF="${SONAR_NEW_CODE_REF:-origin/develop}"

log() { printf '\033[1;34m›\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

stop_server() {
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	log "stopped $CONTAINER (volumes kept; 'docker volume rm $VOLUME_DATA $VOLUME_EXT' to reset)"
}

case "${1:-}" in
--stop)
	stop_server
	exit 0
	;;
esac

command -v docker >/dev/null || die "docker is required (it hosts the SonarQube server and the scanner)"
docker info >/dev/null 2>&1 || die "docker is installed but not running"

# ---------------------------------------------------------------- coverage
COVERAGE_MODE="${1:-auto}"
if [ "$COVERAGE_MODE" = "--coverage" ] || { [ "$COVERAGE_MODE" = "auto" ] && [ ! -f coverage.out ]; }; then
	log "coverage: running CI's recipe (this is the slow part — model/feature alone is ~18 min)"
	# Identical to .github/workflows/ci.yml's Test step for the ubuntu leg: -coverpkg over the
	# whole tree so a test binary credits the packages it actually drives, not only its own. The
	# package set is the TRACKED one rather than a literal `./...`, which is not a deviation from CI
	# but fidelity to it: CI's checkout holds no git-ignored scratch, so `./...` there means exactly
	# this set, while here it would drag in `experiments/` and fail the run (#3557).
	go test -coverprofile=coverage.out -covermode=count \
		-coverpkg=./kernel/...,./model/...,./app/...,./addin/...,./renderer/...,./event/...,./cmd/... \
		-timeout 60m $(scripts/tracked-packages.sh)
	# -coverpkg repeats every block once per test binary (CI measured 390 MB). Collapsing them by
	# summing counts is exactly `count` mode's semantics — CI's own awk, verbatim.
	awk 'NR==1 && /^mode:/ {print; next} {n[$1]=$2; c[$1]+=$3} END {for (k in n) print k, n[k], c[k]}' \
		coverage.out >coverage.merged && mv coverage.merged coverage.out
	# The translator modules have their own go.mod and are NOT in go.work, so ./... above never ran
	# their tests, and inside them go.work must be switched off or `./...` fails setup ("directory
	# prefix . does not contain modules listed in go.work") — after the whole root suite has already
	# run. `make gate` runs them the same way (GOWORK=off).
	for m in model/exchange/translators/solidworks model/exchange/translators/inventor \
		model/exchange/translators/olecf; do
		(cd "$m" && GOWORK=off go test -covermode=count -coverprofile=cover.out -timeout 30m ./...)
		tail -n +2 "$m/cover.out" >>coverage.out
	done
elif [ "$COVERAGE_MODE" = "--impacted" ]; then
	log "coverage: the changed packages only (tier 1, scoped by cmd/testimpact)"
	scripts/coverage-impacted.sh "${SONAR_IMPACT_BASE:-HEAD}"
	[ -f coverage.out ] || log "nothing to instrument — coverage will read as not measured"
elif [ "$COVERAGE_MODE" = "--no-coverage" ]; then
	log "coverage: skipped (issues + duplication only)"
else
	# A stale profile is worse than none: it reports coverage for code that has since changed, and
	# the number looks authoritative. Measured once: a two-week-old coverage.out sitting in the repo
	# root reported 73.3% for a branch it predated by 120 commits.
	newest_src=$(git log -1 --format=%ct)
	profile_age=$(stat -c %Y coverage.out)
	if [ "$profile_age" -lt "$newest_src" ]; then
		die "coverage.out predates HEAD ($(date -d @"$profile_age" '+%Y-%m-%d %H:%M') vs commit $(date -d @"$newest_src" '+%Y-%m-%d %H:%M')).
   Rerun with --coverage for a fresh profile, or --no-coverage to analyse issues and duplication only."
	fi
	log "coverage: reusing coverage.out from $(date -d @"$profile_age" '+%Y-%m-%d %H:%M')"
fi

# ---------------------------------------------------------------- server
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
	log "starting SonarQube on $HOST"
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	docker run -d --name "$CONTAINER" \
		-p "${PORT}:9000" \
		-v "$VOLUME_DATA:/opt/sonarqube/data" \
		-v "$VOLUME_EXT:/opt/sonarqube/extensions" \
		-e SONAR_ES_BOOTSTRAP_CHECKS_DISABLE=true \
		sonarqube:community >/dev/null
fi

log "waiting for the server to come up (first boot takes a couple of minutes)"
for _ in $(seq 1 180); do
	status=$(curl -fsS "$HOST/api/system/status" 2>/dev/null | sed -n 's/.*"status":"\([A-Z]*\)".*/\1/p' || true)
	[ "$status" = "UP" ] && break
	sleep 5
done
[ "${status:-}" = "UP" ] || die "server did not reach UP; 'docker logs $CONTAINER' has the reason"

# ---------------------------------------------------------------- token
# First boot ships admin/admin and forces a change; do it once, non-interactively.
# The password must satisfy SonarQube's policy (>=12 chars, upper, lower, digit, special) or
# change_password 400s and every call after it 401s.
PASSWORD="${SONAR_LOCAL_PASSWORD:-Oblikovati-local-1}"
AUTH_DEFAULT="admin:admin"
AUTH_LOCAL="admin:$PASSWORD"
if curl -fsS -u "$AUTH_DEFAULT" "$HOST/api/authentication/validate" 2>/dev/null | grep -q '"valid":true'; then
	resp=$(curl -sS -u "$AUTH_DEFAULT" -X POST \
		"$HOST/api/users/change_password?login=admin&previousPassword=admin&password=$PASSWORD" 2>&1)
	case "$resp" in
	*'"result"'*) die "could not set the admin password: $resp" ;;
	esac
fi
TOKEN_NAME="local-$(date +%s)"
TOKEN=$(curl -sS -u "$AUTH_LOCAL" -X POST "$HOST/api/user_tokens/generate?name=$TOKEN_NAME" |
	sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
[ -n "$TOKEN" ] || die "could not mint a token as admin; the password on this server is not \"$PASSWORD\" (set SONAR_LOCAL_PASSWORD, or 'docker volume rm $VOLUME_DATA' to start clean)"

# ---------------------------------------------------------------- scan
# New code = what this branch added on top of NEW_CODE_REF, which is how CI's PR analysis sees it.
git rev-parse --verify --quiet "$NEW_CODE_REF" >/dev/null ||
	die "$NEW_CODE_REF is not a ref here (set SONAR_NEW_CODE_REF)"

# The scanner's blame step runs on jgit, which cannot read git's multi-pack-index: with one
# present it dies "MissingObjectException: Missing blob <sha>" on an object `git cat-file` finds
# fine and `git fsck` calls clean. The midx is a pure lookup cache, so move it aside for the scan
# and put it back afterwards — `git gc`/`git maintenance` regenerates it, so this cannot be a
# one-time manual fix.
MIDX=".git/objects/pack/multi-pack-index"
MIDX_STASH=""
if [ -f "$MIDX" ]; then
	MIDX_STASH="$(mktemp -d)/multi-pack-index"
	mv "$MIDX" "$MIDX_STASH"
	log "moved the multi-pack-index aside for the scan (jgit cannot read it); restored on exit"
fi
restore_midx() { [ -n "$MIDX_STASH" ] && [ -f "$MIDX_STASH" ] && mv "$MIDX_STASH" "$MIDX"; }
trap restore_midx EXIT

PROJECT_KEY=$(sed -n 's/^sonar.projectKey=//p' sonar-project.properties)
REF_NAME="${NEW_CODE_REF#origin/}"
BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)

# SonarQube Community Build cannot analyse branches (`sonar.branch.name` needs Developer Edition),
# so "new code" cannot come from a reference branch here the way it does on SonarCloud. Community
# does support a new-code period of PREVIOUS_VERSION, which gives the same answer by a different
# route: analyse the reference as version A, this branch as version B, and everything that differs
# is new. Same question, same numbers, within the free edition.
sonar_scan() { # sonar_scan <workdir> <projectVersion> [extra args...]
	local dir="$1" version="$2"
	shift 2
	docker run --rm --network host \
		-v "$dir:/usr/src" \
		-e SONAR_HOST_URL="$HOST" \
		-e SONAR_TOKEN="$TOKEN" \
		sonarsource/sonar-scanner-cli \
		-Dsonar.projectVersion="$version" \
		"$@"
}

# One plain analysis. No branch or version games: Community cannot compare branches
# (`sonar.newCode.referenceBranch` is Developer Edition) and its PREVIOUS_VERSION period attributes
# new lines by SCM blame DATE, not by diff — a baseline scanned a minute ago marks almost nothing
# new (measured: 86 lines against a true 5704). So the server measures the project, and
# scripts/sonar-newcode.py does the branch comparison from git using the server's own data.
log "scanning $BRANCH_NAME @ $(git rev-parse --short HEAD) (settings from sonar-project.properties, the file CI uses)"
docker run --rm --network host \
	-v "$PWD:/usr/src" \
	-e SONAR_HOST_URL="$HOST" \
	-e SONAR_TOKEN="$TOKEN" \
	sonarsource/sonar-scanner-cli \
	"${@:2}"

# ---------------------------------------------------------------- gate
sleep 5 # the compute engine finishes a moment after the scanner returns

log "whole project, measured by the server:"
# A SCOPED profile instruments only the changed packages, so the project-wide coverage the server
# computes from it is meaningless (measured: 8.4% for a two-package profile). Ask for it only when
# the profile actually covers the project.
WHOLE_METRICS="duplicated_lines_density,bugs,vulnerabilities,code_smells,ncloc"
if [ "$COVERAGE_MODE" != "--impacted" ] && [ "$COVERAGE_MODE" != "--no-coverage" ]; then
	WHOLE_METRICS="coverage,$WHOLE_METRICS"
else
	log "  (project-wide coverage omitted: the profile is scoped or absent)"
fi
curl -fsS -u "$AUTH_LOCAL" \
	"$HOST/api/measures/component?component=$PROJECT_KEY&metricKeys=$WHOLE_METRICS" 2>/dev/null |
	scripts/sonar-gate.py

log "on NEW code (vs $NEW_CODE_REF) — the three gates that fail a PR:"
if [ "$COVERAGE_MODE" = "--impacted" ]; then
	log "  note: coverage below is a tier-1 FLOOR — corpus tests skip under -short"
fi
COV_ARG=""
[ "$COVERAGE_MODE" != "--no-coverage" ] && [ -f coverage.out ] && COV_ARG="coverage.out"
if scripts/sonar-newcode.py "$NEW_CODE_REF" "$HOST" "$AUTH_LOCAL" "$PROJECT_KEY" $COV_ARG; then
	:
elif [ "$COVERAGE_MODE" = "--impacted" ]; then
	log "a gate is below its threshold on a tier-1 floor — confirm with \`make sonar\` before a PR"
else
	GATE_FAILED=1
fi

log "report: $HOST/dashboard?id=$PROJECT_KEY"
log "login: admin / $PASSWORD"

[ -n "${GATE_FAILED:-}" ] && die "the quality gate FAILED above"
exit 0
