#!/usr/bin/env bash
set -euo pipefail
scratch_dir=$1
shift
display_number=117
if [[ -S /tmp/.X11-unix/X$display_number ]]; then
    echo "Private screen $display_number is already active." >&2
    exit 20
fi
auth_file="$scratch_dir/Xauthority"
touch "$auth_file"
xauth -f "$auth_file" add ":$display_number" MIT-MAGIC-COOKIE-1 "$(mcookie)" >/dev/null 2>&1
Xwayland ":$display_number" -geometry 1440x900 -auth "$auth_file" -nolisten tcp -noreset > "$scratch_dir/xwayland.log" 2>&1 &
xwayland_pid=$!
wm_pid=
cleanup() {
    if [[ -n $wm_pid ]]; then kill "$wm_pid" 2>/dev/null || true; fi
    kill "$xwayland_pid" 2>/dev/null || true
    wait "$xwayland_pid" 2>/dev/null || true
}
trap cleanup EXIT
export DISPLAY=":$display_number" XAUTHORITY="$auth_file" XMODIFIERS=@im=none
for attempt in $(seq 1 50); do
    xdpyinfo > "$scratch_dir/xdpyinfo.log" 2>&1 && break
    sleep 0.1
done
xdpyinfo -queryExtensions > "$scratch_dir/extensions.log"
grep -q 'DRI3' "$scratch_dir/extensions.log"
openbox > "$scratch_dir/openbox.log" 2>&1 &
wm_pid=$!
for attempt in $(seq 1 50); do
    xprop -root _NET_SUPPORTING_WM_CHECK | grep -q 'window id' && break
    sleep 0.1
done
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/radeon_icd.json
"$@"
