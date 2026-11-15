# Extrusion preview diagnostic

- Application revision: `61321fbec`.
- Scenario revision: `48987af7b5cbb06515bc6f30385f508382f34064`.
- Environment: Linux, Xvfb, Vulkan Lavapipe and `XMODIFIERS=@im=none`.
- Result: the first two steps passed; the profile selection assertion failed.
  The fourth step was not reached. This is not an acceptance receipt.
- The panel reopened with zero profiles. The status reported four degrees of
  freedom where zero were required. The reviewed capture shows the panel and
  the unconstrained rectangle without a solid preview.
- The native harness uses frame-loop inputs. It does not establish OS-level
  mouse and keyboard delivery. See #3927 and #3928.

```sh
OBK_BDD_ROOT=/tmp/obk-quick-access-3955-source \
OBK_BDD_REPORTS=/tmp/obk-extrusion-02-diagnostic \
OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=extrusion \
OBK_BDD_SCENARIO=extrusion-02 CARGO_TARGET_DIR="$PWD/target" \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
