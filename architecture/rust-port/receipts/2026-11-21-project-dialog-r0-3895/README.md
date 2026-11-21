# Project Dialog evidence baseline

All eight pinned Project Dialog scenarios passed all 101 steps on Linux X11
under Xvfb with Vulkan Lavapipe. The native run covered the template chooser,
inactive-project inspection, wizard navigation and cancellation, workspace
folder selection, and keyboard control of settings and library groups.

The reviewed captures show the original active project after inspection
and the expanded Appearance Libraries group after Material Libraries
collapsed. The pinned source and the fixture notes define the expected
results. The [Go baseline](../2026-11-14-go-project-baseline/README.md)
covers project-file codec behavior.

- Scenario revision `b576445b98cffd686dcc05aa8c017de2a591c8c8`
- Feature SHA-256
  `c38e881942205983e27a3924f950a5732d2cae0e2cb09ae40a16b88e9f581f3d`
- Application source `c49b98b91f1e23fdf1e11c183b0f59b2a0e26cf6`
- Tested binary SHA-256
  `c499c87894d88dbe8bc3f3d4b2b7b7160fee7e2ba321f47900e11d07c51a5c67`
- API revision `98425811b462c18ec573dfdcc33f2c7b264a6e25`
- Martensite revision `bef25f47548431c8f128f08f66a1f719bcee4982`

The runner reused the desktop test binary from the branch's Sketch Text
palette check. It restored and checked the pinned source before and after the
run. The build and runtime inputs were unsealed. Strict native verification
remains open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).

To repeat the focused run on Linux with a system Lavapipe manifest at this
path, use:

```sh
env OBK_BDD_FEATURE=project-dialog XMODIFIERS=@im=none \
  VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  tools/with-xvfb.sh cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
