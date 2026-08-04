#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
#
# fetch-data.sh — vendor the OCCT tests/blend fixture files referenced via
# `restore [locate_data_file <name>] s` into test-utilities/occt-blend/data/.
#
# The oracle (occt_blend_oracle) points CSF_TestDataPath at .../occt-blend/data and
# DRAWEXE's locate_data_file proc searches it recursively, so a flat drop of the
# referenced files (no subdir structure) is sufficient.
#
# Source: there is no public git repository for OCCT's confidential test data (most
# tests/blend fixtures are only available inside Open Cascade SAS's internal system per
# https://dev.opencascade.org/doc/overview/html/occt_contribution__tests.html). What IS
# public is the "OCCT testing dataset" archive OPEN CASCADE published on the forum
# (https://dev.opencascade.org/content/open-cascade-technology-testing-dataset-published,
# 2021-03-29): a ~2500-shape sample used to raise test coverage, distributed as a single
# .tgz. See SOURCES.md for the exact URL/sha256 and which fixtures it does/doesn't cover.
#
# Usage: fetch-data.sh [cache-dir]
#   cache-dir  where to download+extract the dataset archive; default /tmp/occt-shapes.
#              Reused if already extracted (not re-downloaded).
set -euo pipefail

CACHE_DIR="${1:-/tmp/occt-shapes}"
NEEDED_LIST="${NEEDED_LIST:-/tmp/needed-fixtures.txt}"
ARCHIVE_URL="https://dev.opencascade.org/sites/default/files/free/shapes_7.5.0.tgz"
ARCHIVE_NAME="shapes_7.5.0.tgz"

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
DATA_DIR="$HERE/data"
mkdir -p "$DATA_DIR"

if [[ ! -f "$NEEDED_LIST" ]]; then
    echo "fetch-data.sh: needed-fixtures list not found at $NEEDED_LIST" >&2
    exit 1
fi

mkdir -p "$CACHE_DIR"
ARCHIVE_PATH="$CACHE_DIR/$ARCHIVE_NAME"

if [[ ! -f "$ARCHIVE_PATH" ]]; then
    echo "fetch-data.sh: downloading $ARCHIVE_URL"
    # --proto/--proto-redir pin BOTH the initial request and every redirect hop to https, so a
    # redirect cannot silently downgrade this fixture download to plaintext (sonar shell:S6506).
    curl -sL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$ARCHIVE_PATH" "$ARCHIVE_URL"
else
    echo "fetch-data.sh: reusing already-downloaded $ARCHIVE_PATH"
fi

EXTRACT_DIR="$CACHE_DIR/extracted"
if [[ ! -d "$EXTRACT_DIR" ]]; then
    echo "fetch-data.sh: extracting $ARCHIVE_PATH into $EXTRACT_DIR"
    mkdir -p "$EXTRACT_DIR"
    tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"
else
    echo "fetch-data.sh: reusing existing extraction at $EXTRACT_DIR"
fi

requested=0
copied=0
missing=()

while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    requested=$((requested + 1))
    # -print -quit: first match wins; the archive keeps every fixture under a single
    # flat category dir (brep/, geom/, step/, ...) so duplicates are not expected.
    match=$(find "$EXTRACT_DIR" -type f -name "$name" -print -quit)
    if [[ -z "$match" ]]; then
        missing+=("$name")
        continue
    fi
    cp "$match" "$DATA_DIR/$name"
    copied=$((copied + 1))
done < "$NEEDED_LIST"

echo "fetch-data.sh: copied $copied / $requested fixtures into $DATA_DIR"
if [[ "${#missing[@]}" -gt 0 ]]; then
    echo "fetch-data.sh: ${#missing[@]} fixture(s) NOT resolved (not present in the public dataset archive — see SOURCES.md):" >&2
    printf '  %s\n' "${missing[@]}" >&2
fi
