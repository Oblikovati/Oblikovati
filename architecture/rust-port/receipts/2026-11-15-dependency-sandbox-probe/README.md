# Scenario runtime isolation probe

On 2026-11-15, a native BDD test binary ran the pinned document-opening scenario
in a Linux Bubblewrap namespace. The project checkout was not mounted. All nine
scenario steps passed, and the capture shows the part active with the drawing
tab retained. The pinned checkout was clean before and after the run. The
binary SHA-256 was
`b3c2da306052842b32aa2bfd28749ba48366670290e96d441d1311dd2899f857`.

This is a feasibility probe, not a reusable dependency certificate. `/usr`,
`/lib` and `/etc` were exposed read-only without content hashes. The kernel,
`/proc`, `/dev`, environment and display inputs were not fully controlled.
A second run omitted `/proc`, cleared the inherited environment, and set only
`PATH`, `HOME`, `LANG`, the runtime and BDD variables. It also passed all nine
steps; `clean-env.log.gz` and `clean-env-cucumber.json` retain the output.
`/usr`, `/lib`, `/etc`, the kernel, `/dev`, and display inputs still need
identities or tighter confinement. The test used frame-loop input. Do not use
this result to mark a receipt current after application changes (#3974).

A third run used a 301 MB copied runtime root with 556 tracked entries, no
project checkout or `/proc`, and a cleared environment. It passed the same
nine steps. A content and metadata inventory of that root was identical
before and after; `runtime-inventory.json.gz` and `sealed-rerun.log.gz`
record the probe. This does not yet qualify the complete runtime or establish
that a changed application compiles to an equivalent runner.

At clean application commit `32b460500`, the new map generator inventoried
2,720 scenario declarations, 9,289 tracked Git inputs and the 556 runtime
entries. A comparison against the same commit and runtime reported no changes
but still refused receipt reuse because runtime coverage is unbounded.

```sh
bwrap --unshare-all --ro-bind /usr /usr --ro-bind /lib /lib \
  --ro-bind /etc /etc --symlink /usr/bin /bin --symlink /usr/sbin /sbin \
  --symlink /usr/lib64 /lib64 --proc /proc --dev /dev --tmpfs /tmp \
  --ro-bind "$TEST_BINARY" /runner --ro-bind "$SCENARIO_CHECKOUT" /scenario \
  --bind "$REPORTS" /reports --setenv HOME /tmp --setenv XDG_RUNTIME_DIR /tmp \
  --setenv VK_ICD_FILENAMES /usr/share/vulkan/icd.d/lvp_icd.json \
  --setenv OBK_BDD_ROOT /scenario --setenv OBK_BDD_REPORTS /reports \
  --setenv OBK_BDD_FEATURE document-opening \
  --setenv OBK_BDD_SCENARIO 'Activate an already open part through the Open dialog' \
  --chdir /tmp -- /usr/bin/xvfb-run -a \
  -s '-screen 0 1920x1080x24 -noreset' /runner \
  application_scenarios_use_the_native_window --nocapture
```
