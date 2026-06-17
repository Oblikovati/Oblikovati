#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
#
# publish-wiki.sh — assemble the Oblikovati user wiki (static pages + the generated Command
# Manual) and publish it to this repo's GitHub wiki. Idempotent and safe to run on every merge
# to develop: it only commits/pushes when the rendered content actually changed.
#
# The source of truth is docs/wiki/*.md (version-controlled here) plus the Command Manual
# rendered from the built-in command vocabulary (cmd/command-manual). GitHub's wiki is a
# downstream mirror; manual edits there are overwritten on the next run.
#
# Auth: set WIKI_REMOTE to an authenticated URL in CI, e.g.
#   WIKI_REMOTE="https://x-access-token:${TOKEN}@github.com/<owner>/<repo>.wiki.git"
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SRC="docs/wiki"
BUILD="build/wiki"
WIKI_REMOTE="${WIKI_REMOTE:-https://github.com/Oblikovati/Oblikovati.wiki.git}"

# --- 1. Assemble the page set into build/wiki ---------------------------------------
rm -rf "$BUILD"
mkdir -p "$BUILD"

# Static pages — everything in docs/wiki except the repo-only README.
shopt -s nullglob
for f in "$SRC"/*.md; do
  base="$(basename "$f")"
  [[ "$base" = "README.md" ]] && continue
  cp "$f" "$BUILD/$base"
done
shopt -u nullglob

# Generated Command Manual page (rendered from the built-in command vocabulary).
go run ./cmd/command-manual "$BUILD/Command-Manual.md"

# Generated Lua Scripting page (rendered from the wire API + its mcp:summary annotations).
# Resolve the api module dir here so the generator launches no subprocess of its own.
API_DIR="$(go list -m -f '{{.Dir}}' oblikovati.org/api)"
go run ./cmd/lua-manual "$API_DIR" "$BUILD/Lua-Scripting.md"

# --- 2. Sync into the wiki repo (preserve history; init if it's brand new) -----------
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if git clone --depth 1 "$WIKI_REMOTE" "$WORK" 2>/dev/null && [[ -e "$WORK/.git" ]]; then
  echo "publish-wiki: cloned existing wiki"
else
  echo "publish-wiki: wiki has no commits yet — initializing a fresh one"
  rm -rf "$WORK"; mkdir -p "$WORK"
  git -C "$WORK" init -q
  git -C "$WORK" remote add origin "$WIKI_REMOTE"
fi

# Replace the tracked markdown with the freshly built set (handles renames/removals).
find "$WORK" -maxdepth 1 -name '*.md' -delete
cp "$BUILD"/*.md "$WORK"/

# --- 3. Commit + push only if something changed -------------------------------------
git -C "$WORK" add -A
if git -C "$WORK" diff --cached --quiet; then
  echo "publish-wiki: no changes — wiki already up to date"
  exit 0
fi

git -C "$WORK" \
  -c user.name="${GIT_AUTHOR_NAME:-oblikovati-bot}" \
  -c user.email="${GIT_AUTHOR_EMAIL:-bot@users.noreply.github.com}" \
  commit -q -m "docs(wiki): regenerate from ${GITHUB_SHA:-develop}"

# GitHub wikis live on the 'master' branch.
git -C "$WORK" push origin HEAD:master
echo "publish-wiki: pushed updated wiki"
