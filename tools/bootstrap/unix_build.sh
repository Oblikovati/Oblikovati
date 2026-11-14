#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
bootstrap_untracked_hashes() {
    local relative absolute
    while IFS= read -r -d '' relative; do
        absolute="$BOOTSTRAP_ROOT/$relative"
        case $absolute in "$CARGO_TARGET_DIR"/* | "$BOOTSTRAP_ROOT/.tools"/*) continue ;; esac
        [[ -f $absolute ]] || continue
        printf '%s\0%s\0' "$relative" "$(bootstrap_file_digest "$absolute")"
    done < <(git ls-files --others --exclude-standard -z)
}
bootstrap_source_fingerprint() {
    {
        git rev-parse HEAD || return
        git diff --binary HEAD -- . || return
        bootstrap_untracked_hashes || return
    } | bootstrap_digest
}
bootstrap_fresh() {
    [[ -x $BOOTSTRAP_GUI && -f $BOOTSTRAP_STAMP ]] && bootstrap_has git || return 1
    local source binary
    source=$(bootstrap_source_fingerprint) || return
    binary=$(bootstrap_file_digest "$BOOTSTRAP_GUI") || return
    [[ $(sed -n '1p' "$BOOTSTRAP_STAMP") = "$source" && $(sed -n '2p' "$BOOTSTRAP_STAMP") = "$binary" ]]
}
bootstrap_virtual_display() {
    [[ $BOOTSTRAP_OS = Linux && -z ${DISPLAY:-} ]] || return 0
    local display_file="$BOOTSTRAP_TEMP/display" attempt
    Xvfb -displayfd 3 -screen 0 1440x900x24 -noreset 3>"$display_file" >"$BOOTSTRAP_TEMP/xvfb.log" 2>&1 &
    BOOTSTRAP_XVFB_PID=$!
    for ((attempt = 0; attempt < 100; attempt++)); do
        if [[ -s $display_file ]]; then
            DISPLAY=":$(cat "$display_file")" || return
            export DISPLAY
            return
        fi
        kill -0 "$BOOTSTRAP_XVFB_PID" 2>/dev/null || break
        sleep 0.1
    done
    cat "$BOOTSTRAP_TEMP/xvfb.log" >&2
    bootstrap_fail 'Xvfb did not provide a display within 10 seconds; expected a usable X server for the Rust build system'
}
bootstrap_compile() {
    local before after binary task=build
    [[ ${1:-build} != run ]] || task=gui
    before=$(bootstrap_source_fingerprint) || return
    mkdir -p -- "$BOOTSTRAP_ROOT/.tools"
    rm -f -- "$BOOTSTRAP_STAMP"
    bootstrap_virtual_display || return
    cargo xtask "$task" || return
    [[ -x $BOOTSTRAP_GUI ]] || {
        bootstrap_fail "build output $BOOTSTRAP_GUI: expected the oblikovati GUI executable"
        return 1
    }
    after=$(bootstrap_source_fingerprint) || return
    [[ $before = "$after" ]] || {
        bootstrap_fail 'sources changed during the build; rerun to create a current GUI'
        return 1
    }
    binary=$(bootstrap_file_digest "$BOOTSTRAP_GUI") || return
    printf '%s\n%s\n' "$after" "$binary" >"$BOOTSTRAP_STAMP.part"
    mv -- "$BOOTSTRAP_STAMP.part" "$BOOTSTRAP_STAMP"
    printf 'Built %s\n' "$BOOTSTRAP_GUI"
}
